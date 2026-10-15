/*
 * Copyright 2019 National Library of Norway.
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
	"strings"
)

// Url represents a URL.
type Url struct {
	inputUrl         string
	scheme           string
	username         string
	password         string
	host             *string
	port             *string
	decodedPort      int
	path             *path
	query            *string
	fragment         *string
	searchParams     *SearchParams
	validationErrors []error
	parser           *parser
	isIPv4           bool
	isIPv6           bool
}

// Href implements WHATWG url api (https://url.spec.whatwg.org/#api)
// If excludeFragment is true, the fragment component will be excluded from the output.
func (u *Url) Href(excludeFragment bool) string {
	output := u.scheme + ":"
	if u.host != nil {
		output += "//"
		if u.username != "" || u.password != "" {
			output += u.username
			if u.password != "" {
				output += ":" + u.password
			}
			output += "@"
		}
		output += *u.host
		if u.port != nil {
			output += ":" + *u.port
		}
	}
	if u.host == nil && !u.path.isOpaque() && len(u.path.p) > 1 && u.path.p[0] == "" {
		output += "/."
	}

	output += u.path.String()

	if u.query != nil {
		output += "?" + *u.query
	}

	if !excludeFragment && u.fragment != nil {
		output += "#" + *u.fragment
	}

	return output
}

// Protocol implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Protocol() string {
	return u.scheme + ":"
}

// SetProtocol implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetProtocol(scheme string) {
	if !strings.HasSuffix(scheme, ":") {
		scheme = scheme + ":"
	}
	_, _ = u.parser.BasicParser(scheme, nil, u, StateSchemeStart)
}

func (u *Url) Scheme() string {
	return u.scheme
}

// Username implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Username() string {
	return u.username
}

// SetUsername implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetUsername(username string) {
	if u.host == nil || *u.host == "" || u.scheme == "file" {
		return
	}
	u.username = u.parser.PercentEncodeString(username, UserInfoPercentEncodeSet)
}

// Password implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Password() string {
	return u.password
}

// SetPassword implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPassword(password string) {
	if u.host == nil || *u.host == "" || u.scheme == "file" {
		return
	}
	u.password = u.parser.PercentEncodeString(password, UserInfoPercentEncodeSet)
}

// Host implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Host() string {
	if u.host == nil {
		return ""
	}
	if u.port == nil {
		return *u.host
	}
	return *u.host + ":" + *u.port
}

// SetHost implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHost(host string) {
	if u.path.isOpaque() {
		return
	}
	_, _ = u.parser.BasicParser(host, nil, u, StateHost)
}

// Hostname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Hostname() string {
	if u.host == nil {
		return ""
	}
	return *u.host
}

// SetHostname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHostname(host string) {
	if u.path.isOpaque() {
		return
	}
	_, _ = u.parser.BasicParser(host, nil, u, StateHostname)
}

// Port implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Port() string {
	if u.port == nil {
		return ""
	}
	return *u.port
}

// SetPort implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPort(port string) {
	if u.host == nil || *u.host == "" || u.scheme == "file" {
		return
	}
	if port == "" {
		u.port = nil
		u.decodedPort = 0
	} else {
		_, _ = u.parser.BasicParser(port, nil, u, StatePort)
	}
}

func (u *Url) DecodedPort() int {
	if u.decodedPort == 0 {
		return u.getDefaultPort()
	} else {
		return u.decodedPort
	}
}

// Pathname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Pathname() string {
	return u.path.String()
}

// SetPathname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPathname(path string) {
	if u.path.isOpaque() {
		return
	}
	u.path.init()
	_, _ = u.parser.BasicParser(path, nil, u, StatePathStart)
}

// OpaquePath tells if the path is opaque (https://url.spec.whatwg.org/#url-opaque-path)
func (u *Url) OpaquePath() bool {
	return u.path.opaque
}

// Search implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Search() string {
	if u.query == nil || len(*u.query) == 0 {
		return ""
	}
	return "?" + *u.query
}

// SetSearch implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetSearch(query string) {
	if query == "" {
		u.query = nil
		if u.searchParams != nil {
			u.searchParams.params = u.searchParams.params[:0]
		}
		if u.fragment == nil && u.query == nil {
			u.path.stripTrailingSpacesIfOpaque()
		}
		return
	}
	query = strings.TrimPrefix(query, "?")
	if u.query == nil {
		u.query = new(string)
	}
	_, _ = u.parser.BasicParser(query, nil, u, StateQuery)
	if u.searchParams == nil {
		u.newUrlSearchParams()
	} else {
		u.searchParams.init(*u.query)
	}
}

// SearchParams implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SearchParams() *SearchParams {
	if u.searchParams == nil {
		u.newUrlSearchParams()
	}
	return u.searchParams
}

func (u *Url) SetSearchParams(searchParams *SearchParams) {
	u.searchParams = searchParams
	u.searchParams.update()
}

func (u *Url) Query() string {
	if u.query == nil || len(*u.query) == 0 {
		return ""
	}
	return *u.query
}

// Hash implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Hash() string {
	if u.fragment == nil || len(*u.fragment) == 0 {
		return ""
	}
	return "#" + *u.fragment
}

// SetHash implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHash(fragment string) {
	if fragment == "" {
		u.fragment = nil
		if u.fragment == nil && u.query == nil {
			u.path.stripTrailingSpacesIfOpaque()
		}
		return
	}
	fragment = strings.TrimPrefix(fragment, "#")
	u.fragment = new(string)
	_, _ = u.parser.BasicParser(fragment, nil, u, StateFragment)
}

func (u *Url) Fragment() string {
	if u.fragment == nil || len(*u.fragment) == 0 {
		return ""
	}
	return *u.fragment
}

func (u *Url) String() string {
	return u.Href(false)
}

func (u *Url) ValidationErrors() []error {
	return u.validationErrors
}

func (u *Url) newUrlSearchParams() {
	usp := &SearchParams{url: u}
	if u.query != nil {
		usp.init(*u.query)
	}
	u.searchParams = usp
}

func (u *Url) IsIPv4() bool {
	return u.isIPv4
}

func (u *Url) IsIPv6() bool {
	return u.isIPv6
}

// Clone returns a deep copy of the URL.
func (u *Url) Clone() *Url {
	return &Url{
		inputUrl:     u.inputUrl,
		scheme:       u.scheme,
		username:     u.username,
		password:     u.password,
		host:         cloneStringPointer(u.host),
		port:         cloneStringPointer(u.port),
		decodedPort:  u.decodedPort,
		path:         u.path.clone(),
		query:        cloneStringPointer(u.query),
		fragment:     cloneStringPointer(u.fragment),
		searchParams: u.SearchParams().Clone(),
		parser:       u.parser,
		isIPv4:       u.isIPv4,
		isIPv6:       u.isIPv6,
	}
}

func cloneStringPointer(s *string) *string {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}
