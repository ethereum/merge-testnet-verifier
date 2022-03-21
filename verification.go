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

type VerificationProbes []VerificationProbe

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
				newDataPoint, err := (*v.Client).GetDataPoint(v.Verification.MetricName, currentBlockSlot)
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
		if dataType, ok := dataLayer[v.Verification.MetricName]; ok {
			switch dataType {
			case Uint64:
				return v.VerifyUint64()
			case BigInt:
				return v.VerifyBigInt()
			}
		}
	}
	return VerificationOutcome{}, fmt.Errorf("Unknown data: %s", v.Verification.MetricName)
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
