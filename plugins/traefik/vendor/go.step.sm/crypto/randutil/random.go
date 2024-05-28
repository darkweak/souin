// Package randutil provides methods to generate random strings and salts.
package randutil

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"math/big"

	"github.com/pkg/errors"
)

var ascii string

func init() {
	// initialize the charcters in ascii
	aciiBytes := make([]byte, 94)
	for i := range aciiBytes {
		aciiBytes[i] = byte(i + 33)
	}
	ascii = string(aciiBytes)
}

// Salt generates a new random salt of the given size.
func Salt(size int) ([]byte, error) {
	salt := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return nil, errors.Wrap(err, "error generating salt")
	}
	return salt, nil
}

// Bytes generates a new byte slice of the given size.
func Bytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return nil, errors.Wrap(err, "error generating bytes")
	}
	return bytes, nil
}

// String returns a random string of a given length using the characters in
// the given string. It splits the string on runes to support UTF-8
// characters.
func String(length int, chars string) (string, error) {
	result := make([]rune, length)
	runes := []rune(chars)
	x := int64(len(runes))
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(x))
		if err != nil {
			return "", errors.Wrap(err, "error creating random number")
		}
		result[i] = runes[num.Int64()]
	}
	return string(result), nil
}

// Hex returns a random string of the given length using the hexadecimal
// characters in lower case (0-9+a-f).
func Hex(length int) (string, error) {
	return String(length, "0123456789abcdef")
}

// Alphanumeric returns a random string of the given length using the 62
// alphanumeric characters in the POSIX/C locale (a-z+A-Z+0-9).
func Alphanumeric(length int) (string, error) {
	return String(length, "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// ASCII returns a securely generated random ASCII string. It reads random
// numbers from crypto/rand and searches for printable characters. It will
// return an error if the system's secure random number generator fails to
// function correctly, in which case the caller must not continue.
func ASCII(length int) (string, error) {
	return String(length, ascii)
}

// Alphabet returns a random string of the given length using the 52
// alphabetic characters in the POSIX/C locale (a-z+A-Z).
func Alphabet(length int) (string, error) {
	return String(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// UUIDv4 returns the string representation of a UUID version 4. Because 6 bits
// are used to indicate the version 4 and the variant 10, the randomly generated
// part has 122 bits.
func UUIDv4() (string, error) {
	var uuid [16]byte
	_, err := io.ReadFull(rand.Reader, uuid[:])
	if err != nil {
		return "", errors.Wrap(err, "error generating uuid")
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	return encodeUUID(uuid), nil
}

func encodeUUID(uuid [16]byte) string {
	buf := make([]byte, 36)
	hex.Encode(buf, uuid[:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], uuid[10:])
	return string(buf)
}
