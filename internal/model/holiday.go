package model

import (
	"fmt"
	"time"
)

type Holiday struct {
	Date time.Time `gorm:"column:date"`
}

func (holiday Holiday) GetRedisKey() string {
	return fmt.Sprintf("holiday:%s", holiday.Date.Format("20060102"))
}
