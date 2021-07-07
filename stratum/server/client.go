package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tjaxer/stratumserver/logger"
	"github.com/tjaxer/stratumserver/stratum/server/vardiff"
	"github.com/tjaxer/stratumserver/stratum/types"
	"math/big"

	"regexp"
	"strings"
	"sync/atomic"
	"time"
	"crypto/rand"
)





type WorkerConnection struct {
	SubscriptionId string
	LastActivity time.Time
	Shares       *Shares
	SubscriptionBeforeAuth bool
	ExtraNonce1 []byte
	VarDiff *vardiff.VarDiff
	worker uint64

	// auth info
	WorkerName string
	WorkerPass string
	IsAuthorized           bool

	Subscribed bool

	InitialDifficulty  float64
	PendingDifficulty  *big.Float
	CurrentDifficulty  *big.Float
	PreviousDifficulty *big.Float
	ConnectionTimeout int
	TcpProxyProtocol bool
	SocketClosedEvent chan struct{}
	id int
	job atomic.Value
}



func (miner *WorkerConnection) newJob(job *Job) {
	miner.job.Store(job)
}


func NewStratumClient(

	InitialDifficulty float64,
	ConnectionTimeout int,
	TcpProxyProtocol bool,
	MinDiff         float64,
	MaxDiff        float64,
	TargetTime      int64,
	RetargetTime    int64,
	VariancePercent float64,
	X2Mode          bool,
	) *WorkerConnection {

	var varDiff *vardiff.VarDiff

	varDiff = vardiff.NewVarDiff(MinDiff,
		MaxDiff,
		TargetTime,
		RetargetTime,
		VariancePercent,
		X2Mode)

	// create random extranonce for worker
	extraNonce1 := make([]byte, extraNonce1Size)
	_, _ = rand.Read(extraNonce1)

	return &WorkerConnection{
		SubscriptionId: uuid.New().String(),
		PendingDifficulty:      big.NewFloat(0),
		LastActivity:           time.Now(),
		SubscriptionBeforeAuth: false,
		VarDiff:                varDiff,
		InitialDifficulty:		InitialDifficulty,
		ConnectionTimeout:      ConnectionTimeout,
		TcpProxyProtocol:       TcpProxyProtocol,
		Shares:                 new(Shares),
		//ExtraNonce1:            jm.ExtraNonce1Generator.GetExtraNonce1(),
		ExtraNonce1: extraNonce1,
	}
}


//WorkerConnection
func (c *WorkerConnection) difficulyJsonMessage(difficulty *big.Float) (*JsonRpcRequest,error) {
	text, err := difficulty.MarshalText()
	if err!=nil {
		return nil, err
	}

	return &JsonRpcRequest{
		Id:     0,
		Method: "mining.set_difficulty",
		//Params: []json.RawMessage{Jsonify(difficulty)},
		Params: []json.RawMessage{text},
	}, nil
}

func (c *WorkerConnection) HandleMessage(message *JsonRpcRequest, connection conn) error {
	switch message.Method {
	case "mining.subscribe":

		resp,err := c.HandleSubscribe(message)
		if err!=nil {
			return err
		}
		err = connection.SendJsonRPC(resp)
		if err!=nil {
			return err
		}
		c.Subscribed = true
	case "mining.authorize":
		resp,err := c.HandleAuthorize(message, connection.remoteAddr)
		if err!=nil {
			return err
		}
		err = connection.SendJsonRPC(resp)
		if err!=nil {
			return err
		}
		// the init Diff for miners
		c.logInfo().Float64("difficulty", c.InitialDifficulty).Msg("initial difficulty")
		msg,err:=c.difficulyJsonMessage(big.NewFloat(c.InitialDifficulty))
		if err!=nil {
			return err
		}
		err = connection.SendJsonRPC(msg)
		if err!=nil {
			return err
		}

	// TODO: fix
	// sc.SendMiningJob(sc.JobManager.CurrentJob.GetJobParams(true))
	//case "mining.submit":
	//	sc.LastActivity = time.Now()
	//	sc.HandleSubmit(message)
	default:
		logger.Log.Warn().Str("unknown stratum method", string(Jsonify(message))).Send()
	}
	return nil
}


