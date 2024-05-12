//go:build nomysql
// +build nomysql

package mysql

import "github.com/smallstep/nosql/database"

type DB = database.NotSupportedDB
