# ADR-0002: Kafka 4.3 KRaft Mode — ZooKeeper Removed

- **Status**: Accepted
- **Date**: 2026-06-05
- **Deciders**: Principal Engineer

---

## Context

Kafka 4.0+ removes ZooKeeper entirely. KRaft (Kafka Raft Metadata) mode is now the only supported deployment model. This is a **breaking change** from Kafka 3.x docker-compose configurations.

---

## Decision

Use **Kafka 4.3.0 in KRaft mode**. All docker-compose and K8s manifests must NOT include ZooKeeper. Use the `KAFKA_NODE_ID`, `KAFKA_PROCESS_ROLES`, and `KAFKA_CONTROLLER_QUORUM_VOTERS` environment variables.

---

## Consequences

**docker-compose change — KRaft config:**

```yaml
kafka:
  image: apache/kafka:4.3.0
  environment:
    KAFKA_NODE_ID: 1
    KAFKA_PROCESS_ROLES: broker,controller
    KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
    KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
    KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
    KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
    KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
    KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    KAFKA_LOG_DIRS: /var/lib/kafka/data
    CLUSTER_ID: MkU3OEVBNTcwNTJENDM2Qk
```

**Removed — NO ZooKeeper service in docker-compose.**

See `docs/phase-1/deployment-guide.md` for the full configuration.
