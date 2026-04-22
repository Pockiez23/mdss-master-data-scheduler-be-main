package model

import (
	"app/internal/helper"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
)

type DifferenceDate time.Time

func (dd DifferenceDate) Time() time.Time {
	return time.Time(dd)
}

func (dd DifferenceDate) RecalcuationRequest(contractAccount string) RecalcuationRequest {
	return RecalcuationRequest{
		ContractAccount: contractAccount,
		Date:            dd.Time().In(helper.DefaultLocation()),
	}
}

type DifferenceDates []DifferenceDate

func (dds DifferenceDates) ProducerMessages(topic, contractAccount string) ([]*sarama.ProducerMessage, error) {
	var messages []*sarama.ProducerMessage

	for _, date := range dds {
		value, err := json.Marshal(date.RecalcuationRequest(contractAccount))
		if err != nil {
			return nil, err
		}

		messages = append(messages, &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(contractAccount),
			Value: sarama.ByteEncoder(value),
		})
	}

	return messages, nil
}
