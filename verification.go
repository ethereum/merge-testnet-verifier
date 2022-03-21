package main

import (
	"fmt"
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
	VerificationName       string            `json:"verificationName"`
	ClientType             ClientType        `json:"clientType"`
	PostMerge              bool              `json:"postMerge"`
	CheckDelaySeconds      int               `json:"checkDelaySeconds"`
	DataName               DataName          `json:"dataName"`
	AggregateFunction      AggregateFunction `json:"aggregateFunction"`
	AggregateFunctionValue InputValue        `json:"aggregateFunctionValue"`
	PassCriteria           PassCriteria      `json:"passCriteria"`
	PassValue              InputValue        `json:"passValue"`
}

type VerificationProbe struct {
	Verification               *Verification
	Client                     *Client
	PreviousDataPointSlotBlock uint64
	DataPointsPerSlotBlock     DataPoints
}

type VerificationProbes []VerificationProbe

var AllVerifications = []Verification{
	{
		VerificationName:  "Post-Merge Execution Blocks Produced",
		ClientType:        Execution,
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          BlockCount,
		AggregateFunction: Count,
		PassCriteria:      MinimumValue,
		PassValue:         "1",
	},

	{
		VerificationName:  "Post-Merge Execution Blocks Average GasUsed",
		ClientType:        Execution,
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          BlockGasUsed,
		AggregateFunction: Average,
		PassCriteria:      MinimumValue,
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Execution Blocks Average BaseFee",
		ClientType:        Execution,
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          BlockBaseFee,
		AggregateFunction: Average,
		PassCriteria:      MinimumValue,
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Execution Blocks Total Difficulty",
		ClientType:        Execution,
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          BlockDifficulty,
		AggregateFunction: Sum,
		PassCriteria:      MaximumValue,
		PassValue:         "0",
	},
	{
		VerificationName:       "Post-Merge Execution Blocks Invalid Uncle Hash",
		ClientType:             Execution,
		PostMerge:              true,
		CheckDelaySeconds:      1,
		DataName:               BlockUnclesHash,
		AggregateFunction:      CountUnequal,
		AggregateFunctionValue: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
		PassCriteria:           MaximumValue,
		PassValue:              "0",
	},
	{
		VerificationName:       "Post-Merge Execution Blocks Invalid Nonce",
		ClientType:             Execution,
		PostMerge:              true,
		CheckDelaySeconds:      1,
		DataName:               BlockNonce,
		AggregateFunction:      CountUnequal,
		AggregateFunctionValue: "0",
		PassCriteria:           MaximumValue,
		PassValue:              "0",
	},
	{
		VerificationName:  "Post-Merge Beacon Blocks Produced",
		ClientType:        Beacon,
		PostMerge:         true,
		CheckDelaySeconds: 12,
		DataName:          SlotBlock,
		AggregateFunction: Count,
		PassCriteria:      MinimumValue,
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Justified Epochs",
		ClientType:        Beacon,
		PostMerge:         true,
		CheckDelaySeconds: 12,
		DataName:          FinalizedEpoch,
		AggregateFunction: Count,
		PassCriteria:      MinimumValue,
		PassValue:         "1",
	},
	{
		VerificationName:  "Post-Merge Finalized Epochs",
		ClientType:        Beacon,
		PostMerge:         true,
		CheckDelaySeconds: 12,
		DataName:          FinalizedEpoch,
		AggregateFunction: Count,
		PassCriteria:      MinimumValue,
		PassValue:         "2",
	},
	{
		VerificationName:  "Post-Merge Attestations Per Slot",
		ClientType:        Beacon,
		PostMerge:         true,
		CheckDelaySeconds: 1,
		DataName:          SlotAttestationsPercentage,
		AggregateFunction: Average,
		PassCriteria:      MinimumValue,
		PassValue:         "95",
	},
}

func NewVerificationProbes(client Client, verifications []Verification) []VerificationProbe {
	clientType := client.ClientType()
	verifProbes := make([]VerificationProbe, 0)
	for _, v := range verifications {
		if v.ClientType == clientType {
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
			currentBlockSlot := v.PreviousDataPointSlotBlock + 1
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
	if dataLayer, ok := DataTypesPerLayer[v.Verification.ClientType]; ok {
		if dataType, ok := dataLayer[v.Verification.DataName]; ok {
			switch dataType {
			case Uint64:
				return v.VerifyUint64()
			case BigInt:
				return v.VerifyBigInt()
			}
		}
	}
	return VerificationOutcome{}, fmt.Errorf("Unknown data: %s", v.Verification.DataName)
}

func (v *VerificationProbe) VerifyBigInt() (VerificationOutcome, error) {
	aggregatedValue, err := v.DataPointsPerSlotBlock.AggregateBigInt(v.Verification.AggregateFunction, v.Verification.AggregateFunctionValue)
	if err != nil {
		return VerificationOutcome{}, err
	}

	passValue, err := v.Verification.PassValue.ToBigInt()
	if err != nil {
		return VerificationOutcome{}, err

	}
	switch v.Verification.PassCriteria {
	case MinimumValue:
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
	case MaximumValue:
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
	aggregatedValue, err := v.DataPointsPerSlotBlock.AggregateUint64(v.Verification.AggregateFunction, v.Verification.AggregateFunctionValue)
	if err != nil {
		return VerificationOutcome{}, err
	}

	passValue, err := v.Verification.PassValue.ToUint64()
	if err != nil {
		return VerificationOutcome{}, err

	}
	switch v.Verification.PassCriteria {
	case MinimumValue:
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
	case MaximumValue:
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
		if vp.Verification.ClientType == Execution {
			retVal++
		}
	}
	return retVal
}

func (vps VerificationProbes) BeaconVerifications() uint64 {
	retVal := uint64(0)
	for _, vp := range vps {
		if vp.Verification.ClientType == Beacon {
			retVal++
		}
	}
	return retVal
}
