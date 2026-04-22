package scheduler

import (
	"app/internal/global"
	"app/internal/holiday"
	"app/internal/multiply"
	"context"
	"database/sql"
	"log"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type IHandler interface {
	AddFunc(crontab string, fn func()) error
	MasterDataSync()
	SyncHolidays()
	DailyJob()
}

type Handler struct {
	MultiplyService multiply.IService
	HolidayService  holiday.IService
}

func NewHandler(db *sql.DB, pgdb *gorm.DB, rdb *redis.Client, producer sarama.SyncProducer) IHandler {
	return &Handler{
		MultiplyService: multiply.NewService(db, rdb, producer),
		HolidayService:  holiday.NewService(pgdb, rdb),
	}
}

func (handler Handler) AddFunc(crontab string, fn func()) error {
	eid, err := global.Cron.AddFunc(crontab, fn)
	if err != nil {
		return err
	}

	log.Printf("Add func with entry id %d\n", eid)

	return nil
}

func (handler Handler) MasterDataSync() {
	ctx := context.Background()

	if _, err := handler.MultiplyService.FetchMasterData(ctx, nil); err != nil {
		log.Println("Fetch master data:", err)
	}
}

func (handler Handler) SyncHolidays() {
	ctx := context.Background()

	if err := handler.HolidayService.FetchHolidays(ctx); err != nil {
		log.Println("Fetch holidays:", err)
	}
}

func (handler Handler) DailyJob() {
	log.Println("Daily job executed at midnight/startup")
	// TODO: Add logic for the daily job here
}
