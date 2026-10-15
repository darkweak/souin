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

type ErrorType string

// IDNA errors
const (
	DomainToASCII   ErrorType = "Unicode ToASCII records an error or returns the empty string"
	DomainToUnicode ErrorType = "Unicode ToUnicode records an error"
)

// Host parsing errors
const (
	DomainInvalidCodePoint     ErrorType = "The host contains a forbidden domain code point"
	HostInvalidCodePoint       ErrorType = "An opaque host (in a URL that is not special) contains a forbidden host code point"
	IPv4EmptyPart              ErrorType = "An IPv4 address ends with a U+002E (.)"
	IPv4TooManyParts           ErrorType = "An IPv4 address has more than four parts"
	IPv4NonNumericPart         ErrorType = "An IPv4 address contains a non-numeric part"
	IPv4NonDecimalPart         ErrorType = "The IPv4 address contains numbers expressed using hexadecimal or octal digits"
	IPv4OutOfRangePart         ErrorType = "An IPv4 address contains a part that is greater than 255"
	IPv6Unclosed               ErrorType = "An IPv6 address is missing the closing U+005D (])"
	IPv6InvalidCompression     ErrorType = "An IPv6 address begins with improper compression"
	IPv6TooManyPieces          ErrorType = "An IPv6 address has more than eight pieces"
	IPv6MultipleCompression    ErrorType = "An IPv6 address contains multiple instances of '::'"
	IPv6InvalidCodePoint       ErrorType = "An IPv6 address contains a code point that is neither an ASCII hex digit nor a U+003A (:). Or it unexpectedly ends"
	IPv6TooFewPieces           ErrorType = "An uncompressed IPv6 address contains fewer than 8 pieces"
	IPv4InIPv6TooManyPieces    ErrorType = "An IPv4 address is found in an IPv6 address, but the IPv6 address has more than 6 pieces"
	IPv4InIPv6InvalidCodePoint ErrorType = "An IPv4 address is found in an IPv6 address and one of the following is true: 1. An IPv4 part is empty or contains a non-ASCII digit. 2. An IPv4 part contains a leading 0. 3. There are too many IPv4 parts"
	IPv4InIPv6OutOfRangePart   ErrorType = "An IPv4 address is found in an IPv6 address and one of the IPv4 parts is greater than 255"
	IPv4InIPv6TooFewParts      ErrorType = "An IPv4 address is found in an IPv6 address and there are too few IPv4 parts"
)

// URL parsing errors
const (
	InvalidURLUnit                       ErrorType = "A code point is found that is not a URL unit"
	SpecialSchemeMissingFollowingSolidus ErrorType = "The input’s scheme is not followed by '//'"
	MissingSchemeNonRelativeURL          ErrorType = "The input is missing a scheme, because it does not begin with an ASCII alpha, and either no base URL was provided or the base URL cannot be used as a base URL because it has an opaque path"
	InvalidReverseSolidus                ErrorType = "The URL has a special scheme and it uses U+005C (\\) instead of U+002F (/)"
	InvalidCredentials                   ErrorType = "The input includes credentials"
	HostMissing                          ErrorType = "The input has a special scheme, but does not contain a host"
	PortMissing                          ErrorType = "The input has a hostname followed by a ':' but is missing the port number"
	PortOutOfRange                       ErrorType = "The input's port is outside the range [0-65535]"
	PortInvalid                          ErrorType = "The input's port is not a number"
	FileInvalidWindowsDriveLetter        ErrorType = "The input is a relative-URL string that starts with a Windows drive letter and the base URL’s scheme is 'file'"
	FileInvalidWindowsDriveLetterHost    ErrorType = "A file: URL’s host is a Windows drive letter"
)
