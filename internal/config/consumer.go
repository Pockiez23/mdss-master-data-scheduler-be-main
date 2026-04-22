package config

import (
	"context"
	"log"

	"github.com/IBM/sarama"
)

type ConsumerCallback func(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) error

type Consumer struct {
	ready    chan bool
	ctx      context.Context
	callback ConsumerCallback
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Printf("message channel closed!")
				return nil
			}
			log.Printf(
				"Message claimed: value = %s, timestamp = %v, topic = %s, partition = %d, offset = %d",
				string(message.Value),
				message.Timestamp,
				message.Topic,
				message.Partition,
				message.Offset,
			)
			if err := consumer.callback(session, message); err != nil {
				log.Printf(
					"Message clamied: error = %s, timestamp = %v, topic = %s, partition = %d, offset = %d",
					err,
					message.Timestamp,
					message.Topic,
					message.Partition,
					message.Offset,
				)
			}

		case <-session.Context().Done():
			return nil
		}
	}
}
