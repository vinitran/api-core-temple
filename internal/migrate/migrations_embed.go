package migrate

import "embed"

//go:embed sql/*.sql
var migrationsFS embed.FS

const migrationsDir = "sql"
