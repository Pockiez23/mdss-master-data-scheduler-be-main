package holiday

import (
	"app/internal/config"
	"app/internal/global"
	"app/internal/locker"
	"app/internal/model"
	"app/internal/rediz"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type IService interface {
	FetchHolidays(ctx context.Context) error
	SyncHolidaysFromGoogleSheet(ctx context.Context) error
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

func (service Service) SyncHolidaysFromGoogleSheet(ctx context.Context) error {
	conf, err := env.ParseAs[config.Config]()
	if err != nil {
		return errors.Join(errors.New("load config"), err)
	}

	url := conf.GoogleSheet.URL
	apiKey := conf.GoogleSheet.APIKey
	if apiKey == "" {
		log.Println("[GoogleSheet] API Key is missing. Skipping sync.")
		return nil
	}

	// Extract Spreadsheet ID using regex
	re := regexp.MustCompile(`/d/([a-zA-Z0-9-_]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return errors.New("invalid google sheet URL")
	}
	spreadsheetID := matches[1]

	// 1. Get Spreadsheet Metadata to find sheets
	metaURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s?key=%s", spreadsheetID, apiKey)
	resp, err := http.Get(metaURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get sheet metadata: %s", string(b))
	}

	var meta struct {
		Sheets []struct {
			Properties struct {
				Title string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return err
	}

	yearRegex := regexp.MustCompile(`^\d{4}$`)
	var (
		group errgroup.Group
	)

	for _, sheet := range meta.Sheets {
		title := sheet.Properties.Title
		if !yearRegex.MatchString(title) {
			continue
		}

		t := title

		group.Go(func() error {
			log.Printf("[HolidaySync] Starting sync for sheet %s\n", t)
			// 2. Get values for each year sheet
			dataURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?key=%s", spreadsheetID, t, apiKey)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataURL, nil)
			if err != nil {
				return err
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(res.Body)
				return fmt.Errorf("failed to get values for sheet %s: %s", t, string(b))
			}

			var values struct {
				Values [][]string `json:"values"`
			}

			if err := json.NewDecoder(res.Body).Decode(&values); err != nil {
				return err
			}

			upsertCount := 0
			// 3. Upsert data
			for i, row := range values.Values {
				if i == 0 {
					continue // Skip header
				}
				dateStr := row[0]
				day := ""
				if len(row) > 1 {
					day = row[1]
				}
				peakOffpeak := ""
				if len(row) > 2 {
					peakOffpeak = row[2]
				}
				name := ""
				if len(row) > 3 {
					name = row[3]
				}

				parsedDate, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					continue // Skip invalid date
				}

				h := model.Holiday{
					Date:        parsedDate,
					Day:         day,
					PeakOffpeak: peakOffpeak,
					Name:        name,
					UpdatedAt:   time.Now(),
					UpdatedBy:   "system-cron",
				}

				if err := service.Repository.Upsert(h); err != nil {
					log.Printf("[HolidaySync] Failed to upsert %s: %v\n", dateStr, err)
				} else {
					upsertCount++
				}
			}

			log.Printf("[HolidaySync] Upserted %d rows from sheet %s\n", upsertCount, t)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}
