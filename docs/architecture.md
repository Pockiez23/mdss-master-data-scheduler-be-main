# Architecture Deep-Dive

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                  mdss-master-data-scheduler-be               │
│                                                             │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │  Cron    │───▶│  multiply/   │───▶│  Redis (cache)   │  │
│  │ Scheduler│    │  Service     │    │  HashValue       │  │
│  │ (*/15min)│    │              │    │  IsThreePhase    │  │
│  └──────────┘    └──────┬───────┘    │  latest_offset   │  │
│       │                 │            └──────────────────┘  │
│       │                 │ changed?                          │
│  ┌────▼─────┐           ▼                                   │
│  │ holiday/ │    ┌──────────────┐                           │
│  │ Service  │    │  Kafka       │                           │
│  │ (3:15am) │    │  Producer    │                           │
│  └────┬─────┘    └──────────────┘                           │
│       │                                                     │
│  ┌────▼─────────────────────────────────────────────────┐   │
│  │                  REST API (Gin)                       │   │
│  │  GET /api/jobs                                        │   │
│  │  GET /api/contract-accounts/:id                      │   │
│  │  GET /api/holidays                                    │   │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
  ┌────────────┐      ┌─────────────┐     ┌──────────────┐
  │  SAP HANA  │      │ PostgreSQL  │     │    Kafka     │
  │ (ZDMI data)│      │ (holidays)  │     │  (events)    │
  └────────────┘      └─────────────┘     └──────────────┘
```

---

## Fetch Master Data from SAP HANA — Detailed Flow

### Two Query Modes

```
FetchMasterData() called
        │
        ▼
  Redis: GetLatestOffset("multiplier")
        │
        ├── offset is zero/Nil ──▶  FetchAll()  (no WHERE on PROCESSDATE)
        │
        └── offset exists ─────▶  Fetch(offset) (WHERE PROCESSDATE >= ?, PROCESSTIME >= ?)
```

### SAP HANA SQL Query

Both `FetchAll` and `Fetch` run the **same JOIN** — only `Fetch` adds the offset filter.

```sql
SELECT
    zdm_fu_zdmi025.MANDT,      -- tenant/client ID
    EQUNR,                      -- equipment number (meter device)
    SERNR,                      -- serial number = PEA No
    BIS,                        -- end date (YYYYMMDD)
    AB,                         -- start date (YYYYMMDD)
    PROCESSDATE,                -- last processed date (YYYYMMDD)
    PROCESSTIME,                -- last processed time (HHMMSS)
    VKONT,                      -- contract account number
    CAST(BILL_FACTOR AS FLOAT), -- meter multiplier
    meter_master.FUNKLAS,       -- meter type code (e.g. M030)
    meter_type.FUNKTXT,         -- meter type description
    ANLAGE                      -- installation ID

FROM "INFUSER"."REP_RT_ZDM_FU_ZDMI025" AS zdm_fu_zdmi025

INNER JOIN "INFUSER"."REP_RT_METER_MASTER" AS meter_master
    ON  zdm_fu_zdmi025.MANDT = meter_master.MANDT
    AND LTRIM(meter_master.MATNR, '0') = zdm_fu_zdmi025.MATNR
    --  ↑ strips leading zeros from MATNR before matching

INNER JOIN "INFUSER"."REP_RT_METER_TYPE" AS meter_type
    ON  meter_master.MANDT = meter_type.MANDT
    AND meter_master.FUNKLAS = meter_type.FUNKLAS

WHERE
    -- Incremental filter (Fetch only, not FetchAll):
    PROCESSDATE >= ?   -- format: YYYYMMDD  (offset from Redis)
    AND PROCESSTIME >= ? -- format: HHMMSS

    -- Data quality filters (both modes):
    AND SERNR        IS NOT NULL AND SERNR        <> ''
    AND BIS          IS NOT NULL AND BIS          <> ''
    AND AB           IS NOT NULL AND AB           <> ''
    AND PROCESSDATE  IS NOT NULL AND PROCESSDATE  <> ''
    AND PROCESSTIME  IS NOT NULL AND PROCESSTIME  <> ''
    AND VKONT        IS NOT NULL AND VKONT        <> ''
    AND BILL_FACTOR  IS NOT NULL
```

### Table Relationships

```
REP_RT_ZDM_FU_ZDMI025 (zdm_fu_zdmi025)
│  MANDT, MATNR, EQUNR, SERNR, BIS, AB
│  PROCESSDATE, PROCESSTIME, VKONT, BILL_FACTOR, ANLAGE
│
├── INNER JOIN REP_RT_METER_MASTER (meter_master)
│     ON MANDT = MANDT
│     AND LTRIM(MATNR,'0') = MATNR    ← zero-stripped key
│     Provides: FUNKLAS (meter type code)
│
└── INNER JOIN REP_RT_METER_TYPE (meter_type)
      ON MANDT = MANDT
      AND FUNKLAS = FUNKLAS
      Provides: FUNKTXT (meter type description)
