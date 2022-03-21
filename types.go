package main

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type ClientType uint64

const (
	Execution ClientType = iota
	Beacon
)

func (l *ClientType) UnmarshalJSON(input []byte) error {
	s := string(input)
	if s == "Execution" {
		*l = Execution
		return nil
	} else if s == "Beacon" {
		*l = Beacon
		return nil
	}
	return fmt.Errorf("Invalid layer type: %s", s)
}

type DataName uint64

const (
	// Execution Types
	BlockCount DataName = iota
	BlockBaseFee
	BlockGasUsed
	BlockDifficulty
	BlockMixHash
	BlockUnclesHash
	BlockNonce
	// Beacon Types
	SlotBlock
	FinalizedEpoch
	JustifiedEpoch
	SlotAttestations
	SlotAttestationsPercentage
)

var DataNames = map[string]DataName{
	// Execution Types
	"BlockCount":      BlockCount,
	"BlockBaseFee":    BlockBaseFee,
	"BlockGasUsed":    BlockGasUsed,
	"BlockDifficulty": BlockDifficulty,
	"BlockMixHash":    BlockMixHash,
	"BlockUnclesHash": BlockUnclesHash,
	"BlockNonce":      BlockNonce,
	// Beacon Types
	"SlotBlock":                  SlotBlock,
	"FinalizedEpoch":             FinalizedEpoch,
	"JustifiedEpoch":             JustifiedEpoch,
	"SlotAttestations":           SlotAttestations,
	"SlotAttestationsPercentage": SlotAttestationsPercentage,
}

func (dn *DataName) UnmarshalJSON(input []byte) error {
	s := string(input)
	v, ok := DataNames[s]
	if !ok {
		return fmt.Errorf("Invalid data type: %s", s)
	}
	*dn = v
	return nil
}

func (dn DataName) String() string {
	for k, v := range DataNames {
		if dn == v {
			return k
		}
	}
	return ""
}

type DataType uint64

const (
	Uint64 DataType = iota
	BigInt
)

var DataTypesPerLayer = map[ClientType]map[DataName]DataType{
	Execution: {
		BlockCount:      Uint64,
		BlockBaseFee:    BigInt,
		BlockGasUsed:    Uint64,
		BlockDifficulty: BigInt,
		BlockMixHash:    BigInt,
		BlockUnclesHash: BigInt,
		BlockNonce:      Uint64,
	},
	Beacon: {
		SlotBlock:                  Uint64,
		FinalizedEpoch:             Uint64,
		JustifiedEpoch:             Uint64,
		SlotAttestations:           Uint64,
		SlotAttestationsPercentage: Uint64,
	},
}

type AggregateFunction uint64

const (
	Count AggregateFunction = iota
	CountUnequal
	CountEqual
	Average
	Sum
	Percentage
	Min
	Max
)

var AggregateFunctions = map[string]AggregateFunction{
	"Count":        Count,
	"CountUnequal": CountUnequal,
	"CountEqual":   CountEqual,
	"Average":      Average,
	"Sum":          Sum,
	"Percentage":   Percentage,
	"Min":          Min,
	"Max":          Max,
}

func (af *AggregateFunction) UnmarshalJSON(input []byte) error {
	s := string(input)
	v, ok := AggregateFunctions[s]
	if !ok {
		return fmt.Errorf("Invalid aggregate function: %s", s)
	}
	*af = v
	return nil
}

func (af AggregateFunction) String() string {
	for k, v := range AggregateFunctions {
		if af == v {
			return k
		}
	}
	return ""
}

type PassCriteria uint64

const (
	MinimumValue PassCriteria = iota
	MaximumValue
)

var PassCriterias = map[string]PassCriteria{
	"MinimumValue": MinimumValue,
	"MaximumValue": MaximumValue,
}

func (pc *PassCriteria) UnmarshalJSON(input []byte) error {
	s := string(input)
	v, ok := PassCriterias[s]
	if !ok {
		return fmt.Errorf("Invalid aggregate function: %s", s)
	}
	*pc = v
	return nil
}

func (pc PassCriteria) String() string {
	for k, v := range PassCriterias {
		if pc == v {
			return k
		}
	}
	return ""
}

type InputValue string

func (v InputValue) ToBigInt() (*big.Int, error) {
	var err error
	n := new(big.Int)
	vs := string(v)
	if len(vs) >= 2 && vs[0:2] == "0x" {
		n, err = hexutil.DecodeBig(vs)
		if err != nil {
			return nil, err
		}
	} else {
		var ok bool
		n, ok = n.SetString(vs, 10)
		if !ok {
			return nil, fmt.Errorf("Invalid value for bigInt: %s", vs)
		}
	}

	return n, nil
}

func (v InputValue) ToUint64() (uint64, error) {
	var err error
	var n uint64
	vs := string(v)
	if len(vs) >= 2 && vs[0:2] == "0x" {
		n, err = hexutil.DecodeUint64(vs)
		if err != nil {
			return 0, err
		}
	} else {
		n, err = strconv.ParseUint(vs, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid value for Uint64: %s", vs)
		}
	}

	return n, nil
}
