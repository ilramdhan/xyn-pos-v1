package postgres

import "embed"

//go:embed migrations/*.sql

// MigrationsFS embeds all SQL migration files for the tenant service.
var MigrationsFS embed.FS
