/*
 * Copyright 2020 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package errors

import (
	"fmt"
)

// ValidationError indicates that the url is not valid
type ValidationError struct {
	errorType ErrorType
	cause     error  // the root cause for this error
	descr     string // description of the error
	failure   bool   // true if the error is a failure, false if it is a warning
	url       string
}

func (e *ValidationError) Error() string {
	errMsg := fmt.Sprintf("Error: %s", e.errorType)
	if e.descr != "" {
		errMsg += fmt.Sprintf(": '%s'", e.descr)
	}
	if e.url != "" {
		errMsg += fmt.Sprintf(". Url: '%s'", e.url)
	}
	if nil == e.cause {
		return errMsg
	}

	return errMsg + ", Cause: " + e.cause.Error()
}

// Unwrap returns the root cause for this error
func (e *ValidationError) Unwrap() error {
	return e.cause
}

// Type returns the error type
func (e *ValidationError) Type() ErrorType {
	return e.errorType
}

// Url returns the url causing the error
func (e *ValidationError) Url() string {
	return e.url
}

// Failure returns true if the error is a failure, false if it is a warning
func (e *ValidationError) Failure() bool {
	return e.failure
}

// Description returns the error description
func (e *ValidationError) Description() string {
	return e.descr
}

// Type returns the error type
func Type(err error) ErrorType {
	type typer interface {
		Type() ErrorType
	}

	cd, ok := err.(typer)
	if !ok {
		return ""
	}
	return cd.Type()
}

// Description returns the error description
func Description(err error) string {
	type descr interface {
		Description() string
	}

	m, ok := err.(descr)
	if !ok {
		return ""
	}
	return m.Description()
}

// Url returns the url causing the error
func Url(err error) string {
	type url interface {
		Url() string
	}

	m, ok := err.(url)
	if !ok {
		return ""
	}
	return m.Url()
}

// Failure returns true if the error is a failure, false if it is a warning.
// If the error does not implement the Failure() method, true is returned
func Failure(err error) bool {
	type failure interface {
		Failure() bool
	}

	m, ok := err.(failure)
	if !ok {
		return true
	}
	return m.Failure()
}

// Error constructs a new error
func Error(errorType ErrorType, url string, failure bool) error {
	return &ValidationError{
		errorType: errorType,
		url:       url,
		failure:   failure,
	}
}

// ErrorWithDescr constructs a new error
func ErrorWithDescr(errorType ErrorType, descr string, url string, failure bool) error {
	return &ValidationError{
		errorType: errorType,
		descr:     descr,
		url:       url,
		failure:   failure,
	}
}

// Wrap wraps an error with an error code and url
func Wrap(err error, errorType ErrorType, url string, failure bool) error {
	return &ValidationError{
		errorType: errorType,
		url:       url,
		cause:     err,
		failure:   failure,
	}
}

// WrapWithDescr wraps an error with an error code, url and a description
func WrapWithDescr(err error, errorType ErrorType, descr string, url string, failure bool) error {
	return &ValidationError{
		errorType: errorType,
		descr:     descr,
		url:       url,
		cause:     err,
		failure:   failure,
	}
}
