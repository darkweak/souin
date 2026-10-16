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

package url

import (
	"github.com/nlnwa/whatwg-url/errors"
)

// handleError handles an error according to the options set for the parser
func (p *parser) handleError(u *Url, errorType errors.ErrorType, failure bool) error {
	e := errors.Error(errorType, u.inputUrl, failure)
	if p.opts.reportValidationErrors {
		u.validationErrors = append(u.validationErrors, e)
	}
	if failure || p.opts.failOnValidationError {
		return e
	}
	return nil
}

// handleErrorWithDescription handles an error according to the options set for the parser
func (p *parser) handleErrorWithDescription(u *Url, errorType errors.ErrorType, failure bool, descr string) error {
	e := errors.ErrorWithDescr(errorType, descr, u.inputUrl, failure)
	if p.opts.reportValidationErrors {
		u.validationErrors = append(u.validationErrors, e)
	}
	if failure || p.opts.failOnValidationError {
		return e
	}
	return nil
}

// handleWrappedError handles an error according to the options set for the parser
func (p *parser) handleWrappedError(u *Url, errorType errors.ErrorType, failure bool, cause error) error {
	e := errors.Wrap(cause, errorType, u.inputUrl, failure)
	if p.opts.reportValidationErrors {
		u.validationErrors = append(u.validationErrors, e)
	}
	if failure || p.opts.failOnValidationError {
		return e
	}
	return nil
}
