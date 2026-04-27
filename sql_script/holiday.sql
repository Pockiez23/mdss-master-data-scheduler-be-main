DROP TABLE IF EXISTS "mst"."pea_holiday";
-- Sequence and defined type
CREATE SEQUENCE IF NOT EXISTS mst.holiday_id_seq;

-- Table Definition
CREATE TABLE "mst"."pea_holiday" (
    "id" int4 NOT NULL DEFAULT nextval('mst.holiday_id_seq'::regclass),
    "date" date NOT NULL,
    "day" varchar NOT NULL,
    "peak_offpeak" varchar NOT NULL,
    "name" varchar NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "created_by" varchar,
    "updated_at" timestamp,
    "updated_by" varchar,
    PRIMARY KEY ("id")
);