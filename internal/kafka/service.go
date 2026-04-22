package kafka

import (
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

type IService interface {
	Produces(messages []*sarama.ProducerMessage) error
}

type Service struct {
	Producer sarama.SyncProducer
}

func NewService(producer sarama.SyncProducer) IService {
	return &Service{
		Producer: producer,
	}
}

func (service Service) Produces(messages []*sarama.ProducerMessage) error {
	buffer, _ := json.Marshal(messages)

	if err := service.Producer.SendMessages(messages); err != nil {
		return err
	}
	log.Printf(
		"Messages produced: length = %d, messages = %s",
		len(messages),
		string(buffer),
	)

	return nil
}
