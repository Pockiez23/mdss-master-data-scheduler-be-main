# Master Data Scheduler

## Environment variables

| Key                                          | Description                                                     | Type    | Required | Default           | Example                                             |
| -------------------------------------------- | --------------------------------------------------------------- | ------- | -------- | ----------------- | --------------------------------------------------- |
| KAFKA_BROKERS                                | Kafka brokers สามารถใส่ได้หลายตัวโดยคั่นด้วย comma (,)                | String  | ✔        | -                 | 10.1.0.41:9095                                      |
| KAFKA_VERSION                                | เวอร์ชันของ Kafka server                                          | String  | ✔        | 3.7.2             | 3.7.2                                               |
| KAFKA_CLIENT_ID                              | Identity ของ client แต่ละตัว                                      | String  | ✘        | sasl_scram_client | client                                              |
| KAFKA_NET_SASL_ENABLE                        | Enable SASL                                                     | Boolean | ✘        | -                 | true                                                |
| KAFKA_NET_SASL_USERNAME                      | Username                                                        | String  | ✘        | -                 |                                                     |
| KAFKA_NET_SASL_PASSWORD                      | Password                                                        | String  | ✘        | -                 |                                                     |
| KAFKA_NET_SASL_HANDSHAKE                     | Handshake SASL                                                  | Boolean | ✘        | true              | true                                                |
| KAFKA_NET_SASL_MECHANISM                     | Mechanism ของ SASL                                              | String  | ✘        | -                 | PLAIN                                               |
| KAFKA_NET_TLS_ENABLE                         | Enable TLS                                                      | Boolean | ✘        | -                 | true                                                |
| KAFKA_NET_TLS_CA_PATH                        | Path ของไฟล์ ca                                                  | String  | ✘        | -                 |                                                     |
| KAFKA_NET_TLS_INSECURE_SKIP_VERIFY           | ละเว้นการตรวจสอบ TLS                                             | Boolean | ✘        | -                 | false                                               |
| KAFKA_PRODUCER_TOPIC_METER_RECALCULATION_RAW | ชื่อ topic สำหรับ meter recalculation raw                           | String  | ✘        | -                 | meter.recalculation.raw                             |
| KAFKA_PRODUCER_RETRY_MAX                     | จำนวนครั้งที่จะ retry เมื่อ produce ไม่สำเร็จ                             | Numeric | ✘        | 1                 | 3                                                   |
| KAFKA_CONSUMER_TOPICS                        | ชื่อ topics ที่จะรับข้อมูล สามารถใส่ได้หลาย topics โดยคั่นด้วย comma (,)    | String  | ✘        | -                 | some.data.raw                                       |
| KAFKA_CONSUMER_ENABLE_DLQ                    | เปิดการทำงานของ DLQ                                               | Boolean | ✘        | -                 | true                                                |
| KAFKA_CONSUMER_GROUP_ID                      | ระบุ id ของ consumer group                                       | String  | ✘        | -                 |                                                     |
| KAFKA_CONSUMER_OFFSETS_INITIAL               | ระบุว่าให้ consumer เริ่มดึงจาก offset ไหน                            | String  | ✘        | NEWEST            | OLDEST                                              |
| KAFKA_CONSUMER_OFFSETS_AUTO_COMMIT           | เปิดการใช้งาน Auto commit                                         | Boolean | ✘        | true              | true                                                |
| KAFKA_CONSUMER_GROUP_REBALANCE_STRATEGY      | รูปแบบการจัดการการ rebalance ของ consumer group                   | String  | ✘        | STICKY            | STICKY                                              |
| DATABASE_HANA_DSN                            | Data source name ของ HANA                                       | ✔       | String   | -                 |                                                     |
| DATABASE_PG_DSN                              | Data source name ของ PostgreSQL ที่ใช้เก็บข้อมูล เช่น Holidays         | ✔       | String   | -                 | postgresql://user:pass@localhost:5432?database=test |
| REDIS_MASTER_NAME                            | ชื่อของ master node สำหรับ sentinel หากไม่กรอกจะต่อแบบ Standalone แทน | String  |          | -                 |                                                     |
| REDIS_ADDRESSES                              | IPs ของ redis สามารถใส่ได้หลาย ip โดยคั่นด้วย comma (,)              | String  | ✔        | -                 |                                                     |
| REDIS_USERNAME                               | Username ของ redis                                              | String  |          | -                 |                                                     |
| REDIS_PASSWORD                               | Password ของ redis                                              | String  |          | -                 |                                                     |
| REDIS_SENTINEL_USERNAME                      | Username ของ redis sentinel                                     | String  |          | -                 |                                                     |
| REDIS_SENTINEL_PASSWORD                      | Password ของ redis sentinel                                     | String  |          | -                 |                                                     |
| REDIS_DB                                     | DB ของ redis                                                    | Numeric |          | 0                 | 1                                                   |
| REDIS_PREFIX                                 | Prefix key                                                      | String  | ✔        | -                 |                                                     |
| REDIS_POOL_SIZE                              | ขนาดของ connection pool                                         | Numeric | ✘        | 60                |                                                     |
| REDIS_POOL_TIMEOUT_IN_SECOND                 | เวลาของ connection pool timeout                                 | Numeric | ✘        | 300               |                                                     |
| REDIS_MIN_IDLE_CONNECTION                    | จำนวนขั้นต่ำของ connection ที่จะเปิดไว้รอ                                | Numeric | ✘        | 10                |                                                     |
| JOB_MASTER_DATA_CRON                         | Crontab ของการทำงาน master data job                              | String  | ✔        | */15 \* \* \* \*  |                                                     |
| JOB_HOLIDAY_CRON                             | Crontab ของการทำงาน sync holidays job                            | String  | ✔        | 15 3 \* \* \*     |                                                     |
