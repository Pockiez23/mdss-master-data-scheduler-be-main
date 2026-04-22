package model

type ContractAccountResponse struct {
	Multiplier   float64 `json:"multiplier"`
	PEANo        string  `json:"peaNo"`
	Installation string  `json:"installation"`
	StartDate    int64   `json:"startDate"`
	EndDate      int64   `json:"endDate"`
	IsThreePhase bool    `json:"is_three_phase"`
}
