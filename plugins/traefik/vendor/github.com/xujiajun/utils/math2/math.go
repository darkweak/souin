package math2

// Source: https://gist.github.com/petergloor/1eacd3ebcfc3e09ed578dd0d4f80cabe

const (
	MaxUint = ^uint(0)
	MinUint = 0

	MaxInt = int(^uint(0) >> 1)
	MinInt = -MaxInt - 1
)
