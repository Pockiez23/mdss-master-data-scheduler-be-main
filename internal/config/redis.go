package config

type Redis struct {
	MasterName          string `env:"REDIS_MASTER_NAME"`
	Addresses           string `env:"REDIS_ADDRESSES,required"`
	Username            string `env:"REDIS_USERNAME"`
	Password            string `env:"REDIS_PASSWORD"`
	SentinalUsername    string `env:"REDIS_SENTINEL_USERNAME"`
	SentinelPassword    string `env:"REDIS_SENTINEL_PASSWORD"`
	DB                  int    `env:"REDIS_DB"`
	Prefix              string `env:"REDIS_PREFIX,required" envDefault:"meterdata"`
	PoolSize            int    `env:"REDIS_POOL_SIZE" envDefault:"60"`
	PoolTimeoutInSecond int    `env:"REDIS_POOL_TIMEOUT_IN_SECOND" envDefault:"300"`
	MinIdleConnection   int    `env:"REDIS_MIN_IDLE_CONNECTION" envDefault:"10"`
}
