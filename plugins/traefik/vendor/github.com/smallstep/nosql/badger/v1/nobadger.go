//go:build nobadger || nobadgerv1
// +build nobadger nobadgerv1

package badger

import "github.com/smallstep/nosql/database"

type DB = database.NotSupportedDB
