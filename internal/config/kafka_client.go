package config

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
)

type KafkaClient struct {
	sarama.Client
}

type ConsumerProp struct {
	Topics   []string
	GroupID  string
	Callback ConsumerCallback
}

func (client KafkaClient) Consume(prop ConsumerProp) error {
	group, err := sarama.NewConsumerGroupFromClient(prop.GroupID, client)
	if err != nil {
		return err
	}
	defer func() {
		_ = group.Close()
	}()

	// Make consumer
	ctx, cancel := context.WithCancel(context.Background())
	consumer := Consumer{
		ready:    make(chan bool),
		ctx:      ctx,
		callback: prop.Callback,
	}

	keepRunning := true
	wg := &sync.WaitGroup{}
	wg.Go(func() {
		for {
			if err := group.Consume(ctx, prop.Topics, &consumer); err != nil {
				if errors.Is(err, sarama.ErrClosedClient) {
					return
				}
				log.Fatalf("Error from consumer: %v", err)
			}

			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	})

	<-consumer.ready
	log.Println("Sarama consumer up and running!...")


	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			log.Println("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Println("terminating: via signal")
			keepRunning = false
		}
	}
	cancel()
	wg.Wait()
	return client.Close()
}

func (client KafkaClient) NewProducer() (sarama.SyncProducer, error) {
	return sarama.NewSyncProducerFromClient(client)
}
