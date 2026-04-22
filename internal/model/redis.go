package model

import "encoding/json"

type HashValue struct {
	Multiplier   float64 `json:"multiplier"`
	PEANo        string  `json:"pea_no"`
	Installation string  `json:"installation"`
	StartDate    int64   `json:"start_date"`
	EndDate      int64   `json:"end_date"`
	IsThreePhase bool    `json:"is_three_phase"`
}

func (value HashValue) MarshalBinary() (data []byte, err error) {
	return json.Marshal(value)
}

func (value HashValue) ToResponse() ContractAccountResponse {
	return ContractAccountResponse(value)
}

type HashValues []HashValue

func (values HashValues) ToResponses() []ContractAccountResponse {
	var results []ContractAccountResponse

	for _, value := range values {
		results = append(results, value.ToResponse())
	}

	return results
}
