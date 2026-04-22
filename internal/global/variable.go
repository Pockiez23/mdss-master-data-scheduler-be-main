package global

import (
	"app/internal/config"

	"github.com/robfig/cron/v3"
)

var (
	Cron        *cron.Cron
	RedisPrefix string
	Topic       config.KafkaProducerTopic
)
