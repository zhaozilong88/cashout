package eth

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	logging "github.com/ipfs/go-log/v2"
	"github.com/zhaozilong88/cashout/eth/gps"
	"math/big"
)

var log = logging.Logger("eth")

type Config struct {
	Network         string `yaml:"network"`
	ContractAddress string `yaml:"contract_address"`
	GasLimit        uint64 `yaml:"gas_limit"`
	GasPrice        int64  `yaml:"gas_limit"`
}

type Contract struct {
	conf       Config
	token      *gps.GPSToken
	chainId    *big.Int
	privateKey *ecdsa.PrivateKey
}

func NewContract(conf Config) (*Contract, error) {
	client, err := ethclient.Dial(conf.Network)
	if err != nil {
		log.Errorf("Failed to connect to eth: %v", err)
		return nil, err
	}
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Errorf("Failed to get chainId: %v", err)
		return nil, err
	}
	token, err := gps.NewGPSToken(common.HexToAddress(conf.ContractAddress), client)
	if err != nil {
		log.Errorf("Failed to instantiate a Token contract: %v", err)
		return nil, err
	}

	return &Contract{
		conf:    conf,
		token:   token,
		chainId: chainId,
	}, nil
}

func (c *Contract) GetPaidOut(addr string) (*big.Int, error) {
	amount, err := c.token.PaidOut(nil, common.HexToAddress(addr))
	if err != nil {
		log.Errorf("failed to get Accounts, %v", err)
		return amount, err
	}
	return amount, nil
}

func (c *Contract) Cashout(privateKey *ecdsa.PrivateKey, cumulativePayout *big.Int, issuerSig []byte) (*types.Transaction, error) {
	opt, err := bind.NewKeyedTransactorWithChainID(privateKey, c.chainId)
	if err != nil {
		log.Errorf("failed to cashout, %v", err)
		return nil, err
	}
	opt.GasLimit = c.conf.GasLimit
	opt.GasPrice = big.NewInt(c.conf.GasPrice * (1000_000_000))

	tx, err := c.token.CashCheque(opt, cumulativePayout, issuerSig)
	if err != nil {
		log.Errorf("failed to cashout, %v", err)
		return tx, err
	}
	return tx, nil
}
