package urlpattern

// https://urlpattern.spec.whatwg.org/#options-header
type options struct {
	// MUST be an ASCII scode point
	delimiterCodePoint byte
	prefixCodePoint    byte
	ignoreCase         bool
}
