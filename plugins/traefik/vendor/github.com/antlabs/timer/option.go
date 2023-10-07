package timer

type option struct {
	timeWheel bool
	minHeap   bool
	skiplist  bool
	rbtree    bool
}

type Option func(c *option)

func WithTimeWheel() Option {
	return func(o *option) {
		o.timeWheel = true
	}
}

func WithMinHeap() Option {
	return func(o *option) {
		o.minHeap = true
	}
}

//TODO
func WithSkipList() Option {
	return func(o *option) {
		o.skiplist = true
	}
}

//TODO
func WithRbtree() Option {
	return func(o *option) {
		o.rbtree = true
	}
}
