package rfc

import (
	"net/http"
)

type RFCInterface interface {
	IsValidCandidate(req *http.Request) bool
}
