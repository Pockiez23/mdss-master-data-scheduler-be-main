package model

import (
	"fmt"
	"math/big"
)

type Float64FromRat float64

func (f *Float64FromRat) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*f = 0

	case *big.Rat:
		val, _ := value.Float64()
		*f = Float64FromRat(val)

	case float64:
		*f = Float64FromRat(value)

	default:
		return fmt.Errorf("unsupported type: %T convert to float64", src)

	}

	return nil
}

func (f Float64FromRat) Float64() float64 {
	return float64(f)
}
