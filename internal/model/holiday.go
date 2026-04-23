package model

import (
	"fmt"
	"time"
)

type Holiday struct {
	Date        time.Time `gorm:"column:date;primaryKey"`
	Day         string    `gorm:"column:day"`
	PeakOffpeak string    `gorm:"column:peak_offpeak"`
	Name        string    `gorm:"column:name"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
	UpdatedBy   string    `gorm:"column:updated_by"`
}

func (holiday Holiday) GetRedisKey() string {
	return fmt.Sprintf("holiday:%s", holiday.Date.Format("20060102"))
}
