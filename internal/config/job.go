package config

type Job struct {
	MasterDataCron string `env:"JOB_MASTER_DATA_CRON,required" envDefault:"*/15 * * * *"`
	HolidayCron    string `env:"JOB_HOLIDAY_CRON,required" envDefault:"15 3 * * *"`
	DailyCron      string `env:"JOB_DAILY_CRON" envDefault:"0 0 * * *"` // this is test cron for production 1 minute interval `*/1 * * * *`
}
