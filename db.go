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

func (dp DataPoints) AggregateUint64(af AggregateFunction, aggregateFuncValue InputValue) (uint64, error) {
	dataPoints, err := dp.ToUint64()
	if err != nil {
		return 0, err
	}
	aggregatedValue := uint64(0)
	switch af {
	case Count:
		for _, v := range dataPoints {
			if v > 0 {
				aggregatedValue++
			}
		}
	case CountEqual:
		aggregateFuncValue, err := aggregateFuncValue.ToUint64()
		if err != nil {
			return 0, err
		}
		for _, v := range dataPoints {
			if v == aggregateFuncValue {
				aggregatedValue++
			}
		}
	case CountUnequal:
		aggregateFuncValue, err := aggregateFuncValue.ToUint64()
		if err != nil {
			return 0, err
		}
		for _, v := range dataPoints {
			if v != aggregateFuncValue {
				aggregatedValue++
			}
		}
	case Percentage:
		total := uint64(0)
		for _, v := range dataPoints {
			total++
			if v > 0 {
				aggregatedValue++
			}
		}
		aggregatedValue = (aggregatedValue * 100) / total
	case Average:
		count := uint64(0)
		for _, v := range dataPoints {
			aggregatedValue += v
			count++
		}
		if count > 0 {
			aggregatedValue = aggregatedValue / count
		}
	case Sum:
		for _, v := range dataPoints {
			aggregatedValue += v
		}
	case Max:
		for _, v := range dataPoints {
			if v > aggregatedValue {
				aggregatedValue = v
			}
		}
	case Min:
		aggregatedValue = uint64(0)
		firstVal := true
		for _, v := range dataPoints {
			if v < aggregatedValue || firstVal {
				aggregatedValue = v
				firstVal = false
			}
		}
	default:
		return aggregatedValue, fmt.Errorf("Invalid aggregate function for uint64: %s", af)
	}
	return aggregatedValue, nil
}

func (dp DataPoints) AggregateBigInt(af AggregateFunction, aggregateFuncValue InputValue) (*big.Int, error) {
	dataPoints, err := dp.ToInt()
	if err != nil {
		return nil, err
	}
	aggregatedValue := big.NewInt(0)
	switch af {
	case Count:
		for _, v := range dataPoints {
			if v.Cmp(big.NewInt(0)) > 0 {
				aggregatedValue = aggregatedValue.Add(aggregatedValue, big.NewInt(1))
			}
		}
	case CountEqual:
		aggregateFuncValue, err := aggregateFuncValue.ToBigInt()
		if err != nil {
			return nil, err
		}
		for _, v := range dataPoints {
			if v.Cmp(aggregateFuncValue) == 0 {
				aggregatedValue = aggregatedValue.Add(aggregatedValue, big.NewInt(1))
			}
		}
	case CountUnequal:
		aggregateFuncValue, err := aggregateFuncValue.ToBigInt()
		if err != nil {
			return nil, err
		}
		for _, v := range dataPoints {
			if v.Cmp(aggregateFuncValue) != 0 {
				aggregatedValue = aggregatedValue.Add(aggregatedValue, big.NewInt(1))
			}
		}
	case Average:
		count := int64(0)
		for _, v := range dataPoints {
			aggregatedValue = aggregatedValue.Add(aggregatedValue, v)
			count++
		}
		if count > 0 {
			aggregatedValue = aggregatedValue.Div(aggregatedValue, big.NewInt(count))
		}
	case Sum:
		for _, v := range dataPoints {
			aggregatedValue = aggregatedValue.Add(aggregatedValue, v)
		}
	case Min:
		firstVal := true
		for _, v := range dataPoints {
			v := v
			if firstVal || aggregatedValue.Cmp(v) > 0 {
				aggregatedValue = v
				firstVal = false
			}
		}
	case Max:
		firstVal := true
		for _, v := range dataPoints {
			v := v
			if firstVal || aggregatedValue.Cmp(v) < 0 {
				aggregatedValue = v
				firstVal = false
			}
		}
	default:
		return nil, fmt.Errorf("Invalid aggregate function for bigInt: %s", af)
	}
	return aggregatedValue, nil
}
