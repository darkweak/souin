package auth

import (
	"fmt"
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"regexp"
)

// SecurityAPI object contains informations related to the endpoints
type SecurityAPI struct {
	basePath string
	enabled   bool
	secret   []byte
	users    map[string]string
}

func InitializeSecurity(configuration configurationtypes.AbstractConfigurationInterface) *SecurityAPI {
	basePath := configuration.GetAPI().Security.BasePath
	enabled := configuration.GetAPI().Security.Enable
	secret := []byte(configuration.GetAPI().Security.Secret)
	users := make(map[string]string)
	for _, user := range configuration.GetAPI().Security.Users {
		users[user.Username] = user.Password
	}
	if "" == basePath {
		basePath = "/authentication"
	}
	return &SecurityAPI{
		basePath,
		enabled,
		secret,
		users,
	}
}

func (s *SecurityAPI) login(w http.ResponseWriter, r *http.Request) {
	signJWT(s, w, r)
}

func (s *SecurityAPI) refresh(w http.ResponseWriter, r *http.Request) {
	refresh(s, w, r)
}

// GetBasePath will return the basepath for this resource
func (s *SecurityAPI) GetBasePath() string {
	return s.basePath
}

// IsEnabled will return enabled status
func (s *SecurityAPI) IsEnabled() bool {
	return s.enabled
}

// HandleRequest will handle the request
func (s *SecurityAPI) HandleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if regexp.MustCompile(fmt.Sprintf("%s/login", s.GetBasePath())).FindString(r.RequestURI) != "" {
			s.login(w, r)
		} else if regexp.MustCompile(fmt.Sprintf("%s/refresh", s.GetBasePath())).FindString(r.RequestURI) != "" {
			s.refresh(w, r)
		} else {
			w.Write([]byte{})
		}
	default:
		w.Write([]byte{})
	}
}
