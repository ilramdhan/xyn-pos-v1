-- ClickHouse 26.5 — Database & schema initialization
-- WHY ClickHouse? Append-only analytics workloads — order history, revenue aggregations,
-- kitchen throughput, inventory turnover. 100x faster than PostgreSQL for GROUP BY + aggregations.

CREATE DATABASE IF NOT EXISTS xyn_analytics
    ENGINE = Atomic
    COMMENT 'Analytics data from all microservices via Kafka CDC';

CREATE DATABASE IF NOT EXISTS xyn_analytics_staging
    ENGINE = Atomic
    COMMENT 'Analytics staging — mirrors prod, used for testing BI queries';
