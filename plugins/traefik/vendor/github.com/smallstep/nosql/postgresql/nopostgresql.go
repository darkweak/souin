//go:build nopgx
// +build nopgx

package postgresql

import "github.com/smallstep/nosql/database"

type DB = database.NotSupportedDB
