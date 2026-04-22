package holiday

import (
	"app/internal/global"
	"app/internal/locker"
	"app/internal/model"
	"app/internal/rediz"
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type IService interface {
	FetchHolidays(ctx context.Context) error
}

type Service struct {
	Repository      IRepository
	RedisRepository rediz.IRepository
}

func NewService(db *gorm.DB, rdb *redis.Client) IService {
	return &Service{
		Repository:      NewRepository(db),
		RedisRepository: rediz.NewRepository(rdb),
	}
}

func (service Service) FetchHolidays(ctx context.Context) error {
	const (
		task = "holiday"
	)

	// Ensure singular process
	if locker.IsLocked(task) {
		log.Println("[Fetched] Another process is running...\nSkip duplicated process!")
		return global.ErrLocked
	}

	locker.Lock(task)
	defer locker.Unlock(task)

	// Define current offset
	currentOffset := time.Now()

	// Find latest offset
	latestOffset, err := service.RedisRepository.GetLatestOffset(ctx, task)
	if err != nil && err != redis.Nil {
		return errors.Join(errors.New("get latest offset"), err)
	}

	// Fetch data from hana
	data, err := func(offset time.Time) ([]model.Holiday, error) {
		if offset.IsZero() {
			// Fetch all
			return service.Repository.FetchAll()

		} else {
			// Fetch scope
			return service.Repository.Fetch(offset)

		}
	}(latestOffset)
	if err != nil {
		return errors.Join(errors.New("fetch data from HANA"), err)
	}

	log.Printf("[Redis] Storing %d records\n", len(data))

	// Store on redis
	var (
		group errgroup.Group
	)

	for _, datum := range data {
		doc := datum

		group.Go(func() error {
			// Set to redis
			if err := service.RedisRepository.Set(ctx, doc.GetRedisKey(), true, 0); err != nil {
				return err
			}

			return nil
		})
	}

	// Group waiting
	if err := group.Wait(); err != nil {
		return err
	}

	log.Printf("[Redis] Stored %d records\n", len(data))

	// Stamp current offset
	if err := service.RedisRepository.SetLatestOffset(ctx, task, currentOffset); err != nil {
		return errors.Join(errors.New("set latest offset"), err)
	}

	log.Printf("[Redis] Set latest offset to %s\n", currentOffset)

	// Done
	return nil
}
