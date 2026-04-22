# AGENTS.md

## Project Overview
- Scheduled backend service that syncs meter master data (ZDMI) from SAP HANA and holiday data from PostgreSQL into Redis, then publishes Kafka events when data changes.
- Tech stack: Go 1.25.1, Gin (HTTP), cron/v3 (scheduling), GORM (PostgreSQL ORM), Sarama (Kafka), go-redis v9 (cache)
- Microservice within the Archon meter-broker platform; downstream consumers use Kafka recalculation events and Redis-cached multipliers.

## Architecture
- **Pattern**: Layered — Handler → Service → Repository, plus cron-driven job execution
- **Key modules**:
  - `scheduler/` — cron job registration; runs master data and holiday sync
  - `multiply/` — fetches ZDMI from HANA, diffs against Redis, publishes Kafka events
  - `holiday/` — fetches holidays from PostgreSQL, caches in Redis
  - `recal/` — compares stored vs new start/end dates to detect changes
  - `rediz/` — Redis hash/string operations and offset tracking
  - `kafka/` — Kafka SyncProducer wrapper
  - `locker/` — in-memory mutex preventing duplicate cron execution
  - `config/` — env-driven config structs for all external systems
  - `model/` — domain structs: ZDMI, HashValue, Holiday, RecalculationRequest
- **Data flow**: Cron trigger → FetchAll/Fetch(HANA) → diff via Redis → Upsert Redis → Produce Kafka
- **External systems**: SAP HANA/SQL Server, PostgreSQL, Apache Kafka, Redis (Sentinel-capable)

See [docs/architecture.md](docs/architecture.md) for diagrams and deep-dive details.

## Directory Structure
```
cmd/server/          # main.go — app entry point
internal/
  config/            # env parsing for Kafka, Redis, DB, job timing
  global/            # global cron instance, Kafka topic vars, ErrLocked
  scheduler/         # cron job handler (master data + holiday sync)
  job/               # REST API handlers (3 endpoints)
  multiply/          # HANA fetch, Redis upsert, Kafka publish
  holiday/           # PostgreSQL fetch, Redis cache
  recal/             # change detection (date-based diff)
  rediz/             # Redis repository (hash, string, offset ops)
  kafka/             # Kafka producer service
  locker/            # in-memory per-task lock map
  helper/            # root path, timezone (Asia/Bangkok)
  model/             # ZDMI, HashValue, Holiday, RecalculationRequest
docs/                # architecture diagrams and deep-dive docs
```

## Commands
```bash
# Dev
cp .env.example .env   # fill in required vars first
go run cmd/server/main.go

# Build
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app cmd/server/main.go

# Test
go test ./...
go test ./internal/holiday/...          # single package
go test -run TestServiceName ./...      # single test

# Lint
golangci-lint run

# Generate mocks (requires mockery CLI)
mockery
```

## Coding Conventions
- **Interfaces**: always prefixed `I` — `IService`, `IRepository`, `IHandler`
- **Packages**: lowercase, domain-named — `multiply`, `holiday`, `recal`, `rediz`
- **Constants**: `UPPER_SNAKE_CASE` (e.g., `IS_THREE_PHASE`)
- **Error wrapping**: `errors.Join(errors.New("context label"), err)` — never bare `err`
- **Redis Nil**: treated as zero-value (no offset yet), not as an error
- **Logging**: stdlib `log` only; prefix every log line with `[TaskName]` (e.g., `[Redis]`, `[Query]`)
- **Concurrency**: use `errgroup.Group` for fan-out; use `locker` package before any job, not raw mutexes
- **Closure capture**: always copy loop variable with `doc := datum` before `group.Go()`
- **No structured logging** — do not introduce zap/logrus without discussion

## Domain Terminology
| Term | Meaning / Code Location |
|------|------------------------|
| ZDMI | Master meter data row; struct `model.ZDMI`; source table `INFUSER.REP_RT_ZDM_FU_ZDMI025` |
| MANDT | SAP client/tenant ID; part of Redis hash field key |
| EQUNR | Equipment number (meter device ID) |
| SERNR | Serial number = PEA No (meter unique ID); maps to `HashValue.PEANo` |
| VKONT | Contract account number; Redis hash key and Kafka message subject |
| BILL_FACTOR | Meter multiplier (cast to float64); maps to `HashValue.Multiplier` |
| AB / BIS | Start / End date in `YYYYMMDD`; stored as Unix timestamps in Redis |
| FUNKLAS | Meter type code; determines `IsThreePhase` (M030/M040/M050/M060/M080) |
| FUNKTXT | Meter type description text |
| ANLAGE | Installation ID; maps to `HashValue.Installation` |
| HashValue | Redis-cached representation of one ZDMI row; JSON in Redis hash field |
| Offset | Last sync timestamp stored in Redis; key: `{PREFIX}:multiplier:latest_offset` |
| RecalculationRequest | Kafka event sent when master data dates change; contains VKONT + date |
| Holiday | Off-peak holiday date from `mst.pea_holiday`; cached as Redis set |

## Key Constraints & Gotchas
- **Jobs run at startup AND on cron schedule** — both `MasterDataSync` and `SyncHolidays` are called immediately in `main.go` before cron registration. Avoid slow startup or double-sync races.
- **`locker` is NOT distributed** — in-memory only; two replicas can both run the same job.
- **First run fetches ALL HANA rows** (no offset yet); subsequent runs are incremental by `PROCESSDATE >= ? AND PROCESSTIME >= ?`.
- **Three-phase types are hardcoded**: `["M030","M040","M050","M060","M080"]` in `model/zdmi.go:15`. Changes require recompile.
- **Timezone hardcoded to Asia/Bangkok** in `helper/location.go`; dates parsed as `+0700`.
- **No graceful shutdown** — Kafka producer and DB connections are not closed on SIGTERM.
- **Redis prefix collisions** — `REDIS_PREFIX` is not validated; two services with the same prefix corrupt each other's data.
- **HANA queries have no LIMIT** — full table scans on cold start; can be large.
- **No connection pool config for HANA** — uses raw `sql.Open()` defaults.

## Testing
- Framework: `testify` (assert + mock); mocks generated by `mockery` (`/.mockery.yml`)
- Test files: `internal/holiday/service_test.go`, `internal/recal/service_test.go`
- Mock dirs: `internal/*/mocks/`
- Run all: `go test ./...`
- Run specific package: `go test ./internal/recal/...`
- No external services needed for unit tests (all dependencies mocked)

## Dependencies & Integrations
| System | Purpose | Auth |
|--------|---------|------|
| SAP HANA / SQL Server | Source of ZDMI master meter data | DSN in `DATABASE_HANA_DSN` |
| PostgreSQL | Source of holiday data (`mst.pea_holiday`) | DSN in `DATABASE_PG_DSN` |
| Apache Kafka | Publish meter recalculation events | SASL (PLAIN/SCRAM-SHA256/SHA512) + optional TLS |
| Redis (Sentinel) | Cache multipliers, three-phase flags, offsets | Username/password; Sentinel if `REDIS_MASTER_NAME` set |

**Required env vars**: `KAFKA_BROKERS`, `DATABASE_HANA_DSN`, `DATABASE_PG_DSN`, `REDIS_ADDRESSES`, `REDIS_PREFIX`, `JOB_MASTER_DATA_CRON`, `JOB_HOLIDAY_CRON`

---
Last updated: 2026-03-05 | Generator: AI analysis
