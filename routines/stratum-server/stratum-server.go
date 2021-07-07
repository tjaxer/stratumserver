package stratumserver

import (
	appctx "github.com/nixys/nxs-go-appctx/v2"
	"github.com/tjaxer/stratumserver/ctx"
	"github.com/tjaxer/stratumserver/stratum/server"
)



import (
"context"
"time"

)

type stratumServerContext struct {
	server.Server
	done chan interface{}
}

// Runtime executes the routine
func Runtime(cr context.Context, appCtx *appctx.AppContext, crc chan interface{}) {

	var err error

	s := servStart(appCtx)

	for {
		select {
		case <-cr.Done():
			// Program termination.
			err = servShutdown(s)
			if err != nil {
				appCtx.Log().Errorf("stratum server shutdown error: %v", err)
			}
			return
		case <-crc:
			// Updated context application data.
			// Set the new one in current goroutine.
			s, err = servRestart(appCtx, s)
			if err != nil {
				appCtx.Log().Errorf("stratum server reload error: %v", err)
				appCtx.RoutineDoneSend(appctx.ExitStatusFailure)
				return
			}
		}
	}
}

func servStart(appCtx *appctx.AppContext) *stratumServerContext {

	cc := appCtx.CustomCtx().(*ctx.Ctx)

	s := &stratumServerContext{
		Server: server.Server{
			Addr:         cc.Conf.Bind,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		done: make(chan interface{}),
	}

	go func() {
		appCtx.Log().Debugf("server status: starting")
		if cc.Conf.TLS.CertFile != "" && cc.Conf.TLS.KeyFie != "" {
			if err := s.ListenAndServeTLS(cc.Conf.TLS.CertFile, cc.Conf.TLS.KeyFie); err != nil {
				appCtx.Log().Debugf("stratum server status: %v", err)
			}
		} else {
			if err := s.ListenAndServe(); err != nil {
				appCtx.Log().Debugf("stratum server status: %v", err)
			}
		}
		s.done <- true
	}()

	return s
}

func servShutdown(s *stratumServerContext) error {

	//Create shutdown context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//Shutdown the server
	if err := s.Shutdown(ctx); err != nil {
		return err
	}

	<-s.done
	return nil
}

func servRestart(appCtx *appctx.AppContext, s *stratumServerContext) (*stratumServerContext, error) {

	if err := servShutdown(s); err != nil {
		return nil, err
	}

	return servStart(appCtx), nil
}
