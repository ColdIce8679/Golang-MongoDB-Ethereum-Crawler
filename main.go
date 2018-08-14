package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gopkg.in/mgo.v2"
)

type BlockData struct {
	Height           int64  `json:"height"`
	TimeStamp        string `json:"timestamp"`
	TotalTxs         string `json:"totalTxs"`
	TotalUncles      string `json:"totalUncles"`
	Miner            string `json:"miner"`
	GasUsed          string `json:"gasUsed"`
	GasLimit         string `json:"gasLimit"`
	UtilityRateOfGas string `json:"utilityRateOfGas"`
}

func main() {
	// 連接資料庫
	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		fmt.Println("資料庫錯誤")
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	client, err := ethclient.Dial("https://ropsten.infura.io/")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("we have a connection")

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(header.Number.String()) // 5671744

	var number int64 = findLastBlock(session)
	var count int64 = header.Number.Int64()
	fmt.Println(number, count)
	for {
		if number >= count {
			fmt.Println("--完成--")
			time.Sleep(5 * time.Second)
			header, err = client.HeaderByNumber(context.Background(), nil)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("----最高高度調整為 " + strconv.FormatInt(header.Number.Int64(), 10) + " (" + "原先為 " + strconv.FormatInt(count, 10) + " )")
			count = header.Number.Int64()
		}

		// 設定欲查詢的block高度
		blockNumber := big.NewInt(number)
		// 使用高度抓Block資訊
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Fatal(err)
		}

		// 新增至資料庫
		addblockData(session, block)

		number++
	}
}

func addblockData(s *mgo.Session, block *types.Block) {
	session := s.Copy()
	defer session.Close()

	blockData := BlockData{
		Height:           block.Number().Int64(),
		TimeStamp:        strconv.FormatUint(block.Time().Uint64(), 10),
		TotalTxs:         strconv.Itoa(len(block.Transactions())),
		TotalUncles:      strconv.Itoa(len(block.Uncles())),
		Miner:            block.Coinbase().Hex(),
		GasUsed:          strconv.FormatUint(block.GasUsed(), 10),
		GasLimit:         strconv.FormatUint(block.GasLimit(), 10),
		UtilityRateOfGas: strconv.FormatUint(block.GasUsed()/block.GasLimit(), 10),
	}

	c := session.DB("ropsten").C("blocks")
	err := c.Insert(blockData)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("-----新增 " + strconv.FormatInt(blockData.Height, 10) + " 區塊成功-----")
}

func findLastBlock(s *mgo.Session) int64 {
	session := s.Copy()
	defer session.Close()

	var blockData []BlockData

	c := session.DB("ropsten").C("blocks")
	err := c.Find(nil).Sort("-height").Limit(10).All(&blockData)
	if err != nil {
		fmt.Println(err)
	}
	if len(blockData) == 0 {
		return 0
	}
	return blockData[0].Height + 1
}
