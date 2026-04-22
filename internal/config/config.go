package config

type Config struct {
	Database Database
	Redis    Redis
	Job      Job
	Kafka    Kafka
}
