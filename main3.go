package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/wire"
	btcdutil "github.com/btcsuite/btcutil"
	"github.com/tjaxer/stratumserver/miner"
	"github.com/tjaxer/stratumserver/stratum/server/utils"
	"strconv"
	"time"
	"gitlab.com/jaxnet/jaxnetd/jaxutil"
	"gitlab.com/jaxnet/jaxnetd/node/mining"
	"gitlab.com/jaxnet/jaxnetd/types/chaincfg"
)

//func UnpackUint32BE(byte[] buf) uint32 {
//
//}

func dumpTx(tx wire.MsgTx)  {
	var txBuffer2 []byte
	bf:=bytes.NewBuffer(txBuffer2)
	tx.Serialize(bf)
	fmt.Println("Transaction:", hex.EncodeToString(bf.Bytes()))

}

func main() {
	btcBlock := miner.Block100000
	fmt.Println("Block100000 hash:", btcBlock.BlockHash().String())
	fmt.Println("----- sending ------")


	merkleBranch := make([]string, 0)
	b:=btcdutil.NewBlock(&btcBlock)
	beaconHash:=[]byte{
		0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,
		0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,
		0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,
		0x00,0x00,
		}
	coinbaseScript, err := utils.BTCCoinbaseScript(100000, utils.PackUint64LE(0), beaconHash)
	if err != nil {
		fmt.Errorf("error %v", err.Error())
	}

	b.Transactions()[0].MsgTx().TxIn[0].SignatureScript = coinbaseScript
	BtcMiningAddress, err := jaxutil.DecodeAddress("13B2cWz2yeikEUKu9x3Vn2MbypC9txhJNd", &chaincfg.MainNetParams)
	if err != nil {
		fmt.Errorf("error %v", err.Error())
	}

	var conbaseTx *jaxutil.Tx
	conbaseTx, err = mining.CreateJaxCoinbaseTx(10, 3, int32(1000), 0,
		BtcMiningAddress, false)
	if err != nil {
		fmt.Errorf("error %v", err.Error())
	}
	cTx := utils.JaxTxToBtcTx(conbaseTx.MsgTx())
	b.MsgBlock().Transactions[0]=&cTx
	//block.Transactions[0] = &cTx


	dumpTx(*b.Transactions()[0].MsgTx())

	transactionBuf := bytes.NewBuffer(nil)
	b.Transactions()[0].MsgTx().Serialize(transactionBuf)



	transactionHexString:=hex.EncodeToString(transactionBuf.Bytes())
	fmt.Println("Transaction transactionHexString:", transactionHexString)


	part1, part2:= utils.SplitCoinbase(&cTx)
	b.MsgBlock().ClearTransactions()
	transactions:=b.Transactions()
	fmt.Printf("Transactions count: %v\n", len(transactions))
	merkles := blockchain.BuildMerkleTreeStore(transactions, false)
	//merkles := blockchain.BuildMerkleTreeStore(transactions[1:len(transactions)], false)
	merklePath := utils.MerkleTreeBranch(merkles)

	for _, merkle := range merklePath {
		if merkle != nil {
			merkleBranch = append(merkleBranch, merkle.String())
		}
	}


	ExtraNonce1:="00000000"
	fmt.Println("ExtraNonce1:", ExtraNonce1)

	fmt.Println("prevHash:", hex.EncodeToString(
		utils.ReverseByteOrder(
			btcBlock.Header.PrevBlock.CloneBytes())))
	fmt.Println("coinb1:", hex.EncodeToString(part1))
	fmt.Println("coinb2:",hex.EncodeToString(part2))
	fmt.Println("merkleBranch:",merkleBranch)
	fmt.Println("version:",hex.EncodeToString(utils.PackInt32BE(btcBlock.Header.Version)))
	fmt.Println("nBits:",hex.EncodeToString(utils.PackUint32BE(486604799)))
	fmt.Println("nTime:",hex.EncodeToString(utils.PackUint32BE(uint32(time.Now().Unix()))))

	fmt.Println("----- received ------")
	//ExtraNonce2:="0d000000"
	ExtraNonce2:="000000d0"
	fmt.Println("ExtraNonce2:", ExtraNonce2)
	nOnce:="83cea01e"
	fmt.Println("nOnce:", nOnce)
	nnOnce, _ := strconv.ParseUint(nOnce, 16, 32)
	nTime:="60f0ac5d"
	fmt.Println("nTime:", nTime)
	nnTime, _ := strconv.ParseInt(nTime, 16, 32)

	fmt.Println("----- updated  ------")

	rb1:=btcdutil.NewBlock(&btcBlock)
	rb1.MsgBlock().ClearTransactions()
	rb1.MsgBlock().Header.Nonce = uint32(nnOnce)
	rb1.MsgBlock().Header.Timestamp = time.Unix(nnTime,0)

	txHexString:=hex.EncodeToString(part1)+ExtraNonce1+ExtraNonce2+hex.EncodeToString(part2)
	txBuffer,_:=hex.DecodeString(transactionHexString)
	_ = bytes.NewReader(txBuffer)

	var msgTx wire.MsgTx
	e:=msgTx.Deserialize(bytes.NewReader(txBuffer))
	if e != nil {
		fmt.Errorf("Error %v", e.Error())
	}



	rb1.MsgBlock().AddTransaction(&msgTx)
	merklesX := blockchain.BuildMerkleTreeStore(rb1.Transactions(), false)
	calculatedMerkleRoot := merklesX[len(merklesX)-1]



	rb1.MsgBlock().Header.MerkleRoot= *calculatedMerkleRoot

	fmt.Println("Block hash:", rb1.Hash().String() )
	fmt.Println("Block hash:", rb1.MsgBlock().Header.BlockHash().String() )



	fmt.Println("Transaction:", txHexString)
	fmt.Println("Transaction transactionHexString:", transactionHexString)

	dumpTx(msgTx)









	//nTime, err := strconv.ParseInt("60f04374", 16, 32)
	//if err != nil {
	//	panic(err)
	//}

	//btcBlock.Header.Timestamp=uint32(nTime)

// INF handling submit ExtraNonce2="\"08000000\"" component=WorkerConnection jobId="\"XXX-1\"" method=mining.submit nOnce="\"2d2f4ca8\"" nTime="\"60f04374\"" username="\"4788dcb1-3b8f-431b-aaf7-d21e0988cb89.worker1\""
// {"method": "mining.submit", "params": ["4788dcb1-3b8f-431b-aaf7-d21e0988cb89.worker1", "XXX-1", "08000000", "60f04374", "2d2f4ca8"], "id":4}
// Hash:   00000000332d6d2d44c180bfedcda4d887645e97b66c557d7d7bd462299b922d


}
