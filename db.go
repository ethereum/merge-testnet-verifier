package main

import (
	"fmt"
	"math/big"
)

type DataPoints map[uint64]interface{}

func (dp DataPoints) ToInt() (map[uint64]*big.Int, error) {
	dataPoints := make(map[uint64]*big.Int)
	for k, v := range dp {
		bigIntVal, ok := v.(*big.Int)
		if !ok {
			return nil, fmt.Errorf("Invalid data for bigInt: %v", v)
		}
		dataPoints[k] = bigIntVal
	}
	return dataPoints, nil
}

func (dp DataPoints) ToUint64() (map[uint64]uint64, error) {
	dataPoints := make(map[uint64]uint64)
	for k, v := range dp {
		uint64Val, ok := v.(uint64)
		if !ok {
			return nil, fmt.Errorf("Invalid data for uint64: %v", v)
		}
		dataPoints[k] = uint64Val
	}
	return dataPoints, nil
}

func (dp DataPoints) AggregateUint64(aggregateFunc string) (uint64, error) {
	dataPoints, err := dp.ToUint64()
	if err != nil {
		return 0, err
	}
	aggregatedValue := uint64(0)
	switch aggregateFunc {
	case "count":
		for _, v := range dataPoints {
			if v > 0 {
				aggregatedValue++
			}
		}
	case "percentage":
		total := uint64(0)
		for _, v := range dataPoints {
			total++
			if v > 0 {
				aggregatedValue++
			}
		}
		aggregatedValue = (aggregatedValue * 100) / total
	case "average":
		count := uint64(0)
		for _, v := range dataPoints {
			aggregatedValue += v
			count++
		}
		if count > 0 {
			aggregatedValue = aggregatedValue / count
		}
	case "sum":
		for _, v := range dataPoints {
			aggregatedValue += v
		}
	case "maximum":
		for _, v := range dataPoints {
			if v > aggregatedValue {
				aggregatedValue = v
			}
		}
	case "minimum":
		aggregatedValue = uint64(0)
		firstVal := true
		for _, v := range dataPoints {
			if v < aggregatedValue || firstVal {
				aggregatedValue = v
				firstVal = false
			}
		}
	default:
		return aggregatedValue, fmt.Errorf("Invalid aggregate function for uint64: %s", aggregateFunc)
	}
	return aggregatedValue, nil
}

func (dp DataPoints) AggregateBigInt(aggregateFunc string) (*big.Int, error) {
	dataPoints, err := dp.ToInt()
	if err != nil {
		return nil, err
	}
	aggregatedValue := big.NewInt(0)

	switch aggregateFunc {
	case "count":
		for _, v := range dataPoints {
			if v.Cmp(big.NewInt(0)) > 0 {
				aggregatedValue = aggregatedValue.Add(aggregatedValue, big.NewInt(1))
			}
		}
	case "average":
		count := int64(0)
		for _, v := range dataPoints {
			aggregatedValue = aggregatedValue.Add(aggregatedValue, v)
			count++
		}
		if count > 0 {
			fmt.Printf("Total=%v, Count=%v\n", aggregatedValue, count)
			aggregatedValue = aggregatedValue.Div(aggregatedValue, big.NewInt(count))
		}
	case "sum":
		for _, v := range dataPoints {
			aggregatedValue = aggregatedValue.Add(aggregatedValue, v)
		}
	default:
		return nil, fmt.Errorf("Invalid aggregate function for bigInt: %s", aggregateFunc)
	}
	return aggregatedValue, nil
}
