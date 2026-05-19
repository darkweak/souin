package convert

import (
	"fmt"
	"math"
)

type integer interface {
	~int | ~int64
}

// SafeUint64 converts an integer value to [uint64]. It returns an error if the value is out of range.
func SafeUint64[T integer](x T) (uint64, error) {
	if x < 0 {
		return 0, fmt.Errorf("value %d out of range for uint64", x)
	}

	return uint64(x), nil
}

// MustUint64 converts an integer value to [uint64]. It panics if the value is out of range.
func MustUint64[T integer](x T) uint64 {
	u64, err := SafeUint64(x)
	if err != nil {
		panic(err)
	}

	return u64
}

// SafeUint32 converts an integer value to [uint32]. It returns an error if the value is out of range.
func SafeUint32[T integer](x T) (uint32, error) {
	if x < 0 || int64(x) > math.MaxUint32 {
		return 0, fmt.Errorf("value %d out of range for uint32", x)
	}

	return uint32(x), nil
}

// MustUint32 converts an integer value to [uint32]. It panics if the value is out of range.
func MustUint32[T integer](x T) uint32 {
	u32, err := SafeUint32(x)
	if err != nil {
		panic(err)
	}

	return u32
}

// MustUint16 converts an integer value to [uint16]. It panics if the value is out of range.
func MustUint16(x int) uint16 {
	if x < 0 || x > math.MaxUint16 {
		panic(fmt.Errorf("value %d out of range for uint16", x))
	}

	return uint16(x)
}

// SafeUint8 converts an integer value to [uint8]. It returns an error if the value is out of range.
func SafeUint8(x int) (uint8, error) {
	if x < 0 || x > math.MaxUint8 {
		return 0, fmt.Errorf("value %d out of range for uint8", x)
	}

	return uint8(x), nil
}

// MustUint8 converts an integer value to [uint8]. It panics if the value is out of range.
func MustUint8(x int) uint8 {
	u8, err := SafeUint8(x)
	if err != nil {
		panic(err)
	}

	return u8
}
