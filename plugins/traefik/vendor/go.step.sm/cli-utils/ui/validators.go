package ui

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/manifoldco/promptui"
)

var errEmptyValue = errors.New("value is empty")

// NotEmpty is a validation function that checks that the prompted string is not
// empty.
func NotEmpty() promptui.ValidateFunc {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errEmptyValue
		}
		return nil
	}
}

// Address is a validation function that checks that the prompted string is a
// valid TCP address.
func Address() promptui.ValidateFunc {
	return func(s string) error {
		if _, _, err := net.SplitHostPort(s); err != nil {
			return fmt.Errorf("%s is not an TCP address", s)
		}
		return nil
	}
}

// IPAddress is validation function that checks that the prompted string is a
// valid IP address.
func IPAddress() promptui.ValidateFunc {
	return func(s string) error {
		if net.ParseIP(s) == nil {
			return fmt.Errorf("%s is not an ip address", s)
		}
		return nil
	}
}

// DNS is a validation function that checks that the prompted string is a valid
// DNS name or IP address.
func DNS() promptui.ValidateFunc {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errEmptyValue
		}
		if ip := net.ParseIP(s); ip != nil {
			return nil
		}
		if _, _, err := net.SplitHostPort(s + ":443"); err != nil {
			return fmt.Errorf("%s is not a valid DNS name or IP address", s)
		}
		return nil
	}
}

// YesNo is a validation function that checks for a Yes/No answer.
func YesNo() promptui.ValidateFunc {
	return func(s string) error {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "y", "yes", "n", "no":
			return nil
		default:
			return fmt.Errorf("%s is not a valid answer", s)
		}
	}
}
