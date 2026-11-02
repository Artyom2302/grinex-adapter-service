// Package migrations embeds the SQL migration files.
package migrations

import "embed"

// Migrations holds the embedded *.sql files.
//
//go:embed *.sql
var Migrations embed.FS
