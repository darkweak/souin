package caddy

type defaultCacheError struct{}

func (s *defaultCacheError) Error() string {
	return "Invalid/Incomplete default cache declaration"
}
