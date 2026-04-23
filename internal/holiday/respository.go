package holiday

import (
	"app/internal/model"
	"time"

	"gorm.io/gorm"
)

type IRepository interface {
	FetchAll() ([]model.Holiday, error)
	Fetch(offset time.Time) ([]model.Holiday, error)
	Upsert(holiday model.Holiday) error
}

type Repository struct {
	DB *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &Repository{
		DB: db,
	}
}

func (repo Repository) FetchAll() ([]model.Holiday, error) {
	var results []model.Holiday

	if err := repo.DB.Table("mst.pea_holiday").Where(
		"peak_offpeak = ?", "OFFPEAK",
	).Where(
		"name IS NOT NULL",
	).Where(
		`name <> ''`,
	).Order("date ASC").Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (repo Repository) Fetch(offset time.Time) ([]model.Holiday, error) {
	var results []model.Holiday

	if err := repo.DB.Table("mst.pea_holiday").Select("date").Where(
		"peak_offpeak = ?", "OFFPEAK",
	).Where(
		"name IS NOT NULL",
	).Where(
		`name <> ''`,
	).Where(
		"created_at >= ? OR updated_at >= ?", offset, offset,
	).Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (repo Repository) Upsert(holiday model.Holiday) error {
	var count int64
	repo.DB.Table("mst.pea_holiday").Where("date = ?", holiday.Date).Count(&count)

	if count > 0 {
		return repo.DB.Table("mst.pea_holiday").Where("date = ?", holiday.Date).Updates(map[string]interface{}{
			"day":          holiday.Day,
			"peak_offpeak": holiday.PeakOffpeak,
			"name":         holiday.Name,
			"updated_at":   holiday.UpdatedAt,
			"updated_by":   holiday.UpdatedBy,
		}).Error
	}

	return repo.DB.Table("mst.pea_holiday").Create(&holiday).Error
}
