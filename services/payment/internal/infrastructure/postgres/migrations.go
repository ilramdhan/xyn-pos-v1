package postgres

import "embed"

//go:embed migrations/*.sql

// MigrationsFS embeds all SQL migration files for the payment service.
var MigrationsFS embed.FS
