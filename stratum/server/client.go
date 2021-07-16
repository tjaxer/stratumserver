package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	btcdutil "github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tjaxer/stratumserver/logger"
	"github.com/tjaxer/stratumserver/stratum/server/vardiff"
	"github.com/tjaxer/stratumserver/stratum/server2/utils"
	//"github.com/jasonlvhit/gocron"

	"github.com/tjaxer/stratumserver/miner"
	//"github.com/tjaxer/stratumserver/stratum/server2/utils"
	"github.com/tjaxer/stratumserver/stratum/types"
	"math/big"
	//"net"

	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const ExtraNonce2Size=4

type WorkerConnection struct {
	SubscriptionId         string
	LastActivity           time.Time
	Shares                 *Shares
	SubscriptionBeforeAuth bool
	ExtraNonce1            []byte
	VarDiff                *vardiff.VarDiff
	worker                 uint64

	// auth info
	WorkerName   string
	WorkerPass   string
	IsAuthorized bool

	Subscribed bool

	InitialDifficulty  float64
	PendingDifficulty  *big.Float
	CurrentDifficulty  *big.Float
	PreviousDifficulty *big.Float
	ConnectionTimeout  int
	TcpProxyProtocol   bool
	SocketClosedEvent  chan struct{}
	id                 int
	job                atomic.Value
}

func (miner *WorkerConnection) setJob(job *Job) {
	miner.job.Store(job)
}

func NewStratumClient(

	InitialDifficulty float64,
	ConnectionTimeout int,
	TcpProxyProtocol bool,
	MinDiff float64,
	MaxDiff float64,
	TargetTime int64,
	RetargetTime int64,
	VariancePercent float64,
	X2Mode bool,
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
	//_, _ = rand.Read(extraNonce1)
	extraNonce1,_  = hex.DecodeString("00000000")
	return &WorkerConnection{
		SubscriptionId:         uuid.New().String(),
		PendingDifficulty:      big.NewFloat(0),
		LastActivity:           time.Now(),
		SubscriptionBeforeAuth: false,
		VarDiff:                varDiff,
		InitialDifficulty:      InitialDifficulty,
		ConnectionTimeout:      ConnectionTimeout,
		TcpProxyProtocol:       TcpProxyProtocol,
		Shares:                 new(Shares),
		//ExtraNonce1:            jm.ExtraNonce1Generator.GetExtraNonce1(),
		ExtraNonce1: extraNonce1,
	}
}

//WorkerConnection
func (c *WorkerConnection) difficulyJsonMessage(difficulty *big.Float) (*JsonRpcRequest, error) {
	text, err := difficulty.MarshalText()
	if err != nil {
		return nil, err
	}

	return &JsonRpcRequest{
		Id:     0,
		Method: "mining.set_difficulty",
		//Params: []json.RawMessage{Jsonify(difficulty)},
		Params: []json.RawMessage{text},
	}, nil
}


func (c *WorkerConnection) jobJsonMessage(job *Job, forceUpdate bool) (*JsonRpcRequest, error) {

	//return []interface{}{
	//	job.JobId,
	//	prevHashReversed,
	//	hex.EncodeToString(parts[0]),
	//	hex.EncodeToString(parts[1]),
	//	merkleBranch,
	//	hex.EncodeToString(utils.PackInt32BE(job.MinerTask.BitcoinBlockVersion)),
	//	hex.EncodeToString(utils.PackUint32BE(job.MinerTask.BitcoinBlockBits)),
	//	hex.EncodeToString(utils.PackUint32BE(uint32(time.Now().Unix()))), // Updated: implement time rolling
	//	forceUpdate,
	//}

	params:= make([]json.RawMessage, 9)
	params[0] = Jsonify(job.jobId)
	params[1] = Jsonify(job.prevHash)
	params[2] = Jsonify(job.coinb1)
	params[3] = Jsonify(job.coinb2)
	params[4] = Jsonify(job.merkleBranch)
	params[5] = Jsonify(job.version)
	params[6] = Jsonify(job.nBits)
	params[7] = Jsonify(job.nTime)
	params[8] = Jsonify(forceUpdate)

	ret:=&JsonRpcRequest{
		Id:     3,
		Method: "mining.notify",
		Params: params,
	}

	return ret,nil
}

