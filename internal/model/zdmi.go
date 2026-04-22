package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"time"
)

const (
	IS_THREE_PHASE string = "is_three_phase"
)

var threePhaseMeterTypes = []string{"M030", "M040", "M050", "M060", "M080"}

type ZDMI struct {
	MANDT       string         // Primary key
	EQUNR       string         // Primary key
	SERNR       string         // Primary key, PEA No
	BIS         string         // Primary key, End Date
	AB          string         // Start Date
	PROCESSDATE string         // YYYYMMDD
	PROCESSTIME string         // HHMMSS
	VKONT       string         // Contract Account
	BILL_FACTOR Float64FromRat // Multi
	FUNKLAS     string         // Meter Type Code
	FUNKTXT     string         // Meter Type Description
	ANLAGE      string         // Installation
}

func (data ZDMI) Field() string {
	return base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s|%s|%s|%s", data.MANDT, data.EQUNR, data.VKONT, data.BIS))
}

func (data ZDMI) HashValue() (HashValue, error) {
	var result HashValue

	startDate, err := time.Parse("20060102 -0700", fmt.Sprintf("%s +0700", data.AB))
	if err != nil {
		return result, errors.Join(errors.New("parse start date"), err)
	}

	endDate, err := time.Parse("20060102 -0700", fmt.Sprintf("%s +0700", data.BIS))
	if err != nil {
		return result, errors.Join(errors.New("parse end date"), err)
	}

	return HashValue{
		Multiplier:   data.BILL_FACTOR.Float64(),
		PEANo:        data.SERNR,
		Installation: data.ANLAGE,
		StartDate:    startDate.Unix(),
		EndDate:      endDate.Unix(),
		IsThreePhase: slices.Contains(threePhaseMeterTypes, data.FUNKLAS),
	}, nil
}
