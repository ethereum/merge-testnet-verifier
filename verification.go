package main

import (
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type VerificationOutcome struct {
	Success bool
	Message string
}

func (vOut *VerificationOutcome) String(verificationName string) string {
	if vOut.Success {
		return fmt.Sprintf("PASS (%s): %s", verificationName, vOut.Message)
	}
	return fmt.Sprintf("FAIL (%s): %s", verificationName, vOut.Message)
}

type Verification struct {
	VerificationName  string `json:"verificationName"`
	Layer             string `json:"layer"`
	PostMerge         bool   `json:"postMerge"`
	CheckDelaySeconds int    `json:"checkDelaySeconds"`
	DataName          string `json:"dataName"`
	AggregateFunction string `json:"aggregateFunction"`
	PassCriteria      string `json:"passCriteria"`
	PassValue         string `json:"passValue"`
}

type VerificationProbe struct {
	Verification               *Verification
	Client                     *Client
	PreviousDataPointSlotBlock uint64
	DataPointsPerSlotBlock     DataPoints
}

type VerificationProbes []VerificationProbe

var DataTypes = map[string]map[string]string{
	"Execution": {
		"BlockCount":      "uint64",
		"BlockBaseFee":    "bigInt",
		"BlockGasUsed":    "uint64",
		"BlockDifficulty": "bigInt",
	},
	"Beacon": {
		"SlotBlock":                  "uint64",
		"FinalizedEpoch":             "uint64",
		"JustifiedEpoch":             "uint64",
		"SlotAttestations":           "uint64",
		"SlotAttestationsPercentage": "uint64",
	},
}

var AllVerifications = []Verification{
	{
		VerificationName:  "Post-Merge Execution Blocks Produced",
		Layer:             "Execution",
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          "BlockCount",
		AggregateFunction: "count",
		PassCriteria:      "minimum",
		PassValue:         "1",
	},

	{
		VerificationName:  "Post-Merge Execution Blocks Average GasUsed",
		Layer:             "Execution",
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          "BlockGasUsed",
		AggregateFunction: "average",
		PassCriteria:      "minimum",
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Execution Blocks Average BaseFee",
		Layer:             "Execution",
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          "BlockBaseFee",
		AggregateFunction: "average",
		PassCriteria:      "minimum",
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Execution Blocks Total Difficulty",
		Layer:             "Execution",
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          "BlockBaseFee",
		AggregateFunction: "sum",
		PassCriteria:      "maximum",
		PassValue:         "0",
	},
	{
		VerificationName:  "Post-Merge Beacon Blocks Produced",
		Layer:             "Beacon",
		PostMerge:         true,
		CheckDelaySeconds: 12,
		DataName:          "SlotBlock",
		AggregateFunction: "count",
		PassCriteria:      "minimum",
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Justified Epochs",
		Layer:             "Beacon",
		PostMerge:         true,
		CheckDelaySeconds: 12,
		DataName:          "FinalizedEpoch",
		AggregateFunction: "count",
		PassCriteria:      "minimum",
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Finalized Epochs",
		Layer:             "Beacon",
		PostMerge:         true,
		CheckDelaySeconds: 12,
		DataName:          "FinalizedEpoch",
		AggregateFunction: "count",
		PassCriteria:      "minimum",
		PassValue:         "2",
	},
	{
		VerificationName:  "Post-Merge Attestations Per Slot",
		Layer:             "Beacon",
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          "SlotAttestationsPercentage",
		AggregateFunction: "average",
		PassCriteria:      "minimum",
		PassValue:         "95",
	},
}

func NewVerificationProbes(client Client, verifications []Verification) []VerificationProbe {
	clientType := client.ClientType()
	verifProbes := make([]VerificationProbe, 0)
	for _, v := range verifications {
		if v.Layer == clientType {
			dpoints := make(DataPoints)
			verif := v
			vProbe := VerificationProbe{
				Verification:           &verif,
				Client:                 &client,
				DataPointsPerSlotBlock: dpoints,
			}
			verifProbes = append(verifProbes, vProbe)
		}
	}
	return verifProbes
}

func (v *VerificationProbe) Loop(stop <-chan struct{}, wg sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stop:
			return
		case <-time.After(time.Second * time.Duration(v.Verification.CheckDelaySeconds)):
		}
		if v.Verification.PostMerge {
			ttdBlockSlot, err := (*v.Client).UpdateGetTTDBlockSlot()
			if err != nil {
				fmt.Printf("WARN: got error: %v\n", err)
				continue
			}
			if ttdBlockSlot == nil {
				continue
			}

			if *ttdBlockSlot > v.PreviousDataPointSlotBlock {
				v.PreviousDataPointSlotBlock = *ttdBlockSlot
			}
		}
		latestBlockSlot, err := (*v.Client).GetLatestBlockSlotNumber()
		if err != nil {
			fmt.Printf("WARN: got error: %v\n", err)
			continue
		}

		if latestBlockSlot > v.PreviousDataPointSlotBlock {
			currentBlockSlot := v.PreviousDataPointSlotBlock
			for ; currentBlockSlot <= latestBlockSlot; currentBlockSlot++ {
				newDataPoint, err := (*v.Client).GetDataPoint(v.Verification.DataName, currentBlockSlot)
				if err != nil {
					break
				}
				v.DataPointsPerSlotBlock[currentBlockSlot] = newDataPoint
				v.PreviousDataPointSlotBlock = currentBlockSlot
			}
		}
	}

}

