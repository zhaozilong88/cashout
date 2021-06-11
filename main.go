package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	crypto2 "github.com/ethersphere/bee/pkg/crypto"
	"github.com/zhaozilong88/cashout/eth"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
)

var conf = eth.Config{
	Network:         "https://bsc-dataseed.binance.org",
	ContractAddress: "0x5E772AcF0F20b0315391021e0884cb1F1Aa4545C",
	GasLimit:        5718749,
}

var (
	keyFile   = flag.String("key_file", "key.txt", "key file")
	gasPrice  = flag.Int64("gas_price", 5, "gas price Gwei")
	gasLimit  = flag.Uint64("gas_limit", 100000, "gas limit")
	minPayOut = flag.Int64("min_pay_out", 10000, "min pay out")
)

func readKeys(filename string) []string {
	var list []string
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("failed to read file, %v", err)
		return list
	}
	ss := strings.Split(string(data), "\n")
	for _, str := range ss {
		s := strings.TrimSpace(str)
		if len(s) == len("77dd33ed201813038b5c9a33b9eb0d4a07c3b83bd88e709e40228b762feedecd") {
			list = append(list, s)
		}
	}
	return list
}

func handleKeys(contract *eth.Contract, keys []string, minPayOut int64) {
	for _, key := range keys {
		prvKey, err := crypto.HexToECDSA(key)
		if err != nil {
			fmt.Printf("key error: %v\n", key)
			continue
		}
		singer := crypto2.NewDefaultSigner(prvKey)
		addr, err := singer.EthereumAddress()
		if err != nil {
			fmt.Printf("key error: %v\n", key)
			continue
		}

		amount, sign := getCheque(addr.String())
		reward := big.NewInt(amount)
		paidOut, err := contract.GetPaidOut(addr.String())

		//fmt.Printf("%v %v %v\n", addr.String(), reward.String(), paidOut.String())
		a := big.NewInt(0).Add(paidOut, big.NewInt(minPayOut*10000))
		if reward.Cmp(a) > 0 {
			hexSign, _ := hex.DecodeString(sign)
			_, err := contract.Cashout(prvKey, reward, hexSign)
			if err == nil && len(hexSign) > 0 {
				b := big.NewInt(0).Sub(reward, paidOut)
				fmt.Printf("%s %g\n", addr.String(), float64(b.Int64())/10000)
			}
		}
	}
}

func getCheque(address string) (int64, string) {
	// https://api.gpfs.xyz/v1/cheque?address=0x664e01fc0f9a5dc2e814af517dce25071525544f
	// {"code":0,"msg":"success","data":{"amount":2365437062,"paid_out":2066895147,"signature":"2b6fb4d75dbc1ef6f61d941fddbf9a49c6162cc7fea374e1528146088962683d409d83c2191becf367ce14915fff1ab404f1b86b67b2c05fd3482b4d61f1d46b1b"}}

	res, err := http.Get("https://api.gpfs.xyz/v1/cheque?address=" + strings.ToLower(address))
	if err != nil {
		fmt.Printf("failed to get cheque : %v", err)
		return 0, ""
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("failed to get cheque : %v", err)
		return 0, ""
	}
	//fmt.Printf("%v\n", string(data))

	type Data struct {
		Amount    int64  `json:"amount"`
		Signature string `json:"signature"`
	}
	ret := struct {
		Code int  `json:"code"`
		D    Data `json:"data"`
	}{}
	json.Unmarshal(data, &ret)
	return ret.D.Amount, ret.D.Signature
}

func main() {
	flag.Parse()

	keys := readKeys(*keyFile)
	if len(keys) == 0 {
		fmt.Printf("no key in file\n")
		return
	}
	//fmt.Printf("keys: %v\n", keys)
	conf.GasLimit = *gasLimit
	conf.GasPrice = *gasPrice

	contract, err := eth.NewContract(conf)
	if err != nil {
		panic(err)
	}
	handleKeys(contract, keys, *minPayOut)
}
