// Package urlpattern implements the URLPattern web API.
//
// The specification is available at https://urlpattern.spec.whatwg.org/.
package urlpattern

import (
	"errors"
	"regexp"
	"strings"

	"github.com/nlnwa/whatwg-url/url"
)

var (
	ErrNoBaseURL             = errors.New("relative URL and no baseURL provided")
	ErrUnexpectedEmptyString = errors.New("unexpected empty string")
)

// Init-processing mode per https://urlpattern.spec.whatwg.org/#process-a-urlpatterninit.
const (
	initTypePattern = "pattern"
	initTypeURL     = "url"
)

// https://url.spec.whatwg.org/#special-scheme
var specialSchemeSet = map[string]struct{}{
	"ftp":   {},
	"http":  {},
	"https": {},
	"ws":    {},
	"wss":   {},
}

type URLPatternResult struct {
	Inputs     []string
	InitInputs []*URLPatternInit

	Protocol URLPatternComponentResult
	Username URLPatternComponentResult
	Password URLPatternComponentResult
	Hostname URLPatternComponentResult
	Port     URLPatternComponentResult
	Pathname URLPatternComponentResult
	Search   URLPatternComponentResult
	Hash     URLPatternComponentResult
}

type URLPatternComponentResult struct {
	Input  string
	Groups map[string]string
}