func (v *VerificationProbe) Verify() (VerificationOutcome, error) {
	if dataLayer, ok := DataTypes[v.Verification.Layer]; ok {
		if dataType, ok := dataLayer[v.Verification.DataName]; ok {
			switch dataType {
			case "uint64":
				return v.VerifyUint64()
			case "bigInt":
				return v.VerifyBigInt()
			}
		}
	}
	return VerificationOutcome{}, fmt.Errorf("Unknown data: %s", v.Verification.DataName)
}

func (v *VerificationProbe) VerifyBigInt() (VerificationOutcome, error) {
	aggregatedValue, err := v.DataPointsPerSlotBlock.AggregateBigInt(v.Verification.AggregateFunction)
	if err != nil {
		return VerificationOutcome{}, err
	}

	n := new(big.Int)
	passValue, ok := n.SetString(v.Verification.PassValue, 10) //.ParseInt(v.Verification.PassValue, 10, 64)
	if !ok {
		return VerificationOutcome{}, fmt.Errorf("Invalid PassValue for bigInt: %s", v.Verification.PassValue)

	}
	switch v.Verification.PassCriteria {
	case "minimum":
		if aggregatedValue.Cmp(passValue) >= 0 {
			return VerificationOutcome{
				Success: true,
				Message: fmt.Sprintf("%v >= %v", aggregatedValue, passValue),
			}, nil
		} else {
			return VerificationOutcome{
				Success: false,
				Message: fmt.Sprintf("%v < %v", aggregatedValue, passValue),
			}, nil
		}
	case "maximum":
		if aggregatedValue.Cmp(passValue) <= 0 {
			return VerificationOutcome{
				Success: true,
				Message: fmt.Sprintf("%v <= %v", aggregatedValue, passValue),
			}, nil
		} else {
			return VerificationOutcome{
				Success: false,
				Message: fmt.Sprintf("%v > %v", aggregatedValue, passValue),
			}, nil
		}
	}
	return VerificationOutcome{}, fmt.Errorf("Invalid pass criteria for bigInt: %s", v.Verification.PassCriteria)
}

func (v *VerificationProbe) VerifyUint64() (VerificationOutcome, error) {
	aggregatedValue, err := v.DataPointsPerSlotBlock.AggregateUint64(v.Verification.AggregateFunction)
	if err != nil {
		return VerificationOutcome{}, err
	}

	passValue, err := strconv.ParseUint(v.Verification.PassValue, 10, 64)
	if err != nil {
		return VerificationOutcome{}, fmt.Errorf("Invalid PassValue for uint64: %s, %v", v.Verification.PassValue, err)

	}
	switch v.Verification.PassCriteria {
	case "minimum":
		if aggregatedValue >= passValue {
			return VerificationOutcome{
				Success: true,
				Message: fmt.Sprintf("%d >= %d", aggregatedValue, passValue),
			}, nil
		} else {
			return VerificationOutcome{
				Success: false,
				Message: fmt.Sprintf("%d < %d", aggregatedValue, passValue),
			}, nil
		}
	case "maximum":
		if aggregatedValue <= passValue {
			return VerificationOutcome{
				Success: true,
				Message: fmt.Sprintf("%d <= %d", aggregatedValue, passValue),
			}, nil
		} else {
			return VerificationOutcome{
				Success: false,
				Message: fmt.Sprintf("%d > %d", aggregatedValue, passValue),
			}, nil
		}
	}
	return VerificationOutcome{}, fmt.Errorf("Invalid pass criteria for uint64: %s", v.Verification.PassCriteria)
}

func (vps VerificationProbes) ExecutionVerifications() uint64 {
	retVal := uint64(0)
	for _, vp := range vps {
		if vp.Verification.Layer == "Execution" {
			retVal++
		}
	}
	return retVal
}

func (vps VerificationProbes) BeaconVerifications() uint64 {
	retVal := uint64(0)
	for _, vp := range vps {
		if vp.Verification.Layer == "Beacon" {
			retVal++
		}
	}
	return retVal
}
