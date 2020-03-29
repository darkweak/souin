package providers

import (
	"os"
	"regexp"

	"github.com/darkweak/souin/cache/types"
)

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	GetRequestInCache(key string) types.ReverseResponse
	SetRequestInCache(key string, value []byte)
	DeleteRequestInCache(key string)
	Init()
}

// PathnameNotInRegex check if pathname is in parameter regex var
func PathnameNotInRegex(pathname string) bool {
	b, _ := regexp.Match(os.Getenv("REGEX"), []byte(pathname))
	return !b
}
