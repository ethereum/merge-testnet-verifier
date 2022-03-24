package main

import (
	"fmt"
	"time"

	"gopkg.in/inconshreveable/log15.v2"
)

type VerificationOutcome struct {
	Success bool
	Message string
}

type VerificationProbes []*VerificationProbe

func NewVerificationProbes(client Client, verifications []Verification) VerificationProbes {
	clientLayer := client.ClientLayer()
	verifProbes := make([]*VerificationProbe, 0)
	for _, v := range verifications {
		if v.ClientLayer == clientLayer {
			requiredClientFound := true
			if requiredClients, ok := MetricClientTypeRequirements[v.MetricName]; ok {
				requiredClientFound = false
				for _, requiredClient := range requiredClients {
					if requiredClient == client.ClientType() {
						requiredClientFound = true
						break
					}
				}
			}
			if requiredClientFound {
				dpoints := make(DataPoints)
				verif := v
				vProbe := VerificationProbe{
					Verification:           &verif,
					Client:                 client,
					DataPointsPerSlotBlock: dpoints,
				}
				verifProbes = append(verifProbes, &vProbe)
			}
		}
	}
	return verifProbes
}

func (vps *VerificationProbes) AnySyncing() bool {
	if vps == nil {
		return false
	}
	for _, v := range *vps {
		if v.IsSyncing {
			return true
		}
	}
	return false
}

func (v *VerificationProbe) Loop(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(time.Second * time.Duration(v.Verification.CheckDelaySeconds)):
		}

		if v.Verification.PostMerge {
			ttdBlockSlot, err := v.Client.UpdateGetTTDBlockSlot()
			if err != nil {
				log15.Warn("Error getting ttd block/slot", "client", v.Client.ClientType(), "clientID", v.Client.ClientID(), "error", err)
				continue
			}
			if ttdBlockSlot == nil {
				continue
			}

			if *ttdBlockSlot > v.PreviousDataPointSlotBlock {
				v.PreviousDataPointSlotBlock = *ttdBlockSlot
			}
		}

		latestBlockSlot, err := v.Client.GetLatestBlockSlotNumber()
		if err != nil {
			log15.Warn("Error getting latest block/slot number", "error", err)
			continue
		}

		if latestBlockSlot > v.PreviousDataPointSlotBlock {
			finishedSyncing := false
			if !v.IsSyncing && (latestBlockSlot-v.PreviousDataPointSlotBlock) > 10 {
				log15.Info("Syncing data", "type", v.Verification.MetricName)
				v.IsSyncing = true
			}
			currentBlockSlot := v.PreviousDataPointSlotBlock + 1
			for ; currentBlockSlot <= latestBlockSlot; currentBlockSlot++ {
				newDataPoint, err := v.Client.GetDataPoint(v.Verification.MetricName, currentBlockSlot)
				if err != nil {
					log15.Debug("Error during datapoint fetch", "client", v.Client.ClientType(), "clientID", v.Client.ClientID(), "datatype", v.Verification.MetricName, "error", err)
					break
				}
				v.DataPointsPerSlotBlock[currentBlockSlot] = newDataPoint
				v.PreviousDataPointSlotBlock = currentBlockSlot
				if currentBlockSlot == latestBlockSlot {
					finishedSyncing = true
				}
			}
			if v.IsSyncing && finishedSyncing {
				log15.Info("Finished syncing data", "datatype", v.Verification.MetricName)
				v.IsSyncing = false
				if !v.AllProbesClient.AnySyncing() {
					log15.Info("Finished syncing all data", "client", v.Client.ClientType(), "clientID", v.Client.ClientID())
				}
			}
		}
	}

}

func (v *VerificationProbe) Verify() (VerificationOutcome, error) {
	if dataLayer, ok := DataTypesPerLayer[v.Verification.ClientLayer]; ok {
		if dataType, ok := dataLayer[v.Verification.MetricName]; ok {
			switch dataType {
			case Uint64:
				return v.VerifyUint64()
			case BigInt:
				return v.VerifyBigInt()
			}
		}
	}
	return VerificationOutcome{}, fmt.Errorf("unknown data: %s", v.Verification.MetricName)
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
	return VerificationOutcome{}, fmt.Errorf("invalid pass criteria for bigInt: %s", v.Verification.PassCriteria)
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
	return VerificationOutcome{}, fmt.Errorf("invalid pass criteria for uint64: %s", v.Verification.PassCriteria)
}

func (vps VerificationProbes) ExecutionVerifications() uint64 {
	retVal := uint64(0)
	for _, vp := range vps {
		if vp.Verification.ClientLayer == Execution {
			retVal++
		}
	}
	return retVal
}

func (vps VerificationProbes) BeaconVerifications() uint64 {
	retVal := uint64(0)
	for _, vp := range vps {
		if vp.Verification.ClientLayer == Beacon {
			retVal++
		}
	}
	return retVal
}
