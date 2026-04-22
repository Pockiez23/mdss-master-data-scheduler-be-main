package recal

import (
	"app/internal/helper"
	"app/internal/model"
	"app/internal/rediz"
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type IService interface {
	CheckMasterDataExisting(ctx context.Context, datum model.ZDMI) (model.DifferenceDates, error)
}

type Service struct {
	RedisRepository rediz.IRepository
}

func NewService(rdb *redis.Client) IService {
	return &Service{
		RedisRepository: rediz.NewRepository(rdb),
	}
}

func (service Service) CheckMasterDataExisting(ctx context.Context, datum model.ZDMI) (model.DifferenceDates, error) {
	// Define
	var (
		differenceDates model.DifferenceDates
		loc             = helper.DefaultLocation()
	)

	// Check redis key
	value, err := service.RedisRepository.GetValueByHashField(ctx, datum.VKONT, datum.Field())
	if err != nil {
		if err == redis.Nil {
			// Do nothing when data not found in redis cache
			return nil, nil
		}

		return nil, err
	}

	// Found!
	var hashValue model.HashValue
	if err := json.Unmarshal([]byte(value), &hashValue); err != nil {
		return nil, err
	}

	// Compare
	newHashValue, err := datum.HashValue()
	if err != nil {
		return nil, err
	}

	if newHashValue.StartDate < hashValue.StartDate {
		diffDays := (hashValue.StartDate - newHashValue.StartDate) / 86400

		for day := int(diffDays); day >= 1; day-- {
			differenceDates = append(
				differenceDates,
				model.DifferenceDate(time.Unix(hashValue.StartDate, 0).In(loc).AddDate(0, 0, (day*-1))),
			)
		}
	}

	if newHashValue.EndDate > hashValue.EndDate {
		diffDays := (newHashValue.EndDate - hashValue.EndDate) / 86400

		for day := 1; day <= int(diffDays); day++ {
			differenceDates = append(
				differenceDates,
				model.DifferenceDate(time.Unix(hashValue.EndDate, 0).In(loc).AddDate(0, 0, day)),
			)
		}
	}

	// Done
	return differenceDates, nil
}
