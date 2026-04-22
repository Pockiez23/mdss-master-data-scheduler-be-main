package rediz

import (
	"app/internal/global"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type IRepository interface {
	GetLatestOffset(ctx context.Context, key string) (time.Time, error)
	SetLatestOffset(ctx context.Context, key string, offset time.Time) error
	Upsert(ctx context.Context, keySuffix string, field string, value any) error
	GetValueByHashField(ctx context.Context, keySuffix string, field string) (string, error)
	SetGlobalThreePhase(ctx context.Context, keySuffix string, value bool) error
	Set(ctx context.Context, keySuffix string, value any, expiresIn time.Duration) error
}

type Repository struct {
	DB     *redis.Client
	Prefix string
}

func NewRepository(rdb *redis.Client) IRepository {
	return &Repository{
		DB:     rdb,
		Prefix: global.RedisPrefix,
	}
}

func (repo Repository) GetLatestOffset(ctx context.Context, key string) (time.Time, error) {
	ts, err := repo.DB.Get(ctx, fmt.Sprintf("%s:%s:latest_offset", repo.Prefix, key)).Result()
	if err != nil {
		return time.Time{}, err
	}

	timestamp, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

func (repo Repository) SetLatestOffset(ctx context.Context, key string, offset time.Time) error {
	if _, err := repo.DB.Set(ctx, fmt.Sprintf("%s:%s:latest_offset", repo.Prefix, key), offset.Unix(), 0).Result(); err != nil {
		return err
	}

	return nil
}

func (repo Repository) Upsert(ctx context.Context, keySuffix string, field string, value any) error {
	hkey := fmt.Sprintf("%s:%s", repo.Prefix, keySuffix)

	if _, err := repo.DB.HSet(ctx, hkey, field, value).Result(); err != nil {
		return err
	}

	return nil
}

func (repo Repository) GetValueByHashField(ctx context.Context, keySuffix string, field string) (string, error) {
	hkey := fmt.Sprintf("%s:%s", repo.Prefix, keySuffix)

	return repo.DB.HGet(ctx, hkey, field).Result()
}

func (repo Repository) SetGlobalThreePhase(ctx context.Context, keySuffix string, value bool) error {
	key := fmt.Sprintf("%s:%s", repo.Prefix, keySuffix)

	if _, err := repo.DB.Set(ctx, key, value, 0).Result(); err != nil {
		return err
	}

	return nil
}

func (repo Repository) Set(ctx context.Context, keySuffix string, value any, expiresIn time.Duration) error {
	key := fmt.Sprintf("%s:%s", repo.Prefix, keySuffix)

	if _, err := repo.DB.Set(ctx, key, value, expiresIn).Result(); err != nil {
		return err
	}

	return nil
}
