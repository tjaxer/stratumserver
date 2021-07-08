package server

import (
	"encoding/json"
)

const extraNonce1Size = 4

type Job struct {


//	return []interface{}{
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

	jobId string // ID of the job. Use this ID while submitting share generated from this job.
	prevHash string// []byte // Hash of previous block.
	coinb1 string //[]byte //Initial part of coinbase transaction.
	coinb2 string // []byte //Final part of coinbase transaction.
	merkleBranch  []string//[][]byte //List of hashes, will be used for calculation of merkle root. This is not a list of all transactions, it only contains prepared hashes of steps of merkle tree algorithm. Please read some materials for understanding how merkle trees calculation works. Unfortunately this example don't have any step hashes included, my bad!
	version  string//int32 //Bitcoin block version.
	nBits  string // uint32//Encoded current network difficulty
	nTime  string //uint32 //Current ntime/
	//clean_jobs - When true, server indicates that submitting shares from previous jobs don't have a sense and such shares will be rejected. When this flag is set, miner should also drop all previous jobs, so job_ids can be eventually rotated.

}

func newJob(jobId string,
	prevHash string,
	coinb1 string,
	coinb2 string,
	merkleBranch []string,
	version  string,
	nBits  string,
	nTime  string,
	) *Job {
	return &Job{
		jobId,
		prevHash,
		coinb1,
		coinb2,
		merkleBranch,
		version,
		nBits,
		nTime,
	}

}



func (job *Job) StratumJsonRequest(forceUpdate bool) *JsonRpcRequest{

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

	return &JsonRpcRequest{
		Id: nil,
		Method: "mining.notify",
		Params: params,
	}
}