// AuthorizeFn validates worker name and rig name.
func (c *WorkerConnection) AuthorizeFn(remoteAddress string, workerName string, password string) (authorized bool, err error) {
	logger.Log.Info().Str("workerName", workerName).Str("password", password).Str("ipAddr", remoteAddress).Msg("Authorize worker")

	var miner, rig string
	workerName = strings.TrimPrefix(workerName, `"`)
	workerName = strings.TrimSuffix(workerName, `"`)
	names := strings.Split(workerName, ".")
	if len(names) < 2 {
		miner = names[0]
		rig = "unknown"
	} else {
		miner = names[0]
		rig = names[1]
	}

	// JMP-64. Miner/rig validation.
	// Miner should be in UUID v4 format.
	_, err = uuid.Parse(miner)
	if err != nil {
		return false,  fmt.Errorf("%s", types.ErrMinerInvalidFormat)
	}

	// Rig is a string with [a-zA-Z0-9]+ regexp format.
	r, _ := regexp.Compile("[a-zA-Z0-9]+")
	if !r.MatchString(rig) {
		return false, fmt.Errorf("%s", types.ErrRigInvalidFormat)
	}

	return true,  nil
}


func (c *WorkerConnection) HandleAuthorize(message *JsonRpcRequest, remoteAddress string) (*JsonRpcResponse, error) {
	c.logInfo().Msg("handling authorize")

	c.WorkerName = string(message.Params[0])
	c.WorkerPass = string(message.Params[1])

	authorized,  err := c.AuthorizeFn(remoteAddress, c.WorkerName, c.WorkerPass)
	c.IsAuthorized = err == nil && authorized
	logger.Log.Info().Str("c.WorkerName", c.WorkerName).Bool("sc.IsAuthorized", c.IsAuthorized).Msg("worker authorized")

		if c.IsAuthorized {
			return &JsonRpcResponse{
				Id:     message.Id,
				Result: Jsonify(c.IsAuthorized),
				Error:  nil,
			}, nil
		}
		return &JsonRpcResponse{
				Id:     message.Id,
				Result: Jsonify(c.IsAuthorized),
				Error: &JsonRpcError{
					Code:    20,
					Message: string(Jsonify(err)),
				},
			}, err

}



func (c *WorkerConnection) HandleSubscribe(message *JsonRpcRequest) (*JsonRpcResponse, error) {
	c.logInfo().Msg("handling subscribe")
	//if !c.IsAuthorized {
	//	c.SubscriptionBeforeAuth = true
	//}

	//extraNonce2Size := sc.JobManager.ExtraNonce2Size


	resp:= &JsonRpcResponse{
		Id: message.Id,
		Result: Jsonify([]interface{}{
			[][]string{
				{"mining.set_difficulty", c.SubscriptionId },
				{"mining.notify", c.SubscriptionId},
			},
			hex.EncodeToString(c.ExtraNonce1),
			len(c.ExtraNonce1),
		}),
		Error: nil,
	}
	return resp,nil
}


func (c *WorkerConnection) logTrace() *zerolog.Event {
	return logger.Log.Trace().Str("component", "WorkerConnection")
}

func (c *WorkerConnection) logDebug() *zerolog.Event {
	return logger.Log.Debug().Str("component", "WorkerConnection")
}


func (c *WorkerConnection) logInfo() *zerolog.Event {
	return logger.Log.Info().Str("component", "WorkerConnection")
}

func (c *WorkerConnection) logWarn() *zerolog.Event {
	return logger.Log.Warn().Str("component", "WorkerConnection")
}


func (c *WorkerConnection) logErr(err error) *zerolog.Event {
	return logger.Log.Err(err).Str("component", "WorkerConnection")
}
