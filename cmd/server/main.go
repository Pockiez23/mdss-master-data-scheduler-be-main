package main

import (
	"app/internal/config"
	"app/internal/global"
	"app/internal/job"
	"app/internal/scheduler"
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/SAP/go-hdb/driver"
	"github.com/caarlos0/env/v11"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
	sarama.Logger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)

	global.Cron = cron.New()
	global.Cron.Start()
}

func main() {
	conf, err := env.ParseAs[config.Config]()
	if err != nil {
		log.Fatal(errors.Join(errors.New("load configuration"), err))
	}

	// Cache configuration to global
	global.RedisPrefix = conf.Redis.Prefix
	global.Topic = conf.Kafka.Producer.Topic

	// Redis client
	var rdb *redis.Client

	if conf.Redis.MasterName == "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:         strings.Split(conf.Redis.Addresses, ",")[0],
			Username:     conf.Redis.Username,
			Password:     conf.Redis.Password,
			DB:           conf.Redis.DB,
			PoolSize:     conf.Redis.PoolSize,
			PoolTimeout:  time.Duration(conf.Redis.PoolTimeoutInSecond) * time.Second,
			MinIdleConns: conf.Redis.MinIdleConnection,
		})
	} else {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       conf.Redis.MasterName,
			SentinelAddrs:    strings.Split(conf.Redis.Addresses, ","),
			Username:         conf.Redis.Username,
			Password:         conf.Redis.Password,
			SentinelUsername: conf.Redis.SentinalUsername,
			SentinelPassword: conf.Redis.SentinelPassword,
			DB:               conf.Redis.DB,
			PoolSize:         conf.Redis.PoolSize,
			PoolTimeout:      time.Duration(conf.Redis.PoolTimeoutInSecond) * time.Second,
			MinIdleConns:     conf.Redis.MinIdleConnection,
		})
	}

	// Kafka configuration
	kafkaConf := conf.Kafka.NewConfig()

	// Kafka client
	client, err := conf.Kafka.NewClient(kafkaConf)
	if err != nil {
		log.Fatal(errors.Join(errors.New("create kafka client"), err))
	}

	// Producer
	producer, err := client.NewProducer()
	if err != nil {
		log.Fatal(errors.Join(errors.New("create kafka producer"), err))
	}

	// Test redis connection
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatal(errors.Join(errors.New("ping redis connection"), err))
	}

	// New HANA database connection
	hdb, err := func() (*sql.DB, error) {
		if strings.HasPrefix(conf.Database.HANADSN, "sqlserver") {
			return sql.Open("sqlserver", conf.Database.HANADSN)
		} else if strings.HasPrefix(conf.Database.HANADSN, "postgres") || strings.HasPrefix(conf.Database.HANADSN, "postgresql") {
			return sql.Open("pgx", conf.Database.HANADSN)
		} else {
			return sql.Open(driver.DriverName, conf.Database.HANADSN)
		}
	}()
	if err != nil {
		log.Fatal(errors.Join(errors.New("connect database"), err))
	}

	// Test database connection
	if err := hdb.Ping(); err != nil {
		log.Fatal(errors.Join(errors.New("ping database connection"), err))
	}

	// New PostgreSQL database connection
	pgdb, err := gorm.Open(postgres.Open(conf.Database.PGDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		log.Fatal(errors.Join(errors.New("connect database"), err))
	}

	// Add scheduler
	schedule := scheduler.NewHandler(hdb, pgdb, rdb, producer)
	if err := schedule.AddFunc(conf.Job.MasterDataCron, schedule.MasterDataSync); err != nil {
		log.Fatal(errors.Join(errors.New("create master data sync scheduler"), err))
	}
	if err := schedule.AddFunc(conf.Job.HolidayCron, schedule.SyncHolidays); err != nil {
		log.Fatal(errors.Join(errors.New("create holiday sync scheduler"), err))
	}
	if err := schedule.AddFunc(conf.Job.DailyCron, schedule.DailyJob); err != nil {
		log.Fatal(errors.Join(errors.New("create daily sync scheduler"), err))
	}

	// Initializer
	schedule.MasterDataSync()
	schedule.SyncHolidays()
	schedule.DailyJob()

	// New handler
	handler := job.NewHandler(hdb, pgdb, rdb, producer)

	// API Router
	router := gin.Default()

	group := router.Group("/api")

	group.GET("/jobs", handler.GetJobs)
	group.GET("/contract-accounts/:id", handler.GetContractAccount)
	group.GET("/holidays", handler.GetHolidays)
	group.POST("/holidays/sync", handler.SyncGoogleSheet)
	// try port 3030 for testing
	if err := router.Run(":3030"); err != nil {
		log.Fatal(err)
	}
}
