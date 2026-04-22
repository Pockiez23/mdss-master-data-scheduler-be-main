package config

import (
	"app/internal/helper"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

type Kafka struct {
	Brokers  string `env:"KAFKA_BROKERS,required"`
	Version  string `env:"KAFKA_VERSION,required" envDefault:"3.7.2"`
	ClientID string `env:"KAFKA_CLIENT_ID" envDefault:"sasl_scram_client"`
	Net      KafkaNet
	Producer KafkaProducer
	Consumer KafkaConsumer
}

// Net
type KafkaNet struct {
	SASL KafkaNetSASL
	TLS  KafkaNetTLS
}

type KafkaNetSASL struct {
	Enable    bool   `env:"KAFKA_NET_SASL_ENABLE"`
	User      string `env:"KAFKA_NET_SASL_USERNAME"`
	Password  string `env:"KAFKA_NET_SASL_PASSWORD"`
	Handshake bool   `env:"KAFKA_NET_SASL_HANDSHAKE" envDefault:"true"`
	Mechanism string `env:"KAFKA_NET_SASL_MECHANISM"`
}

type KafkaNetTLS struct {
	Enable             bool   `env:"KAFKA_NET_TLS_ENABLE"`
	CAPath             string `env:"KAFKA_NET_TLS_CA_PATH"`
	InsecureSkipVerify bool   `env:"KAFKA_NET_TLS_INSECURE_SKIP_VERIFY"`
}

// Producer
type KafkaProducer struct {
	Topic KafkaProducerTopic
	Retry KafkaProducerRetry
}

type KafkaProducerTopic struct {
	MeterRecalculationRaw string `env:"KAFKA_PRODUCER_TOPIC_METER_RECALCULATION_RAW"`
}

type KafkaProducerRetry struct {
	Max int `env:"KAFKA_PRODUCER_RETRY_MAX" envDefault:"1"`
}

// Consumer
type KafkaConsumer struct {
	Topics    string `env:"KAFKA_CONSUMER_TOPICS"`
	Group     KafkaConsumerGroup
	Offsets   KafkaConsumerOffsets
	EnableDLQ bool `env:"KAFKA_CONSUMER_ENABLE_DLQ"`
}

type KafkaConsumerGroup struct {
	ID        string `env:"KAFKA_CONSUMER_GROUP_ID"`
	Rebalance KafkaConsumerGroupRebalance
}

type KafkaConsumerOffsets struct {
	Initial    string `env:"KAFKA_CONSUMER_OFFSETS_INITIAL"`
	AutoCommit KafkaConsumerOffsetsAutoCommit
}

type KafkaConsumerOffsetsAutoCommit struct {
	Enable bool `env:"KAFKA_CONSUMER_OFFSETS_AUTO_COMMIT" envDefault:"true"`
}

type KafkaConsumerGroupRebalance struct {
	GroupStrategy string `env:"KAFKA_CONSUMER_GROUP_REBALANCE_STRATEGY" envDefault:"STICKY"`
}

func (kafka Kafka) NewConfig() *sarama.Config {
	hostname, _ := os.Hostname()

	version, err := sarama.ParseKafkaVersion(kafka.Version)
	if err != nil {
		// Use default version when cannot parsed
		version = sarama.DefaultVersion
	}

	conf := sarama.NewConfig()
	conf.Version = version
	conf.ClientID = fmt.Sprintf("%s-%s-%d", kafka.ClientID, hostname, os.Getgid())
	conf.Metadata.Full = true

	// Net
	conf.Net.DialTimeout = 60 * time.Second
	conf.Net.ReadTimeout = 300 * time.Second
	conf.Net.WriteTimeout = 300 * time.Second

	// SASL
	conf.Net.SASL.Enable = kafka.Net.SASL.Enable
	conf.Net.SASL.User = kafka.Net.SASL.User
	conf.Net.SASL.Password = kafka.Net.SASL.Password
	conf.Net.SASL.Handshake = kafka.Net.SASL.Handshake
	if conf.Net.SASL.Enable {
		switch kafka.Net.SASL.Mechanism {
		case "SHA256":
			conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
			conf.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256

		case "SHA512":
			conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
			conf.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512

		case "PLAIN":
			conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext

		default:
			log.Fatal("SASL mechanism invalid")
		}
	}

	// TLS
	conf.Net.TLS.Enable = kafka.Net.TLS.Enable
	if conf.Net.TLS.Enable {
		caCert, err := os.ReadFile(fmt.Sprintf("%s/%s", helper.GetRootPath(), kafka.Net.TLS.CAPath))
		if err != nil {
			log.Fatal(errors.Join(errors.New("certificate path invalid"), err))
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		conf.Net.TLS.Config = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: kafka.Net.TLS.InsecureSkipVerify,
		}
	}

	// Producer
	conf.Producer.Retry.Max = kafka.Producer.Retry.Max
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Return.Successes = true

	// Consumer
	conf.Consumer.Group.Rebalance.Timeout = 300 * time.Second
	conf.Consumer.Group.Session.Timeout = 60 * time.Second
	conf.Consumer.Offsets.AutoCommit.Enable = kafka.Consumer.Offsets.AutoCommit.Enable
	switch kafka.Consumer.Group.Rebalance.GroupStrategy {
	case "ROUNDROBIN":
		conf.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}

	case "RANGE":
		conf.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}

	default:
		conf.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategySticky()}

	}

	switch kafka.Consumer.Offsets.Initial {
	case "OLDEST":
		conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	default:
		conf.Consumer.Offsets.Initial = sarama.OffsetNewest

	}

	return conf
}

func (kafka Kafka) NewClient(conf *sarama.Config) (KafkaClient, error) {
	client, err := sarama.NewClient(strings.Split(kafka.Brokers, ","), conf)
	if err != nil {
		return KafkaClient{}, err
	}

	return KafkaClient{client}, nil
}
