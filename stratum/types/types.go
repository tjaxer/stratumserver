package types

type ErrorWrap int

const (
	ErrJobNotFound              ErrorWrap = 20
	ErrIncorrectExtraNonce2Size ErrorWrap = 21
	ErrIncorrectNTimeSize       ErrorWrap = 22
	ErrNTimeOutOfRange          ErrorWrap = 23
	ErrIncorrectNonceSize       ErrorWrap = 24
	ErrDuplicateShare           ErrorWrap = 25
	ErrLowDiffShare             ErrorWrap = 26
	ErrMinerInvalidFormat       ErrorWrap = 27
	ErrRigInvalidFormat         ErrorWrap = 28
)

var codeToErrMap = map[int]string{
	10: "you are banned by pool",
	20: "job not found",
	21: "incorrect size of extranonce2",
	22: "incorrect size of ntime",
	23: "ntime out of range",
	24: "incorrect size of nonce",
	25: "duplicate share",
	26: "low difficulty share",
	27: "miner should be in UUID v4 format",
	28: "rig should be in [a-zA-Z0-9]+ format",
}

func (err ErrorWrap) String() string {
	return codeToErrMap[int(err)]
}
