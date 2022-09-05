//go:build generate

// this file exists to track tool dependency without using them during build

package tools

import (
	_ "github.com/kyleconroy/sqlc/cmd/sqlc"
)

//go:generate go run github.com/kyleconroy/sqlc/cmd/sqlc generate --file assets/sql/sqlc.yaml
