package server

import (
	"encoding/json"
	"github.com/tjaxer/stratumserver/logger"
)

type JsonRpc interface {
	GetJsonRpcId() int64
	Json() []byte
}

type JsonRpcResponse struct {
	Id     interface{}     `json:"id"` // be int64 or null
	Result json.RawMessage `json:"result,omitempty"`
	Error  *JsonRpcError   `json:"error,omitempty"`
}

func (j *JsonRpcResponse) GetJsonRpcId() int64 {
	if j.Id == nil {
		return 0
	}

	return j.Id.(int64)
}

func (j *JsonRpcResponse) Json() []byte {
	raw, _ := json.Marshal(j)
	return raw
}

type JsonRpcRequest struct {
	Id     interface{}       `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func (j *JsonRpcRequest) GetJsonRpcId() int64 {
	if j.Id == nil {
		return 0
	}

	return j.Id.(int64)
}

func (j *JsonRpcRequest) Json() []byte {
	raw, _ := json.Marshal(j)
	return raw
}

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TODO: Copied from utils. Probably move to separate file in this package.

// Jsonify encodes i to []byte
func Jsonify(i interface{}) []byte {
	r, err := json.Marshal(i)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Jsonify")
		return nil
	}

	return r
}

func RawJsonToString(raw json.RawMessage) string {
	var str string
	err := json.Unmarshal(raw, &str)
	if err != nil {
		logger.Log.Error().Err(err).Msg("RawJsonToString")
	}

	return str
}
