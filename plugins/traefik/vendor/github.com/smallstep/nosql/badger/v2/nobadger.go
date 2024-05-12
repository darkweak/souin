//go:build nobadger || nobadgerv2
// +build nobadger nobadgerv2

package badger

import "github.com/smallstep/nosql/database"

type DB = database.NotSupportedDB
