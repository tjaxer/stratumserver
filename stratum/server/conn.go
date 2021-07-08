package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"github.com/rs/zerolog"
	"github.com/tjaxer/stratumserver/logger"
	"io"
	"net"
	"runtime"
	"sync/atomic"
	"time"
)

// A conn represents the server side of an HTTP connection.
type conn struct {
	// server is the server on which the connection arrived.
	// Immutable; never nil.
	server *Server

	// cancelCtx cancels the connection-level context.
	cancelCtx context.CancelFunc

	// rwc is the underlying network connection.
	// This is never wrapped by other types and is the value given out
	// to CloseNotifier callers. It is usually of type *net.TCPConn or
	// *tls.Conn.
	rwc net.Conn

	// remoteAddr is rwc.RemoteAddr().String(). It is not populated synchronously
	// inside the Listener's Accept goroutine, as some implementations block.
	// It is populated immediately inside the (*conn).serve goroutine.
	// This is the value of a Handler's (*Request).RemoteAddr.
	remoteAddr string

	// tlsState is the TLS connection state when using TLS.
	// nil means not TLS.
	tlsState *tls.ConnectionState

	// werr is set to the first write error to rwc.
	// It is set via checkConnErrorWriter{w}, where bufw writes.
	werr error

	// r is bufr's read source. It's a wrapper around rwc that provides
	// io.LimitedReader-style limiting (while reading request headers)
	// and functionality to support CloseNotifier. See *connReader docs.
	r *connReader

	// bufr reads from r.
	bufr *bufio.Reader

	// bufw writes to checkConnErrorWriter{c}, which populates werr on error.
	bufw *bufio.Writer

	// lastMethod is the method of the most recent request
	// on this connection, if any.
	lastMethod string

	curReq atomic.Value // of *response (which has a Request in it)

	curState struct{ atomic uint64 } // packed (unixtime<<8|uint8(ConnState))

}

// Close the connection.
func (c *conn) Close() error {
	c.finalFlush()
	return c.rwc.Close()
}

func (c *conn) finalFlush() {
	if c.bufr != nil {
		// Steal the bufio.Reader (~4KB worth of memory) and its associated
		// reader for a future connection.
		putBufioReader(c.bufr)
		c.bufr = nil
	}

	if c.bufw != nil {
		c.bufw.Flush()
		// Steal the bufio.Writer (~4KB worth of memory) and its associated
		// writer for a future connection.
		putBufioWriter(c.bufw)
		c.bufw = nil
	}
}

// rstAvoidanceDelay is the amount of time we sleep after closing the
// write side of a TCP connection before closing the entire socket.
// By sleeping, we increase the chances that the client sees our FIN
// and processes its final data before they process the subsequent RST
// from closing a connection with known unread data.
// This RST seems to occur mostly on BSD systems. (And Windows?)
// This timeout is somewhat arbitrary (~latency around the planet).
const rstAvoidanceDelay = 500 * time.Millisecond

type closeWriter interface {
	CloseWrite() error
}

var _ closeWriter = (*net.TCPConn)(nil)

// closeWrite flushes any outstanding data and sends a FIN packet (if
// client is connected via TCP), signalling that we're done. We then
// pause for a bit, hoping the client processes it before any
// subsequent RST.
//
// See https://golang.org/issue/3595
func (c *conn) closeWriteAndWait() {
	c.finalFlush()
	if tcp, ok := c.rwc.(closeWriter); ok {
		tcp.CloseWrite()
	}
	time.Sleep(rstAvoidanceDelay)
}

func (c *conn) SendJsonRPC(jsonRPCs JsonRpc) error {
	raw := jsonRPCs.Json()

	message := make([]byte, 0, len(raw)+1)
	message = append(raw, '\n')
	_, err := c.bufw.Write(message)
	if err != nil {
		c.logErr(err).Str("raw data", string(raw)).Msg("failed inputting")
		return err
	}

	err = c.bufw.Flush()
	if err != nil {
		c.logErr(err).Msg("failed sending data to miner")
		return err
	}
	c.logDebug().Str("raw data", string(raw)).Msg("SendJsonRPC")
	return nil
}

func (c *conn) logTrace() *zerolog.Event {
	return logger.Log.Trace().Str("component", "stratum connection")
}

func (c *conn) logDebug() *zerolog.Event {
	return logger.Log.Debug().Str("component", "stratum connection")
}

