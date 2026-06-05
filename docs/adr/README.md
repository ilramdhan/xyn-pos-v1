# Architecture Decision Records (ADR)

This directory contains Architecture Decision Records for `xyn-pos-v1`.

## What is an ADR?

An ADR documents a significant architectural decision: the context, the options considered, the decision made, and the consequences. They serve as a permanent record of *why* the system is built the way it is.

## Format

Each ADR uses this structure:

```
# ADR-XXXX: [Short Decision Title]

- **Status**: Proposed | Accepted | Deprecated | Superseded by ADR-XXXX
- **Date**: YYYY-MM-DD
- **Deciders**: [Names or roles]

## Context
What forces are at play? What problem are we solving?

## Decision
What was decided?

## Options Considered
Brief comparison of alternatives evaluated.

## Consequences
What becomes easier or harder after this decision?
```

## Index

| # | Title | Status | Date |
|---|---|---|---|
| [ADR-0001](0001-manual-di-over-wire.md) | Manual DI over Google Wire | Accepted | 2026-06-05 |
| [ADR-0002](0002-kafka-kraft-mode.md) | Kafka 4.3 KRaft — ZooKeeper Removed | Accepted | 2026-06-05 |
| [ADR-0003](0003-hybrid-multi-tenancy.md) | Hybrid Multi-Tenancy Strategy | Accepted | 2026-06-05 |

## Naming Convention

`{sequence}-{short-kebab-title}.md`

Sequence is zero-padded to 4 digits. Never reuse a number, even if an ADR is deprecated.
