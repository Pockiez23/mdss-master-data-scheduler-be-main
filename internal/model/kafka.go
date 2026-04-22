package model

import "time"

type RecalcuationRequest struct {
	ContractAccount string    `json:"contractAccount"`
	Date            time.Time `json:"date"` // Timezone Asia/Bangkok
}
