package multiply

import (
	"app/internal/model"
	"database/sql"
	"log"
	"time"
)

type IRepository interface {
	FetchAll() ([]model.ZDMI, error)
	Fetch(offset time.Time) ([]model.ZDMI, error)
}

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) IRepository {
	return &Repository{
		DB: db,
	}
}

func (repo Repository) FetchAll() ([]model.ZDMI, error) {
	query := `SELECT
		zdm_fu_zdmi025.MANDT,
		EQUNR,
		SERNR,
		BIS,
		AB,
		PROCESSDATE,
		PROCESSTIME,
		VKONT,
		CAST(BILL_FACTOR AS FLOAT),
		meter_master.FUNKLAS,
		meter_type.FUNKTXT,
		ANLAGE
	FROM
		"INFUSER"."REP_RT_ZDM_FU_ZDMI025" AS zdm_fu_zdmi025
	INNER JOIN "INFUSER"."REP_RT_METER_MASTER" AS meter_master
		ON zdm_fu_zdmi025.MANDT = meter_master.MANDT AND LTRIM(meter_master.MATNR, '0') = zdm_fu_zdmi025.MATNR
	INNER JOIN "INFUSER"."REP_RT_METER_TYPE" AS meter_type
		ON meter_master.MANDT = meter_type.MANDT AND meter_master.FUNKLAS = meter_type.FUNKLAS
	WHERE
		SERNR IS NOT NULL AND SERNR <> '' AND
		BIS IS NOT NULL AND BIS <> '' AND
		AB IS NOT NULL AND AB <> '' AND
		PROCESSDATE IS NOT NULL AND PROCESSDATE <> '' AND
		PROCESSTIME IS NOT NULL AND PROCESSTIME <> '' AND
		VKONT IS NOT NULL AND VKONT <> '' AND
		BILL_FACTOR IS NOT NULL`

	log.Printf("[Query] %s\n", query)

	rows, err := repo.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var data []model.ZDMI

	for rows.Next() {
		var datum model.ZDMI

		if err := rows.Scan(
			&datum.MANDT,
			&datum.EQUNR,
			&datum.SERNR,
			&datum.BIS,
			&datum.AB,
			&datum.PROCESSDATE,
			&datum.PROCESSTIME,
			&datum.VKONT,
			&datum.BILL_FACTOR,
			&datum.FUNKLAS,
			&datum.FUNKTXT,
			&datum.ANLAGE,
		); err != nil {
			return nil, err
		}

		data = append(data, datum)
	}

	log.Printf("[Result] %d records\n", len(data))

	return data, nil
}

func (repo Repository) Fetch(offset time.Time) ([]model.ZDMI, error) {
	query := `SELECT
		zdm_fu_zdmi025.MANDT,
		EQUNR,
		SERNR,
		BIS,
		AB,
		PROCESSDATE,
		PROCESSTIME,
		VKONT,
		CAST(BILL_FACTOR AS FLOAT),
		meter_master.FUNKLAS,
		meter_type.FUNKTXT,
		ANLAGE
	FROM
		"INFUSER"."REP_RT_ZDM_FU_ZDMI025" AS zdm_fu_zdmi025
	INNER JOIN "INFUSER"."REP_RT_METER_MASTER" AS meter_master
		ON zdm_fu_zdmi025.MANDT = meter_master.MANDT AND LTRIM(meter_master.MATNR, '0') = zdm_fu_zdmi025.MATNR
	INNER JOIN "INFUSER"."REP_RT_METER_TYPE" AS meter_type
		ON meter_master.MANDT = meter_type.MANDT AND meter_master.FUNKLAS = meter_type.FUNKLAS
	WHERE
		PROCESSDATE >= $1 AND PROCESSTIME >= $2 AND
		SERNR IS NOT NULL AND SERNR <> '' AND
		BIS IS NOT NULL AND BIS <> '' AND
		AB IS NOT NULL AND AB <> '' AND
		PROCESSDATE IS NOT NULL AND PROCESSDATE <> '' AND
		PROCESSTIME IS NOT NULL AND PROCESSTIME <> '' AND
		VKONT IS NOT NULL AND VKONT <> '' AND
		BILL_FACTOR IS NOT NULL`

	log.Printf("[Query] %s, %s, %s\n", query, offset.Format("20060102"), offset.Format("150405"))

	rows, err := repo.DB.Query(query, offset.Format("20060102"), offset.Format("150405"))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var data []model.ZDMI

	for rows.Next() {
		var datum model.ZDMI

		if err := rows.Scan(
			&datum.MANDT,
			&datum.EQUNR,
			&datum.SERNR,
			&datum.BIS,
			&datum.AB,
			&datum.PROCESSDATE,
			&datum.PROCESSTIME,
			&datum.VKONT,
			&datum.BILL_FACTOR,
			&datum.FUNKLAS,
			&datum.FUNKTXT,
			&datum.ANLAGE,
		); err != nil {
			return nil, err
		}

		data = append(data, datum)
	}

	log.Printf("[Result] %d records\n", len(data))

	return data, nil
}
