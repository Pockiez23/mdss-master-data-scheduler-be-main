package multiply

import (
	"app/internal/global"
	"app/internal/kafka"
	"app/internal/locker"
	"app/internal/model"
	"app/internal/recal"
	"app/internal/rediz"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

type IService interface {
	FetchMasterData(ctx context.Context, contractAccount *string) (model.HashValues, error)
}

type Service struct {
	HanaRepository       IRepository
	RedisRepository      rediz.IRepository
	RecalculationService recal.IService
	KafkaService         kafka.IService
}

func NewService(db *sql.DB, rdb *redis.Client, producer sarama.SyncProducer) IService {
	return &Service{
		RedisRepository:      rediz.NewRepository(rdb),
		HanaRepository:       NewRepository(db),
		RecalculationService: recal.NewService(rdb),
		KafkaService:         kafka.NewService(producer),
	}
}

func (service Service) FetchMasterData(ctx context.Context, contractAccount *string) (model.HashValues, error) {
	const (
		task = "multiplier"
	)

	var (
		hashValues model.HashValues
	)

	// Ensure singular process
	if locker.IsLocked(task) {
		log.Println("[Fetched] Another process is running...\nSkip duplicated process!")
		return nil, global.ErrLocked
	}

	locker.Lock(task)
	defer locker.Unlock(task)

	// Define current offset
	currentOffset := time.Now()

	// Find latest offset
	latestOffset, err := service.RedisRepository.GetLatestOffset(ctx, task)
	if err != nil && err != redis.Nil {
		return nil, errors.Join(errors.New("get latest offset"), err)
	}

	// Fetch data from hana
	data, err := func(offset time.Time) ([]model.ZDMI, error) {
		if offset.IsZero() {
			// Fetch all
			return service.HanaRepository.FetchAll()

		} else {
			// Fetch scope
			return service.HanaRepository.Fetch(offset)

		}
	}(latestOffset)
	if err != nil {
		return nil, errors.Join(errors.New("fetch data from HANA"), err)
	}

	log.Printf("[Redis] Storing %d records\n", len(data))

	// Store on redis
	var (
		group errgroup.Group
	)

	for _, datum := range data {
		doc := datum

		group.Go(func() error {
			value, err := doc.HashValue()
			if err != nil {
				return errors.Join(errors.New("convert zdmi to hash value"), err)
			}

			if contractAccount != nil && *contractAccount == doc.VKONT {
				hashValues = append(hashValues, value)
			}

			// Check master data changed
			diff, err := service.RecalculationService.CheckMasterDataExisting(ctx, doc)
			if err != nil {
				return err
			}

			// Upsert to redis
			if err := service.RedisRepository.Upsert(ctx, doc.VKONT, doc.Field(), value); err != nil {
				return err
			}

			// Set three phase flag
			if err := service.RedisRepository.SetGlobalThreePhase(
				ctx,
				fmt.Sprintf("%s:%s", doc.VKONT, model.IS_THREE_PHASE),
				value.IsThreePhase,
			); err != nil {
				return err
			}

			// If non difference data, then return
			if len(diff) == 0 {
				return nil
			}

			// Produce to kafa when changed!
			messages, err := diff.ProducerMessages(global.Topic.MeterRecalculationRaw, doc.VKONT)
			if err != nil {
				return err
			}

			return service.KafkaService.Produces(messages)
		})
	}

	// Group waiting
	if err := group.Wait(); err != nil {
		return nil, err
	}

	log.Printf("[Redis] Stored %d records\n", len(data))

	// Stamp current offset
	if err := service.RedisRepository.SetLatestOffset(ctx, task, currentOffset); err != nil {
		return nil, errors.Join(errors.New("set latest offset"), err)
	}

	log.Printf("[Redis] Set latest offset to %s\n", currentOffset)

	return hashValues, nil
}