func (c *WorkerConnection) HandleMessage(message *JsonRpcRequest, connection conn) error {
	switch message.Method {
	case "mining.subscribe":

		resp, err := c.HandleSubscribe(message)
		if err != nil {
			return err
		}
		err = connection.SendJsonRPC(resp)
		if err != nil {
			return err
		}
		c.Subscribed = true
	case "mining.authorize":
		c.logInfo().Msg("----- here is it 1  -----")
		resp, err := c.HandleAuthorize(message, connection.remoteAddr)
		if err != nil {
			c.logInfo().Msg("----- here is error  -----")
			return err
		}
		err = connection.SendJsonRPC(resp)
		if err != nil {
			c.logInfo().Msg("----- here is error  2 -----")
			return err
		}
		c.logInfo().Msg("----- here is it 2  -----")
		msg, err := c.difficulyJsonMessage(big.NewFloat(c.InitialDifficulty))
		if err != nil {
			return err
		}
		err = connection.SendJsonRPC(msg)
		if err != nil {
			return err
		}




		btcBlock := miner.Block100000
		merkleBranch := make([]string, 0)
		transactions:=btcdutil.NewBlock(&btcBlock).Transactions()
		merkles := blockchain.BuildMerkleTreeStore(transactions[1:len(transactions)], false)
		merklePath := utils.MerkleTreeBranch(merkles)

		for _, merkle := range merklePath {
			if merkle != nil {
				merkleBranch = append(merkleBranch, merkle.String())
			}
		}


		part1, part2:= utils.SplitCoinbase(transactions[0].MsgTx())
		job:=newJob("XXX-1",  // jobId string
			hex.EncodeToString(
				utils.ReverseByteOrder(
					btcBlock.Header.PrevBlock.CloneBytes())), // prevHash string (reverced)
			hex.EncodeToString(part1),//coinb1 string,
			hex.EncodeToString(part2),//coinb2 string,
			merkleBranch, //merkleBranch []string,
			hex.EncodeToString(utils.PackInt32BE(btcBlock.Header.Version)), //version string,
			hex.EncodeToString(utils.PackUint32BE(486604799)), //nBits string,
			//hex.EncodeToString(utils.PackUint32BE(btcBlock.Header.Bits)), //nBits string,
			hex.EncodeToString(utils.PackUint32BE(uint32(time.Now().Unix())))) //nTime string

		// send new job
		msg, err = c.jobJsonMessage( job,true)
		if err != nil {
			return err
		}
		c.logInfo().Msg("----- here is it -----")
		err = connection.SendJsonRPC(msg)
		if err != nil {
			return err
		}

	// TODO: fix
	// sc.SendMiningJob(sc.JobManager.CurrentJob.GetJobParams(true))
	case "mining.submit":
		//	sc.LastActivity = time.Now()
		msg, err := c.HandleSubmit(message, connection.remoteAddr)
		if err != nil {
			return err
		}
		err = connection.SendJsonRPC(msg)
		if err != nil {
			return err
		}
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
		return false, fmt.Errorf("%s", types.ErrMinerInvalidFormat)
	}

	// Rig is a string with [a-zA-Z0-9]+ regexp format.
	r, _ := regexp.Compile("[a-zA-Z0-9]+")
	if !r.MatchString(rig) {
		return false, fmt.Errorf("%s", types.ErrRigInvalidFormat)
	}

	return true, nil
}

func (c *WorkerConnection) HandleAuthorize(message *JsonRpcRequest, remoteAddress string) (*JsonRpcResponse, error) {
	c.logInfo().Msg("handling authorize")

	c.WorkerName = string(message.Params[0])
	c.WorkerPass = string(message.Params[1])

	authorized, err := c.AuthorizeFn(remoteAddress, c.WorkerName, c.WorkerPass)
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

	// ToDo: Session resume

	resp := &JsonRpcResponse{
		Id: message.Id,
		Result: Jsonify([]interface{}{
			[][]string{
				{"mining.notify", c.SubscriptionId},
			},
			hex.EncodeToString(c.ExtraNonce1),
			ExtraNonce2Size,
		}),
		Error: nil,
	}
	return resp, nil
}

func (c *WorkerConnection) HandleSubmit(message *JsonRpcRequest, remoteAddress string) (*JsonRpcResponse, error) {
//mining.submit("username", "job id", "ExtraNonce2", "nTime", "nOnce")
	username, _ := message.Params[0].MarshalJSON()
	jobId, _ := message.Params[1].MarshalJSON()
	ExtraNonce2, _ := message.Params[2].MarshalJSON()
	nTime, _ := message.Params[3].MarshalJSON()
	nOnce, _ := message.Params[4].MarshalJSON()


	c.logInfo().
		Str("method", string(message.Method) ).
		Str("username",string(username)).
		Str("jobId",string(jobId)).
		Str("ExtraNonce2",string(ExtraNonce2)).
		Str("nTime",string(nTime)).
		Str("nOnce",string(nOnce)).
		Msg("handling submit")


	//share, minerResult := sc.JobManager.ProcessSubmit(
	//	RawJsonToString(message.Params[1]),
	//	sc.PreviousDifficulty,
	//	sc.CurrentDifficulty,
	//	sc.ExtraNonce1,
	//	RawJsonToString(message.Params[2]),
	//	RawJsonToString(message.Params[3]),
	//	RawJsonToString(message.Params[4]),
	//	sc.RemoteAddress,
	//	RawJsonToString(message.Params[0]),
	//)

	//func (jm *JobManager) ProcessSubmit(jobId string, prevDiff, diff *big.Float, extraNonce1 []byte,
	//	hexExtraNonce2, hexNTime, hexNonce string, ipAddr net.Addr, workerName string) (share *Share, minerResult *tasks.MinerResult) {
	//	minerResult = new(tasks.MinerResult)


	return nil, nil
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