func (c *conn) logInfo() *zerolog.Event {
	return logger.Log.Info().Str("component", "stratum connection")
}

func (c *conn) logWarn() *zerolog.Event {
	return logger.Log.Warn().Str("component", "stratum connection")
}

func (c *conn) logErr(err error) *zerolog.Event {
	return logger.Log.Err(err).Str("component", "stratum connection")
}

//var errTooLarge = errors.New("http: request too large")

// Read next request from connection.
func (c *conn) readMessage(ctx context.Context) (message *JsonRpcRequest, err error) {

	var (
		//wholeReqDeadline time.Time // or zero if none
		hdrDeadline time.Time // or zero if none
	)
	t0 := time.Now()
	if d := c.server.readHeaderTimeout(); d != 0 {
		hdrDeadline = t0.Add(d)
	}
	if d := c.server.ReadTimeout; d != 0 {
		//wholeReqDeadline = t0.Add(d)
	}
	c.rwc.SetReadDeadline(hdrDeadline)
	if d := c.server.WriteTimeout; d != 0 {
		defer func() {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}()
	}

	c.r.setReadLimit(c.server.initialReadLimitSize())
	raw, err := c.bufr.ReadBytes('\n')
	if err != nil {
		c.logErr(err).Msg("Read line")

		return nil, err
	}
	c.logDebug().Str("data", hex.EncodeToString(raw)).Msg("Read line")
	err = json.Unmarshal(raw, &message)
	if err != nil {
		c.logErr(err).Str("raw data", string(raw)).Msg("Malformed message from")
		c.closeWriteAndWait()
		//_ = sc.Socket.Close()
		//sc.SocketClosedEvent <- struct{}{}

		return nil, err
	}

	//req, err := readRequest(c.bufr, keepHostHeader)
	//if err != nil {
	//	if c.r.hitReadLimit() {
	//		return nil, errTooLarge
	//	}
	//	return nil, err
	//}

	//if !http1ServerSupportsRequest(req) {
	//	return nil, badRequestError("unsupported protocol version")
	//}
	//
	//c.lastMethod = req.Method
	//c.r.setInfiniteReadLimit()
	//
	//hosts, haveHost := req.Header["Host"]
	//isH2Upgrade := req.isH2Upgrade()
	//if req.ProtoAtLeast(1, 1) && (!haveHost || len(hosts) == 0) && !isH2Upgrade && req.Method != "CONNECT" {
	//	return nil, badRequestError("missing required Host header")
	//}
	//if len(hosts) > 1 {
	//	return nil, badRequestError("too many Host headers")
	//}
	//if len(hosts) == 1 && !httpguts.ValidHostHeader(hosts[0]) {
	//	return nil, badRequestError("malformed Host header")
	//}
	//for k, vv := range req.Header {
	//	if !httpguts.ValidHeaderFieldName(k) {
	//		return nil, badRequestError("invalid header name")
	//	}
	//	for _, v := range vv {
	//		if !httpguts.ValidHeaderFieldValue(v) {
	//			return nil, badRequestError("invalid header value")
	//		}
	//	}
	//}
	//delete(req.Header, "Host")
	//
	//ctx, cancelCtx := context.WithCancel(ctx)
	//req.ctx = ctx
	//req.RemoteAddr = c.remoteAddr
	//req.TLS = c.tlsState
	//if body, ok := req.Body.(*body); ok {
	//	body.doEarlyClose = true
	//}
	//
	//// Adjust the read deadline if necessary.
	//if !hdrDeadline.Equal(wholeReqDeadline) {
	//	c.rwc.SetReadDeadline(wholeReqDeadline)
	//}
	//
	//w = &response{
	//	conn:          c,
	//	cancelCtx:     cancelCtx,
	//	req:           req,
	//	reqBody:       req.Body,
	//	handlerHeader: make(Header),
	//	contentLength: -1,
	//	closeNotifyCh: make(chan bool, 1),
	//
	//	// We populate these ahead of time so we're not
	//	// reading from req.Header after their Handler starts
	//	// and maybe mutates it (Issue 14940)
	//	wants10KeepAlive: req.wantsHttp10KeepAlive(),
	//	wantsClose:       req.wantsClose(),
	//}
	//if isH2Upgrade {
	//	w.closeAfterReply = true
	//}
	//w.cw.res = w
	//w.w = newBufioWriterSize(&w.cw, bufferBeforeChunkingSize)
	return message, nil
}

// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
	c.remoteAddr = c.rwc.RemoteAddr().String()
	ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logErr(err.(error)).Bytes("stack", buf).Str("serving", c.remoteAddr).Msg("panic")
		}
		c.Close()
		c.setState(c.rwc, StateClosed)
	}()

	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
		if d := c.server.ReadTimeout; d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
		}
		if d := c.server.WriteTimeout; d != 0 {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			// If the handshake failed due to the client not speaking
			// TLS, assume they're speaking plaintext HTTP and write a
			// 400 response on the TLS conn's underlying net.Conn.
			tlsConn.Close()
			c.server.logErr(err).Str("address", c.rwc.RemoteAddr().String()).Msg("http: TLS handshake error")
			return
		}
		c.tlsState = new(tls.ConnectionState)
		*c.tlsState = tlsConn.ConnectionState()

		// ToDo: Implement TLS over tcp
		//if proto := c.tlsState.NegotiatedProtocol; validNextProto(proto) {
		//	if fn := c.server.TLSNextProto[proto]; fn != nil {
		//		h := initALPNRequest{ctx, tlsConn, serverHandler{c.server}}
		//		fn(c.server, tlsConn, h)
		//	}
		//	return
		//}
	}

	ctx, cancelCtx := context.WithCancel(ctx)
	c.cancelCtx = cancelCtx
	defer cancelCtx()

	c.r = &connReader{conn: c}
	c.bufr = newBufioReader(c.r)
	c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)
	c.logDebug().Msg("connected")

	// job_rebroadcast_timeout: 600
	//  tcp_proxy_protocol: false
	//  connection_timeout: 600
	//  ports:
	//    port: 3032
	//    diff: 5
	//    var_diff:
	//      min_diff: 1
	//      max_diff: 100000
	//      target_time: 15
	//      retarget_time: 90
	//      variance_percent: 30
	//    tls: false
	worker := NewStratumClient(
		5,      // InitialDifficulty float64,
		600,    // ConnectionTimeout int,
		false,  // TcpProxyProtocol bool,
		1,      // MinDiff         float64,
		100000, // MaxDiff        float64,
		15,     // TargetTime      int64,
		90,     // RetargetTime    int64,
		30,     // VariancePercent float64,
		false,  // X2Mode          bool,

	)

	for {

		message, err := c.readMessage(ctx)
		if c.r.remain != c.server.initialReadLimitSize() {
			// If we read any bytes off the wire, we're active.
			c.setState(c.rwc, StateActive)
		}

		//select {
		//case <-sc.SocketClosedEvent:
		//	return
		//default:

		if err != nil {
			c.logErr(err).Msg("read error")
			if err == io.EOF {
				//sc.SocketClosedEvent <- struct{}{}
				return
			}
			e, ok := err.(net.Error)

			if !ok {
				c.logErr(err).Msg("failed to ready bytes from socket due to non-network error:")
				return
			}

			if ok && e.Timeout() {
				c.logErr(err).Msg("socket is timeout")
				return
			}

			if ok && e.Temporary() {
				c.logErr(err).Msg("failed to ready bytes from socket due to temporary error")
				continue
			}

			c.logErr(err).Msg("failed to ready bytes from socket:")
			return
		}
		// ToDo: Flool protection
		// TODO: remove hardcode
		//if len(raw) > 10240 {
		//	// socketFlooded
		//	c.logWarn().Str("from", c.remoteAddr).Str("raw data", string(raw)).Msg("Flooding message")
		//	//_ = c.Close()
		//	c.setState(c.rwc, StateClosed)
		//	c.closeWriteAndWait()
		//	//sc.SocketClosedEvent <- struct{}{}
		//	return
		//}
		//
		//if len(raw) == 0 {
		//	continue
		//}

		//sc.BanningManager.CheckBan(sc.RemoteAddress.String())

		if &message != nil {
			logger.Log.Debug().Str("client message", string(message.Json())).Msg("handling message")
			var resp *JsonRpcResponse
			switch message.Method {
			case "mining.subscribe":
				resp, err = worker.HandleSubscribe(message)
			case "mining.authorize":
				resp, err = worker.HandleAuthorize(message, c.remoteAddr)
				//case "mining.submit":
				//	sc.LastActivity = time.Now()
				//	sc.HandleSubmit(message)
			default:
				logger.Log.Warn().Str("unknown stratum method", string(Jsonify(message))).Send()
			}

			if err != nil {
				c.logErr(err)
				return
			}
			if resp != nil {
				if err := c.SendJsonRPC(resp); err != nil {
					c.logErr(err)
					return
				}
			}
		}
		//}

	}

	//for {
	//	w, err := c.readRequest(ctx)
	//	if c.r.remain != c.server.initialReadLimitSize() {
	//		// If we read any bytes off the wire, we're active.
	//		c.setState(c.rwc, StateActive)
	//	}
	//	if err != nil {
	//		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
	//
	//		switch {
	//		case err == errTooLarge:
	//			// Their HTTP client may or may not be
	//			// able to read this if we're
	//			// responding to them and hanging up
	//			// while they're still writing their
	//			// request. Undefined behavior.
	//			const publicErr = "431 Request Header Fields Too Large"
	//			fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
	//			c.closeWriteAndWait()
	//			return
	//
	//		case isUnsupportedTEError(err):
	//			// Respond as per RFC 7230 Section 3.3.1 which says,
	//			//      A server that receives a request message with a
	//			//      transfer coding it does not understand SHOULD
	//			//      respond with 501 (Unimplemented).
	//			code := StatusNotImplemented
	//
	//			// We purposefully aren't echoing back the transfer-encoding's value,
	//			// so as to mitigate the risk of cross side scripting by an attacker.
	//			fmt.Fprintf(c.rwc, "HTTP/1.1 %d %s%sUnsupported transfer encoding", code, StatusText(code), errorHeaders)
	//			return
	//
	//		case isCommonNetReadError(err):
	//			return // don't reply
	//
	//		default:
	//			publicErr := "400 Bad Request"
	//			if v, ok := err.(badRequestError); ok {
	//				publicErr = publicErr + ": " + string(v)
	//			}
	//
	//			fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
	//			return
	//		}
	//	}
	//
	//	// Expect 100 Continue support
	//	req := w.req
	//	if req.expectsContinue() {
	//		if req.ProtoAtLeast(1, 1) && req.ContentLength != 0 {
	//			// Wrap the Body reader with one that replies on the connection
	//			req.Body = &expectContinueReader{readCloser: req.Body, resp: w}
	//			w.canWriteContinue.setTrue()
	//		}
	//	} else if req.Header.get("Expect") != "" {
	//		w.sendExpectationFailed()
	//		return
	//	}
	//
	//	c.curReq.Store(w)
	//
	//	if requestBodyRemains(req.Body) {
	//		registerOnHitEOF(req.Body, w.conn.r.startBackgroundRead)
	//	} else {
	//		w.conn.r.startBackgroundRead()
	//	}
	//
	//	// HTTP cannot have multiple simultaneous active requests.[*]
	//	// Until the server replies to this request, it can't read another,
	//	// so we might as well run the handler in this goroutine.
	//	// [*] Not strictly true: HTTP pipelining. We could let them all process
	//	// in parallel even if their responses need to be serialized.
	//	// But we're not going to implement HTTP pipelining because it
	//	// was never deployed in the wild and the answer is HTTP/2.
	//	serverHandler{c.server}.ServeHTTP(w, w.req)
	//	w.cancelCtx()
	//	if c.hijacked() {
	//		return
	//	}
	//	w.finishRequest()
	//	if !w.shouldReuseConnection() {
	//		if w.requestBodyLimitHit || w.closedRequestBodyEarly() {
	//			c.closeWriteAndWait()
	//		}
	//		return
	//	}
	//	c.setState(c.rwc, StateIdle)
	//	c.curReq.Store((*response)(nil))
	//
	//	if !w.conn.server.doKeepAlives() {
	//		// We're in shutdown mode. We might've replied
	//		// to the user without "Connection: close" and
	//		// they might think they can send another
	//		// request, but such is life with HTTP/1.1.
	//		return
	//	}
	//
	//	if d := c.server.idleTimeout(); d != 0 {
	//		c.rwc.SetReadDeadline(time.Now().Add(d))
	//		if _, err := c.bufr.Peek(4); err != nil {
	//			return
	//		}
	//	}
	//	c.rwc.SetReadDeadline(time.Time{})
	//}
}