```

### ZDMI → HashValue Mapping

```
ZDMI (raw from HANA)                HashValue (stored in Redis)
─────────────────────────────────   ──────────────────────────────────────
VKONT          ──────────────────▶  Redis hash key:  {PREFIX}:{VKONT}
MANDT+EQUNR+VKONT+BIS (base64) ──▶  Redis hash field (ZDMI.Field())
BILL_FACTOR    ──────────────────▶  HashValue.Multiplier   (float64)
SERNR          ──────────────────▶  HashValue.PEANo        (string)
ANLAGE         ──────────────────▶  HashValue.Installation (string)
AB  (YYYYMMDD) ──────▶ Unix(+0700)▶  HashValue.StartDate   (int64)
BIS (YYYYMMDD) ──────▶ Unix(+0700)▶  HashValue.EndDate     (int64)
FUNKLAS ∈ threePhaseMeterTypes? ──▶  HashValue.IsThreePhase (bool)
```

**Three-phase meter type codes** (hardcoded in [model/zdmi.go:15](../internal/model/zdmi.go#L15)):
```
M030, M040, M050, M060, M080
```

### Post-Fetch Processing (per record, parallel via errgroup)

```
For each ZDMI record (goroutine):
        │
        ├─1─▶ Convert to HashValue (parse dates, check IsThreePhase)
        │
        ├─2─▶ recal.CheckMasterDataExisting(ctx, doc)
        │       └── Compare Redis stored StartDate/EndDate vs new values
        │           Returns []RecalculationRequest if dates changed
        │
        ├─3─▶ rediz.Upsert(ctx, VKONT, Field(), HashValue)
        │       └── HSET {PREFIX}:{VKONT}  <field>  <json>
        │
        ├─4─▶ rediz.SetGlobalThreePhase(ctx, "{VKONT}:is_three_phase", bool)
        │       └── SET {PREFIX}:{VKONT}:is_three_phase  "true"/"false"
        │
        └─5─▶ (if diff found) kafka.Produces(RecalculationRequest messages)
                └── Topic: KAFKA_PRODUCER_TOPIC_METER_RECALCULATION_RAW
```

### Redis Key Structure

```
{REDIS_PREFIX}:{VKONT}
    └── Hash: field=base64(MANDT|EQUNR|VKONT|BIS)
              value={"multiplier":X,"pea_no":"...","installation":"...",
                     "start_date":unix,"end_date":unix,"is_three_phase":bool}

{REDIS_PREFIX}:{VKONT}:is_three_phase
    └── String: "true" or "false"

{REDIS_PREFIX}:multiplier:latest_offset
    └── String: RFC3339 timestamp of last successful sync

{REDIS_PREFIX}:holiday:latest_offset
    └── String: RFC3339 timestamp of last holiday sync
```

### Offset / Incremental Sync Logic

```
Startup (no offset in Redis):
  GetLatestOffset → redis.Nil
    └── FetchAll() from HANA (full scan, no date filter)

Subsequent runs (offset exists):
  GetLatestOffset → time.Time
    └── Fetch(offset) from HANA
          WHERE PROCESSDATE >= YYYYMMDD AND PROCESSTIME >= HHMMSS

After success:
  SetLatestOffset(currentOffset)   ← time.Now() captured at start of job
```

> **Gotcha**: `currentOffset` is captured at job start, not after the query. If the
> job takes a long time, records processed during that window with newer PROCESSDATE
> will be re-fetched on the next run (safe but redundant).

---

## Holiday Sync Flow

```
PostgreSQL mst.pea_holiday
  WHERE peak_offpeak = 'OFFPEAK'
    AND name IS NOT NULL AND name <> ''
    AND (created_at >= offset OR updated_at >= offset)   ← incremental
        │
        ▼
  Redis: SET/cache holiday dates
        │
        ▼
  SetLatestOffset("holiday", currentOffset)
```

---

## Kafka Event Schema

Topic: `KAFKA_PRODUCER_TOPIC_METER_RECALCULATION_RAW`

```json
{
  "contract_account": "VKONT_VALUE",
  "date": "YYYY-MM-DD"     // date when master data change was detected
}
```

Trigger: start date or end date of a ZDMI record differs from the previously cached `HashValue`.

---

Last updated: 2026-03-05 | Generator: AI analysis
