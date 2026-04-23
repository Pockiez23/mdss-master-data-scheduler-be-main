package config

type Config struct {
	Database    Database
	Redis       Redis
	Job         Job
	Kafka       Kafka
	GoogleSheet GoogleSheet
}

type GoogleSheet struct {
	URL    string `env:"GOOGLE_SHEET_URL,required"`
	APIKey string `env:"GOOGLE_SHEET_API_KEY"`
}
