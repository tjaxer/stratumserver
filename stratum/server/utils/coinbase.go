package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	btcdchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	btcdwire "github.com/btcsuite/btcd/wire"
	"github.com/tjaxer/stratumserver/logger"
	"gitlab.com/jaxnet/jaxnetd/types/wire"
	"math/rand"
	"strings"
)

const (
	CoinbaseFlags = "/P2SH/jaxnet/"
)

func MerkleTreeBranch(merkles []*btcdchainhash.Hash) (path []*btcdchainhash.Hash) {
	b := len(merkles)

	for k := b; k > 1; {
		path = append(path, merkles[b-k+1])
		k = (k - 1) / 2
	}

	return
}


func StringsIndexOf(data []string, element string) int {
	for k, v := range data {
		if strings.Compare(element, v) == 0 {
			return k
		}
	}
	return -1 // not found.
}

func RandHexUint64() string {
	randomNumBytes := make([]byte, 8)
	_, err := rand.Read(randomNumBytes)
	if err != nil {
		logger.Log.Error().Err(err).Send()
	}

	return hex.EncodeToString(randomNumBytes)
}

// ReverseByteOrder converts bytes in Little endian from/to Big endian
func ReverseByteOrder(b []byte) []byte {
	_b := make([]byte, len(b))
	copy(_b, b)

	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint32(_b[i*4:], binary.BigEndian.Uint32(_b[i*4:]))
	}
	return ReverseBytes(_b)
}

func ReverseBytes(b []byte) []byte {
	_b := make([]byte, len(b))
	copy(_b, b)

	for i, j := 0, len(_b)-1; i < j; i, j = i+1, j-1 {
		_b[i], _b[j] = _b[j], _b[i]
	}
	return _b
}

func PackInt32BE(n int32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return b
}

func PackUint32BE(n uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return b
}

func SerializeNumber(n uint64) []byte {
	if n >= 1 && n <= 16 {
		return []byte{
			0x50 + byte(n),
		}
	}

	l := 1
	buff := make([]byte, 9)
	for n > 0x7f {
		buff[l] = byte(n & 0xff)
		l++
		n >>= 8
	}
	buff[0] = byte(l)
	buff[l] = byte(n)

	return buff[0 : l+1]
}

func SerializeString(s string) []byte {
	if len(s) < 253 {
		return bytes.Join([][]byte{
			{byte(len(s))},
			[]byte(s),
		}, nil)
	} else if len(s) < 0x10000 {
		return bytes.Join([][]byte{
			{253},
			PackUint16LE(uint16(len(s))),
			[]byte(s),
		}, nil)
	} else if len(s) < 0x100000000 {
		return bytes.Join([][]byte{
			{254},
			PackUint32LE(uint32(len(s))),
			[]byte(s),
		}, nil)
	} else {
		return bytes.Join([][]byte{
			{255},
			PackUint64LE(uint64(len(s))),
			[]byte(s),
		}, nil)
	}
}

func PackUint16LE(n uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, n)
	return b
}

func PackUint32LE(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func PackUint64LE(n uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	return b
}

func VarIntBytes(n uint64) []byte {
	if n < 0xFD {
		return []byte{byte(n)}
	}

	if n <= 0xFFFF {
		buff := make([]byte, 3)
		buff[0] = 0xFD
		binary.LittleEndian.PutUint16(buff[1:], uint16(n))
		return buff
	}

	if n <= 0xFFFFFFFF {
		buff := make([]byte, 5)
		buff[0] = 0xFE
		binary.LittleEndian.PutUint32(buff[1:], uint32(n))
		return buff
	}

	buff := make([]byte, 9)
	buff[0] = 0xFF
	binary.LittleEndian.PutUint64(buff[1:], uint64(n))
	return buff
}

func Uint256BytesFromHash(h string) []byte {
	container := make([]byte, 32)
	fromHex, err := hex.DecodeString(h)
	if err != nil {
		logger.Log.Error().Err(err).Send()
	}

	copy(container, fromHex)

	return ReverseBytes(container)
}


func BTCCoinbaseScript(nextBlockHeight int64, extraNonce, beaconHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddInt64(nextBlockHeight).
		AddData(extraNonce).
		AddData(beaconHash).
		AddData([]byte(CoinbaseFlags)).
		Script()
}



func SplitCoinbase(tx *btcdwire.MsgTx) ([]byte, []byte) {
	buf := bytes.NewBuffer(nil)
	tx.Serialize(buf)

	rawTx := buf.Bytes()
	heightLenIdx := 42
	heightLen := int(rawTx[heightLenIdx])
	if heightLen > 0xF {
		fmt.Println("heightLen 0xF:", heightLen)
		// if value more than 0xF,
		// this indicates that height is packed as an opcode OP_0 .. OP_16 (small int)
		// height = int(rawTx[heightLenIdx] - (txscript.OP_1 - 1))
		// so height value doesn't gives additional padding
		heightLen = 0
	}
	fmt.Println("heightLen:", heightLen)
	extraNonceLenIdx := heightLenIdx + 1 + heightLen
	extraNonceLen := int(rawTx[extraNonceLenIdx])
	fmt.Println("extraNonceLen:",extraNonceLen)
	// extraNonceLenIdx := 42
	// extraNonceLen := int(rawTx[extraNonceLenIdx])

	part1 := rawTx[0:extraNonceLenIdx]
	part1 = append(part1, 0x08)
	part2 := rawTx[extraNonceLenIdx+extraNonceLen+1:]
	return part1, part2
}



func JaxTxToBtcTx(tx *wire.MsgTx) btcdwire.MsgTx {
	msgTx := btcdwire.MsgTx{
		Version:  tx.Version,
		TxIn:     make([]*btcdwire.TxIn, len(tx.TxIn)),
		TxOut:    make([]*btcdwire.TxOut, len(tx.TxOut)),
		LockTime: tx.LockTime,
	}

	for i := range msgTx.TxIn {
		msgTx.TxIn[i] = &btcdwire.TxIn{
			PreviousOutPoint: btcdwire.OutPoint{
				Hash:  btcdchainhash.Hash(tx.TxIn[i].PreviousOutPoint.Hash),
				Index: tx.TxIn[i].PreviousOutPoint.Index,
			},
			SignatureScript: tx.TxIn[i].SignatureScript,
			Witness:         btcdwire.TxWitness(tx.TxIn[i].Witness),
			Sequence:        tx.TxIn[i].Sequence,
		}
	}

	for i := range msgTx.TxOut {
		msgTx.TxOut[i] = &btcdwire.TxOut{
			Value:    tx.TxOut[i].Value,
			PkScript: tx.TxOut[i].PkScript,
		}
	}
	return msgTx
}
