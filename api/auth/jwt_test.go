package auth

import (
	"fmt"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckToken(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	security := InitializeSecurity(config)
	r := httptest.NewRequest("GET", fmt.Sprintf("http://%s%s/valid_path", config.GetAPI().BasePath, security.basePath), nil)
	w := httptest.NewRecorder()
	jwt, err := CheckToken(security, w, r)
	if w.Result().StatusCode != http.StatusUnauthorized {
		errors.GenerateError(t, "Status code should be 401")
	}
	if jwt != nil {
		errors.GenerateError(t, "Token should be nil")
	}
	if err == nil {
		errors.GenerateError(t, "It should throw an error")
	} else if err.Error() != "An error occurred, Token not found" {
		errors.GenerateError(t, fmt.Sprintf("Error isn't equals to expected, %s given", err.Error()))
	}

	// Invalid token
	http.SetCookie(w, &http.Cookie{Name: tests.GetTokenName(), Value: "badvalue"})
	r = &http.Request{
		Header: http.Header{
			"Cookie": w.HeaderMap["Set-Cookie"],
		},
	}
	w = httptest.NewRecorder()
	jwt, err = CheckToken(security, w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		errors.GenerateError(t, "Status code should be 400")
	}
	if jwt == nil {
		errors.GenerateError(t, "Token shouldn't be nil")
	}
	if jwt.Username != "" {
		errors.GenerateError(t, "Token username shouldn't exist")
	}
	if jwt.ExpiresAt != 0 {
		errors.GenerateError(t, "Token expiration shouldn't be set")
	}
	if err == nil {
		errors.GenerateError(t, "It should throw an error")
	} else if err.Error() != "An error occurred, Impossible to sign the JWT" {
		errors.GenerateError(t, fmt.Sprintf("Error isn't equals to expected, %s given", err.Error()))
	}

	// Invalid signature token
	http.SetCookie(w, &http.Cookie{
		Name:  tests.GetTokenName(),
		Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIxIiwiZXhwIjoxNjE0MTI0ODd9.wwIwMd36VaQ6CYNdgG7LiZ_zZ95tAz157zEoDz4Etl0",
		Path:  "/",
	})
	r = &http.Request{
		Header: http.Header{
			"Cookie": w.HeaderMap["Set-Cookie"],
		},
	}
	w = httptest.NewRecorder()
	jwt, err = CheckToken(security, w, r)
	if w.Result().StatusCode != http.StatusUnauthorized {
		errors.GenerateError(t, "Status code should be 401")
	}
	if jwt == nil {
		errors.GenerateError(t, "Token shouldn't be nil")
	}
	if jwt.Username != "user1" {
		errors.GenerateError(t, "Token username should exist")
	}
	if jwt.ExpiresAt == 0 {
		errors.GenerateError(t, "Token expiration should be set and different than 0")
	}
	if err == nil {
		errors.GenerateError(t, "It should throw an error")
	} else if err.Error() != "An error occurred, Impossible to sign the JWT" {
		errors.GenerateError(t, fmt.Sprintf("Error isn't equals to expected, %s given", err.Error()))
	}

	// Expired token
	http.SetCookie(w, &http.Cookie{
		Name:  tests.GetTokenName(),
		Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIxIiwiZXhwIjoxNjE0MTI0N30.Fnr0YYPqZvZQ_6brZo_Ax7VEUkTfZVaJ8Fm-jAYb8Kc",
		Path:  "/",
	})
	r = &http.Request{
		Header: http.Header{
			"Cookie": w.HeaderMap["Set-Cookie"],
		},
	}
	w = httptest.NewRecorder()
	jwt, err = CheckToken(security, w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		errors.GenerateError(t, "Status code should be 400")
	}
	if jwt == nil {
		errors.GenerateError(t, "Token shouldn't be nil")
	}
	if jwt.Username != "user1" {
		errors.GenerateError(t, "Token username should exist")
	}
	if jwt.ExpiresAt == 0 {
		errors.GenerateError(t, "Token expiration should be set and different than 0")
	}
	if err == nil {
		errors.GenerateError(t, "It should throw an error")
	} else if err.Error() != "An error occurred, Impossible to sign the JWT" {
		errors.GenerateError(t, fmt.Sprintf("Error isn't equals to expected, %s given", err.Error()))
	}

	// Valid token
	w = httptest.NewRecorder()
	http.SetCookie(w, tests.GetValidToken())
	r = &http.Request{
		Header: http.Header{
			"Cookie": w.HeaderMap["Set-Cookie"],
		},
	}
	jwt, err = CheckToken(security, w, r)
	if w.Result().StatusCode != http.StatusOK {
		errors.GenerateError(t, "Status code should be 200")
	}
	if jwt == nil {
		errors.GenerateError(t, "Token shouldn't be nil")
	}
	if jwt.Username != "user1" {
		errors.GenerateError(t, "Token username should exist")
	}
	if jwt.ExpiresAt == 0 {
		errors.GenerateError(t, "Token expiration should be set and different than 0")
	}
	if err != nil {
		errors.GenerateError(t, "It shouldn't throw an error")
	}
}
