package main

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/inconshreveable/log15.v2"
)

type TotalDifficulty struct {
	TotalDifficulty *hexutil.Big `json:"totalDifficulty"`
}

type ExecutionClient struct {
	Type   ClientType
	ID     int
	RPCUrl string
	Eth    *ethclient.Client
	RPC    *rpc.Client

	// Merge related
	TTD                TTD
	TTDBlockNumber     *uint64
	TTDBlockTimestamp  uint64
	UpdateTTDTimestamp func(uint64)

	// Lock
	l sync.Mutex

	// Context related
	lastCtx    context.Context
	lastCancel context.CancelFunc
}

func NewExecutionClient(clientType ClientType, id int, rpcUrl string) (*ExecutionClient, error) {
	rpcClient, err := rpc.DialHTTP(rpcUrl)
	if err != nil {
		return nil, err
	}

	return &ExecutionClient{
		Type:   clientType,
		ID:     id,
		RPCUrl: rpcUrl,
		Eth:    ethclient.NewClient(rpcClient),
		RPC:    rpcClient,
	}, nil
}

func (el *ExecutionClient) ClientLayer() ClientLayer {
	return Execution
}

func (el *ExecutionClient) ClientVersion() (string, error) {
	var clientVersion *string
	if err := el.RPC.CallContext(el.Ctx(), &clientVersion, "web3_clientVersion"); err != nil {
		return "", err
	}
	return *clientVersion, nil
}

func (el *ExecutionClient) UpdateGetTTDBlockSlot() (*uint64, error) {
	el.l.Lock()
	defer el.l.Unlock()

	if el.TTDBlockNumber == nil {
		var td *TotalDifficulty
		if err := el.RPC.CallContext(el.Ctx(), &td, "eth_getBlockByNumber", "latest", false); err != nil {
			return nil, err
		}

		if td.TotalDifficulty.ToInt().Cmp(el.TTD.Int) >= 0 {
			// TTD has been reached, we need to go backwards from latest block to find the non-zero difficulty block
			latestHeader, err := el.Eth.BlockByNumber(el.Ctx(), nil)
			if err != nil {
				return nil, err
			}
			for currentNumber := int64(latestHeader.NumberU64()); currentNumber >= 0; currentNumber-- {
				currentHeader, err := el.Eth.BlockByNumber(el.Ctx(), big.NewInt(currentNumber))
				if err != nil {
					return nil, err
				}
				if currentHeader.Difficulty().Cmp(big.NewInt(0)) > 0 {
					// We got the first block from head with a non-zero difficulty, this is the TTD block
					bn := currentHeader.Number().Uint64()
					el.TTDBlockNumber = &bn
					el.TTDBlockTimestamp = currentHeader.Time()
					if el.UpdateTTDTimestamp != nil {
						el.UpdateTTDTimestamp(el.TTDBlockTimestamp)
					}
					log15.Info("TTD Block Reached", "client", el.ClientID(), "block", bn)
					break
				}
				if currentNumber == 0 {
					return nil, fmt.Errorf("unable to get TTD Block")
				}
			}
		}
	}
	return el.TTDBlockNumber, nil
}

func (el *ExecutionClient) GetLatestBlockSlotNumber() (uint64, error) {
	el.l.Lock()
	defer el.l.Unlock()
	return el.Eth.BlockNumber(el.Ctx())
}

func (el *ExecutionClient) GetDataPoint(dataName MetricName, blockNumber uint64) (interface{}, error) {
	el.l.Lock()
	defer el.l.Unlock()
	header, err := el.Eth.HeaderByNumber(el.Ctx(), big.NewInt(int64(blockNumber)))
	if err != nil {
		return nil, err
	}
	switch dataName {
	case ExecutionBlockCount:
		// no error occured, we have a block
		return uint64(1), nil

	case ExecutionBaseFee:
		return header.BaseFee, nil

	case ExecutionGasUsed:
		return header.GasUsed, nil

	case ExecutionDifficulty:
		return header.Difficulty, nil

	case ExecutionMixHash:
		return header.MixDigest.Big(), nil

	case ExecutionUnclesHash:
		return header.UncleHash.Big(), nil

	case ExecutionNonce:
		return header.Nonce.Uint64(), nil
	}

	return nil, fmt.Errorf("invalid data name: %s", dataName)
}

func (el *ExecutionClient) Ctx() context.Context {
	if el.lastCtx != nil {
		el.lastCancel()
	}
	el.lastCtx, el.lastCancel = context.WithTimeout(context.Background(), 10*time.Second)
	return el.lastCtx
}

func (el *ExecutionClient) String() string {
	return el.RPCUrl
}

func (el *ExecutionClient) ClientType() ClientType {
	return el.Type
}

func (el *ExecutionClient) ClientID() int {
	return el.ID
}

func (el *ExecutionClient) Close() error {
	el.Eth.Close()
	return nil
}
