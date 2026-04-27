package job

import (
	"app/internal/global"
	"app/internal/holiday"
	"app/internal/multiply"
	"database/sql"
	"errors"
	"net/http"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type IHandler interface {
	GetJobs(ctx *gin.Context)
	GetContractAccount(ctx *gin.Context)
	GetHolidays(ctx *gin.Context)
	SyncGoogleSheet(ctx *gin.Context)
}

type Handler struct {
	MultiplyService multiply.IService
	HolidayService  holiday.IService
}

// func NewHandler(db *sql.DB, pgdb *gorm.DB, rdb *redis.Client, producer sarama.SyncProducer) IHandler {
func NewHandler(db *sql.DB, holidayDb *gorm.DB, rdb *redis.Client, producer sarama.SyncProducer) IHandler {
	return &Handler{
		MultiplyService: multiply.NewService(db, rdb, producer),
		//HolidayService:  holiday.NewService(pgdb, rdb),
		HolidayService: holiday.NewService(holidayDb, rdb),
	}
}

func (handler Handler) GetJobs(ctx *gin.Context) {
	var results []gin.H

	for _, entry := range global.Cron.Entries() {
		results = append(results, gin.H{
			"id":   entry.ID,
			"prev": entry.Prev,
			"next": entry.Next,
		})
	}

	ctx.JSON(http.StatusOK, results)
}

// GetContractAccount
//
// Path /api/contract-accounts/:id
//
//	Response [
//		{
//		    "multiplier": 275,
//		    "peaNo": "6200035469",
//		    "startDate": 1597338000,
//		    "endDate": 1597770000
//		},
//		{
//		    "multiplier": 250,
//		    "peaNo": "6200035411",
//		    "startDate": 1597338000,
//		    "endDate": 1597770000
//		}
//	]
func (handler Handler) GetContractAccount(ctx *gin.Context) {
	cid := ctx.Param("id")

	result, err := handler.MultiplyService.FetchMasterData(ctx, &cid)
	if err != nil {
		if errors.Is(err, global.ErrLocked) {
			ctx.Status(http.StatusNoContent)

			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	ctx.JSON(http.StatusOK, result.ToResponses())
}

func (handler Handler) GetHolidays(ctx *gin.Context) {
	if err := handler.HolidayService.FetchHolidays(ctx); err != nil {
		if errors.Is(err, global.ErrLocked) {
			ctx.Status(http.StatusNoContent)

			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	ctx.Status(http.StatusNoContent)
}

func (handler Handler) SyncGoogleSheet(ctx *gin.Context) {
	if err := handler.HolidayService.SyncHolidaysFromGoogleSheet(ctx); err != nil {
		if errors.Is(err, global.ErrLocked) {
			ctx.Status(http.StatusNoContent)
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	//ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Google Sheet synced to PostgreSQL"})
	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Google Sheet synced to SQL Server"})
}
