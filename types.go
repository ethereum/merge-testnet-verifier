package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gopkg.in/yaml.v2"
)

type Verification struct {
	VerificationName       string            `yaml:"VerificationName"`
	ClientLayer            ClientLayer       `yaml:"ClientLayer"`
	PostMerge              bool              `yaml:"PostMerge"`
	CheckDelaySeconds      int               `yaml:"CheckDelaySeconds"`
	MetricName             MetricName        `yaml:"MetricName"`
	AggregateFunction      AggregateFunction `yaml:"AggregateFunction"`
	AggregateFunctionValue InputValue        `yaml:"AggregateFunctionValue"`
	PassCriteria           PassCriteria      `yaml:"PassCriteria"`
	PassValue              InputValue        `yaml:"PassValue"`
}

type VerificationProbe struct {
	Verification               *Verification
	AllProbesClient            *VerificationProbes
	Client                     Client
	IsSyncing                  bool
	PreviousDataPointSlotBlock uint64
	DataPointsPerSlotBlock     DataPoints
}

type Verifications []Verification

func (vs *Verifications) Set(filePath string) error {
	yamlFile, err := os.Open(filePath)
	if err != nil {
		return err
	}

	bVal, _ := ioutil.ReadAll(yamlFile)
	var newVerifications Verifications
	if err := yaml.Unmarshal(bVal, &newVerifications); err != nil {
		return err
	}

	*vs = append(*vs, newVerifications...)
	return nil
}

func (vs *Verifications) String() string {
	names := make([]string, 0)
	for _, v := range *vs {
		names = append(names, v.VerificationName)
	}
	return strings.Join(names, ",")
}

type ClientType uint64

const (
	Geth ClientType = iota
	Nethermind
	Besu
	Erigon
	Lodestar
	Nimbus
	Teku
	Prysm
	Lighthouse
)

var ClientTypeNames = map[string]ClientType{
	// Execution Types
	"Geth":       Geth,
	"Nethermind": Nethermind,
	"Besu":       Besu,
	"Erigon":     Erigon,
	"Lodestar":   Lodestar,
	"Nimbus":     Nimbus,
	"Teku":       Teku,
	"Prysm":      Prysm,
	"Lighthouse": Lighthouse,
}

func (c ClientType) String() string {
	for k, v := range ClientTypeNames {
		if v == c {
			return k
		}
	}
	return ""
}

func ParseClientTypeString(str string) (ClientType, bool) {
	strLower := strings.ToLower(str)
	for k, v := range ClientTypeNames {
		if strings.ToLower(k) == strLower {
			return v, true
		}
	}
	return Geth, false
}

type ClientLayer uint64

const (
	Execution ClientLayer = iota
	Beacon
)

func (l *ClientLayer) UnmarshalText(input []byte) error {
	s := string(input)
	if s == "Execution" {
		*l = Execution
		return nil
	} else if s == "Beacon" {
		*l = Beacon
		return nil
	}
	return fmt.Errorf("invalid layer type: %s", s)
}

var ClientTypeToLayer = map[ClientType]ClientLayer{
	Geth:       Execution,
	Nethermind: Execution,
	Besu:       Execution,
	Erigon:     Execution,
	Lodestar:   Beacon,
	Nimbus:     Beacon,
	Teku:       Beacon,
	Prysm:      Beacon,
	Lighthouse: Beacon,
}

type MetricName uint64

const (
	// Execution Types
	BlockCount MetricName = iota
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
	EpochAttestationPerformance
	EpochTargetAttestationPerformance
	SyncParticipationCount
	SyncParticipationPercentage
)

var MetricClientTypeRequirements = map[MetricName][]ClientType{
	EpochAttestationPerformance: {
		// This metric requires validator_inclusion API from lighthouse
		Lighthouse,
	},
	EpochTargetAttestationPerformance: {
		// This metric requires validator_inclusion API from lighthouse
		Lighthouse,
	},
}

var MetricNames = map[string]MetricName{
	// Execution Types
	"BlockCount":      BlockCount,
	"BlockBaseFee":    BlockBaseFee,
	"BlockGasUsed":    BlockGasUsed,
	"BlockDifficulty": BlockDifficulty,
	"BlockMixHash":    BlockMixHash,
	"BlockUnclesHash": BlockUnclesHash,
	"BlockNonce":      BlockNonce,
	// Beacon Types
	"SlotBlock":                         SlotBlock,
	"FinalizedEpoch":                    FinalizedEpoch,
	"JustifiedEpoch":                    JustifiedEpoch,
	"SlotAttestations":                  SlotAttestations,
	"SlotAttestationsPercentage":        SlotAttestationsPercentage,
	"EpochAttestationPerformance":       EpochAttestationPerformance,
	"EpochTargetAttestationPerformance": EpochTargetAttestationPerformance,
	"SyncParticipationCount":            SyncParticipationCount,
	"SyncParticipationPercentage":       SyncParticipationPercentage,
}

func (dn *MetricName) UnmarshalText(input []byte) error {
	s := string(input)
	v, ok := MetricNames[s]
	if !ok {
		return fmt.Errorf("invalid data type: %s", s)
	}
	*dn = v
	return nil
}

func (dn MetricName) String() string {
	for k, v := range MetricNames {
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

var DataTypesPerLayer = map[ClientLayer]map[MetricName]DataType{
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
		SlotBlock:                         Uint64,
		FinalizedEpoch:                    Uint64,
		JustifiedEpoch:                    Uint64,
		SlotAttestations:                  Uint64,
		SlotAttestationsPercentage:        Uint64,
		EpochAttestationPerformance:       Uint64,
		EpochTargetAttestationPerformance: Uint64,
		SyncParticipationCount:            Uint64,
		SyncParticipationPercentage:       Uint64,
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

func (af *AggregateFunction) UnmarshalText(input []byte) error {
	s := string(input)
	v, ok := AggregateFunctions[s]
	if !ok {
		return fmt.Errorf("invalid aggregate function: %s", s)
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

func (pc *PassCriteria) UnmarshalText(input []byte) error {
	s := string(input)
	v, ok := PassCriterias[s]
	if !ok {
		return fmt.Errorf("invalid aggregate function: %s", s)
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
			return nil, fmt.Errorf("invalid value for bigInt: %s", vs)
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
			return 0, fmt.Errorf("invalid value for Uint64: %s", vs)
		}
	}

	return n, nil
}
