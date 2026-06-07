package postgres

import "embed"

//go:embed migrations/*.sql

// MigrationsFS embeds all SQL migration files for the pos service.
var MigrationsFS embed.FS
