package config

type Database struct {
	HANADSN string `env:"DATABASE_HANA_DSN,required"`
	PGDSN   string `env:"DATABASE_PG_DSN,required"`
}