// https://urlpattern.spec.whatwg.org/#url-pattern-struct
type URLPattern struct {
	protocol *component
	username *component
	password *component
	hostname *component
	port     *component
	pathname *component
	search   *component
	hash     *component
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-protocol
func (u *URLPattern) Protocol() string {
	return u.protocol.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-username
func (u *URLPattern) Username() string {
	return u.username.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-password
func (u *URLPattern) Password() string {
	return u.password.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-hostname
func (u *URLPattern) Hostname() string {
	return u.hostname.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-port
func (u *URLPattern) Port() string {
	return u.port.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-pathname
func (u *URLPattern) Pathname() string {
	return u.pathname.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-search
func (u *URLPattern) Search() string {
	return u.search.patternString
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-hash
func (u *URLPattern) Hash() string {
	return u.hash.patternString
}

// https://urlpattern.spec.whatwg.org/#component
type component struct {
	patternString     string
	regularExpression *regexp.Regexp
	groupNameList     []string
	hasRegexpGroups   bool
}

// https://urlpattern.spec.whatwg.org/#protocol-component-matches-a-special-scheme
func (c *component) protocolComponentMatchesSpecialScheme() bool {
	for scheme := range specialSchemeSet {
		if c.regularExpression.MatchString(scheme) {
			return true
		}
	}

	return false
}

// https://urlpattern.spec.whatwg.org/#url-pattern-create
func New(input string, baseURL string, options *Options) (*URLPattern, error) {
	init, err := parseConstructorString(input)
	if err != nil {
		return nil, err
	}

	if baseURL == "" && init.Protocol == nil {
		return nil, ErrNoBaseURL
	}

	if baseURL != "" {
		init.BaseURL = &baseURL
	}

	return init.New(options)
}

// https://urlpattern.spec.whatwg.org/#url-pattern-create
func (init *URLPatternInit) New(opt *Options) (*URLPattern, error) {
	if opt == nil {
		opt = &Options{}
	}

	processedInit, err := init.process(initTypePattern, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	star := "*"
	if processedInit.Protocol == nil {
		processedInit.Protocol = &star
	}
	if processedInit.Username == nil {
		processedInit.Username = &star
	}
	if processedInit.Password == nil {
		processedInit.Password = &star
	}
	if processedInit.Hostname == nil {
		processedInit.Hostname = &star
	}
	if processedInit.Port == nil {
		processedInit.Port = &star
	}
	if processedInit.Pathname == nil {
		processedInit.Pathname = &star
	}
	if processedInit.Search == nil {
		processedInit.Search = &star
	}
	if processedInit.Hash == nil {
		processedInit.Hash = &star
	}

	var emptyString string
	// Only clear the port when the protocol is a WHATWG special scheme; the
	// exported DefaultPorts map is user-extendable, so keying off it alone
	// would quietly apply the behaviour to arbitrary user-added protocols.
	//
	// In "pattern" mode processedInit.Protocol is not canonicalized, so
	// lowercase it for the comparison: the protocol component is compiled
	// with canonicalizeProtocol (which lowercases), so the effective pattern
	// is the lowercase form.
	canonicalProtocol := strings.ToLower(*processedInit.Protocol)
	if _, isSpecial := specialSchemeSet[canonicalProtocol]; isSpecial {
		if dp, ok := DefaultPorts[canonicalProtocol]; ok && *processedInit.Port == dp {
			processedInit.Port = &emptyString
		}
	}

	defaultOptions := options{}

	urlPattern := &URLPattern{}
	urlPattern.protocol, err = compileComponent(*processedInit.Protocol, canonicalizeProtocol, defaultOptions)
	if err != nil {
		return nil, err
	}
	urlPattern.username, err = compileComponent(*processedInit.Username, canonicalizeUsername, defaultOptions)
	if err != nil {
		return nil, err
	}

	urlPattern.password, err = compileComponent(*processedInit.Password, canonicalizePassword, defaultOptions)
	if err != nil {
		return nil, err
	}

	// If the result running hostname pattern is an IPv6 address given processedInit["hostname"] is true, then set urlPattern’s hostname component to the result of compiling a component given processedInit["hostname"], canonicalize an IPv6 hostname, and hostname options.

	protocolMatchesSpecialScheme := urlPattern.protocol.protocolComponentMatchesSpecialScheme()

	hostnameOptions := options{delimiterCodePoint: '.'}
	switch {
	case hostnamePatternIsIPv6Address(*processedInit.Hostname):
		urlPattern.hostname, err = compileComponent(*processedInit.Hostname, canonicalizeIPv6Hostname, hostnameOptions)
	case protocolMatchesSpecialScheme || *processedInit.Protocol == "*":
		urlPattern.hostname, err = compileComponent(*processedInit.Hostname, canonicalizeDomainName, hostnameOptions)
	default:
		urlPattern.hostname, err = compileComponent(*processedInit.Hostname, func(s string) (string, error) { return canonicalizeHostname(s, "") }, hostnameOptions)
	}
	if err != nil {
		return nil, err
	}

	urlPattern.port, err = compileComponent(*processedInit.Port, func(s string) (string, error) { return canonicalizePort(s, "") }, defaultOptions)
	if err != nil {
		return nil, err
	}

	compileOptions := defaultOptions
	compileOptions.ignoreCase = opt.IgnoreCase

	pathnameOptions := options{'/', '/', false}

	if protocolMatchesSpecialScheme {
		pathCompileOptions := pathnameOptions
		pathCompileOptions.ignoreCase = opt.IgnoreCase

		urlPattern.pathname, err = compileComponent(*processedInit.Pathname, canonicalizePathname, pathCompileOptions)
		if err != nil {
			return nil, err
		}
	} else {
		urlPattern.pathname, err = compileComponent(*processedInit.Pathname, canonicalizeOpaquePathname, compileOptions)
		if err != nil {
			return nil, err
		}
	}

	urlPattern.search, err = compileComponent(*processedInit.Search, canonicalizeSearch, compileOptions)
	if err != nil {
		return nil, err
	}

	urlPattern.hash, err = compileComponent(*processedInit.Hash, canonicalizeHash, compileOptions)
	if err != nil {
		return nil, err
	}

	return urlPattern, nil
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-exec
func (u *URLPattern) ExecInit(input *URLPatternInit) *URLPatternResult {
	protocol := ""
	username := ""
	password := ""
	hostname := ""
	port := ""
	pathname := ""
	search := ""
	hash := ""

	inputs := []*URLPatternInit{input}

	applyResult, err := input.process(initTypeURL, &protocol, &username, &password, &hostname, &port, &pathname, &search, &hash)
	if err != nil {
		return nil
	}

	protocol = *applyResult.Protocol
	username = *applyResult.Username
	password = *applyResult.Password
	hostname = *applyResult.Hostname
	port = *applyResult.Port
	pathname = *applyResult.Pathname
	search = *applyResult.Search
	hash = *applyResult.Hash

	r := u.match(protocol, username, password, hostname, port, pathname, search, hash)
	if r != nil {
		r.InitInputs = inputs
	}

	return r
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-exec
func (u *URLPattern) Exec(input, baseURLString string) *URLPatternResult {
	inputs := []string{input}

	var baseURL *url.Url
	var err error

	if baseURLString != "" {
		baseURL, err = url.Parse(baseURLString)
		if err != nil {
			return nil
		}

		inputs = append(inputs, baseURLString)
	}

	ur, err := urlParser.BasicParser(input, baseURL, nil, url.NoState)
	if err != nil {
		return nil
	}

	r := u.match(
		ur.Scheme(), ur.Username(), ur.Password(), ur.Hostname(),
		ur.Port(), ur.Pathname(), ur.Query(), ur.Fragment(),
	)
	if r != nil {
		r.Inputs = inputs
	}

	return r
}

// https://urlpattern.spec.whatwg.org/#url-pattern-match
func (u *URLPattern) match(protocol, username, password, hostname, port, pathname, search, hash string) *URLPatternResult {
	protocolExecResult := u.protocol.regularExpression.FindStringSubmatch(protocol)
	usernameExecResult := u.username.regularExpression.FindStringSubmatch(username)
	passwordExecResult := u.password.regularExpression.FindStringSubmatch(password)
	hostnameExecResult := u.hostname.regularExpression.FindStringSubmatch(hostname)
	portExecResult := u.port.regularExpression.FindStringSubmatch(port)
	pathnameExecResult := u.pathname.regularExpression.FindStringSubmatch(pathname)
	searchExecResult := u.search.regularExpression.FindStringSubmatch(search)
	hashExecResult := u.hash.regularExpression.FindStringSubmatch(hash)

	if protocolExecResult == nil ||
		usernameExecResult == nil ||
		passwordExecResult == nil ||
		hostnameExecResult == nil ||
		portExecResult == nil ||
		pathnameExecResult == nil ||
		searchExecResult == nil ||
		hashExecResult == nil {
		return nil
	}

	result := &URLPatternResult{}
	result.Protocol = createComponentMatchResult(*u.protocol, protocol, protocolExecResult)
	result.Username = createComponentMatchResult(*u.username, username, usernameExecResult)
	result.Password = createComponentMatchResult(*u.password, password, passwordExecResult)
	result.Hostname = createComponentMatchResult(*u.hostname, hostname, hostnameExecResult)
	result.Port = createComponentMatchResult(*u.port, port, portExecResult)
	result.Pathname = createComponentMatchResult(*u.pathname, pathname, pathnameExecResult)
	result.Search = createComponentMatchResult(*u.search, search, searchExecResult)
	result.Hash = createComponentMatchResult(*u.hash, hash, hashExecResult)

	return result
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-test
func (u *URLPattern) Test(input, baseURL string) bool {
	return u.Exec(input, baseURL) != nil
}

// https://urlpattern.spec.whatwg.org/#dom-urlpattern-test
func (u *URLPattern) TestInit(input *URLPatternInit) bool {
	return u.ExecInit(input) != nil
}

// https://urlpattern.spec.whatwg.org/#url-pattern-has-regexp-groups
func (u *URLPattern) HasRegexpGroups() bool {
	return u.protocol.hasRegexpGroups ||
		u.username.hasRegexpGroups ||
		u.password.hasRegexpGroups ||
		u.hostname.hasRegexpGroups ||
		u.port.hasRegexpGroups ||
		u.pathname.hasRegexpGroups ||
		u.search.hasRegexpGroups ||
		u.hash.hasRegexpGroups
}

// https://urlpattern.spec.whatwg.org/#create-a-component-match-result
func createComponentMatchResult(component component, input string, execResult []string) URLPatternComponentResult {
	result := URLPatternComponentResult{Input: input}

	if len(component.groupNameList) == 0 || (len(execResult) == 2 && execResult[0] == "" && execResult[1] == "") {
		return result
	}

	result.Groups = make(map[string]string, len(component.groupNameList))
	limit := len(execResult)
	if namedLimit := len(component.groupNameList) + 1; namedLimit < limit {
		limit = namedLimit
	}
	for index := 1; index < limit; index++ {
		name := component.groupNameList[index-1]
		value := execResult[index]

		result.Groups[name] = value
	}

	return result
}

type Options struct {
	IgnoreCase bool
}

// https://urlpattern.spec.whatwg.org/#dictdef-urlpatterninit
type URLPatternInit struct {
	Protocol *string
	Username *string
	Password *string
	Hostname *string
	Port     *string
	Pathname *string
	Search   *string
	Hash     *string

	BaseURL *string
}

// https://urlpattern.spec.whatwg.org/#process-a-urlpatterninit
func (init *URLPatternInit) process(iType string, protocol, username, password, hostname, port, pathname, search, hash *string) (*URLPatternInit, error) {
	result := &URLPatternInit{protocol, username, password, hostname, port, pathname, search, hash, nil}

	var (
		baseURL *url.Url
		err     error
	)
	if init.BaseURL != nil {
		baseURL, err = url.Parse(*init.BaseURL)
		if err != nil {
			return nil, err
		}

		if init.Protocol == nil {
			p := processBaseURLString(baseURL.Scheme(), iType)
			result.Protocol = &p
		}

		// TODO: the end of this block can be simplified, but let's be as close as possible from the standard algorithm for now

		if iType != initTypePattern && init.Protocol == nil && init.Hostname == nil && init.Port == nil && init.Username == nil {
			u := processBaseURLString(baseURL.Username(), iType)
			result.Username = &u
		}

		if iType != initTypePattern && init.Protocol == nil && init.Hostname == nil && init.Port == nil && init.Username == nil && init.Password == nil {
			password := baseURL.Password()
			p := processBaseURLString(password, iType)
			result.Password = &p
		}

		if init.Protocol == nil && init.Hostname == nil {
			baseHost := baseURL.Hostname()
			h := processBaseURLString(baseHost, iType)
			result.Hostname = &h
		}

		if init.Protocol == nil && init.Hostname == nil && init.Port == nil {
			p := baseURL.Port()
			result.Port = &p
		}

		if init.Protocol == nil && init.Hostname == nil && init.Port == nil && init.Pathname == nil {
			p := processBaseURLString(baseURL.Pathname(), iType)
			result.Pathname = &p
		}

		if init.Protocol == nil && init.Hostname == nil && init.Port == nil && init.Pathname == nil && init.Search == nil {
			s := processBaseURLString(baseURL.Query(), iType)
			result.Search = &s
		}

		if init.Protocol == nil && init.Hostname == nil && init.Port == nil && init.Pathname == nil && init.Search == nil && init.Hash == nil {
			h := processBaseURLString(baseURL.Fragment(), iType)
			result.Hash = &h
		}
	}

	if init.Protocol != nil {
		p, err := processProtocolForInit(*init.Protocol, iType)
		if err != nil {
			return nil, err
		}

		result.Protocol = &p
	}

	if init.Username != nil {
		u, err := processUsernameForInit(*init.Username, iType)
		if err != nil {
			return nil, err
		}

		result.Username = &u
	}

	if init.Password != nil {
		p, err := processPasswordForInit(*init.Password, iType)
		if err != nil {
			return nil, err
		}

		result.Password = &p
	}

	var proto string
	if result.Protocol == nil {
		proto = ""
	} else {
		proto = *result.Protocol
	}

	if init.Hostname != nil {
		h, err := processHostnameForInit(*init.Hostname, proto, iType)
		if err != nil {
			return nil, err
		}

		result.Hostname = &h
	}

	if init.Port != nil {
		p, err := processPortForInit(*init.Port, proto, iType)
		if err != nil {
			return nil, err
		}

		result.Port = &p
	}

	if init.Pathname != nil {
		result.Pathname = init.Pathname

		// TODO: according to the spec, we should check that he path is opaque, but it's illogical and breaks the tests
		if baseURL != nil && !baseURL.OpaquePath() && !isAbsolutePathname(*result.Pathname, iType) {
			baseURLPath := processBaseURLString(baseURL.Pathname(), iType)

			slashIndex := strings.LastIndex(baseURLPath, "/")
			if slashIndex != -1 {
				newPathname := baseURLPath[0:slashIndex+1] + *result.Pathname
				result.Pathname = &newPathname
			}
		}

		p, err := processPathnameForInit(*result.Pathname, proto, iType)
		if err != nil {
			return nil, err
		}

		result.Pathname = &p
	}

	if init.Search != nil {
		s, err := processSearchForInit(*init.Search, iType)
		if err != nil {
			return nil, err
		}

		result.Search = &s
	}

	if init.Hash != nil {
		h, err := processHashForInit(*init.Hash, iType)
		if err != nil {
			return nil, err
		}

		result.Hash = &h
	}

	return result, nil
}

// https://urlpattern.spec.whatwg.org/#process-a-base-url-string
func processBaseURLString(input, uType string) string {
	if uType != initTypePattern {
		return input
	}

	return escapePatternString(input)
}

// https://urlpattern.spec.whatwg.org/#process-protocol-for-init
func processProtocolForInit(value, pType string) (string, error) {
	strippedValue := strings.TrimSuffix(value, ":")

	if pType == initTypePattern {
		return strippedValue, nil
	}

	return canonicalizeProtocol(strippedValue)
}

// https://urlpattern.spec.whatwg.org/#process-username-for-init
func processUsernameForInit(value, uType string) (string, error) {
	if uType == initTypePattern {
		return value, nil
	}

	return canonicalizeUsername(value)
}

// https://urlpattern.spec.whatwg.org/#process-password-for-init
func processPasswordForInit(value, uType string) (string, error) {
	if uType == initTypePattern {
		return value, nil
	}

	return canonicalizePassword(value)
}

// https://urlpattern.spec.whatwg.org/#process-hostname-for-init
func processHostnameForInit(value, protocolValue, uType string) (string, error) {
	if uType == initTypePattern {
		return value, nil
	}

	if protocolValue == "" {
		return canonicalizeDomainName(value)
	}

	if _, ok := specialSchemeSet[protocolValue]; ok {
		return canonicalizeDomainName(value)
	}

	return canonicalizeHostname(value, protocolValue)
}

// https://urlpattern.spec.whatwg.org/#process-port-for-init
func processPortForInit(portValue, protocolValue, pType string) (string, error) {
	if pType == initTypePattern {
		return portValue, nil
	}

	return canonicalizePort(portValue, protocolValue)
}

// https://urlpattern.spec.whatwg.org/#process-pathname-for-init
func processPathnameForInit(pathnameValue, protocolValue, ptype string) (string, error) {
	if ptype == initTypePattern {
		return pathnameValue, nil
	}

	if protocolValue == "" {
		return canonicalizePathname(pathnameValue)
	}

	if _, ok := specialSchemeSet[protocolValue]; ok {
		return canonicalizePathname(pathnameValue)
	}

	return canonicalizeOpaquePathname(pathnameValue)
}

// https://urlpattern.spec.whatwg.org/#process-search-for-init
func processSearchForInit(value, sType string) (string, error) {
	strippedValue := strings.TrimPrefix(value, "?")

	if sType == initTypePattern {
		return strippedValue, nil
	}

	return canonicalizeSearch(strippedValue)
}

// https://urlpattern.spec.whatwg.org/#process-hash-for-init
func processHashForInit(value, hType string) (string, error) {
	strippedValue := strings.TrimPrefix(value, "#")

	if hType == initTypePattern {
		return strippedValue, nil
	}

	return canonicalizeHash(strippedValue)
}

// https://urlpattern.spec.whatwg.org/#is-an-absolute-pathname
func isAbsolutePathname(input, pType string) bool {
	if input == "" {
		return false
	}

	if input[0] == '/' {
		return true
	}

	if pType == initTypeURL {
		return false
	}

	return strings.HasPrefix(input, "\\/") || strings.HasPrefix(input, "{/")
}

// https://urlpattern.spec.whatwg.org/#hostname-pattern-is-an-ipv6-address
func hostnamePatternIsIPv6Address(input string) bool {
	if len(input) < 2 {
		return false
	}

	i := []rune(input)

	if i[0] == '[' {
		return true
	}
	if i[0] == '{' && i[1] == '[' {
		return true
	}
	if i[0] == '\\' && i[1] == '[' {
		return true
	}

	return false
}
