package api

import (
	"encoding/json"
	"fmt"
	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"regexp"
)

// SouinAPI object contains informations related to the endpoints
type SouinAPI struct {
	basePath string
	enabled  bool
	providers map[string]types.AbstractProviderInterface
	security *auth.SecurityAPI
}

func initializeSouin(providers map[string]types.AbstractProviderInterface, configuration configurationtypes.AbstractConfigurationInterface, api *auth.SecurityAPI) *SouinAPI {
	basePath := configuration.GetAPI().Souin.BasePath
	enabled := configuration.GetAPI().Souin.Enable
	var security *auth.SecurityAPI
	if configuration.GetAPI().Souin.Security {
		security = api
	}
	if "" == basePath {
		basePath = "/souin"
	}
	return &SouinAPI{
		basePath,
		enabled,
		providers,
		security,
	}
}

// BulkDelete allow user to delete multiple items with regexp
func (s *SouinAPI) BulkDelete(rg *regexp.Regexp) {
	for _, v := range s.GetAll() {
		for _, key := range v {
			if rg.Match([]byte(key)) {
				s.Delete(key)
			}
		}
	}
}

// Delete will delete a record into the provider cache system and will update the Souin API if enabled
func (s *SouinAPI) Delete(key string) {
	for _, p := range s.providers {
		p.Delete(key)
	}
}

// GetAll will retrieve all stored keys in the provider
func (s *SouinAPI) GetAll() map[string][]string {
	list := map[string][]string{}
	for pName, p := range s.providers {
		list[pName] = p.ListKeys()
	}

	return list
}

// GetBasePath will return the basepath for this resource
func (s *SouinAPI) GetBasePath() string {
	return s.basePath
}

// IsEnabled will return enabled status
func (s *SouinAPI) IsEnabled() bool {
	return s.enabled
}

// HandleRequest will handle the request
func (s *SouinAPI) HandleRequest(w http.ResponseWriter, r *http.Request) {
	res := []byte{}
	if s.security != nil {
		if _, err := auth.CheckToken(s.security, w, r); err != nil {
			w.Write(res)
			return
		}
	}
	compile := regexp.MustCompile(fmt.Sprintf("%s/.+", s.GetBasePath())).FindString(r.RequestURI) != ""
	switch r.Method {
	case http.MethodGet:
		if compile {
			w.WriteHeader(http.StatusNotFound)
		} else {
			res, _ = json.Marshal(s.GetAll())
		}
		w.Header().Set("Content-Type", "application/json")
	case "PURGE":
		if compile {
			submatch := regexp.MustCompile(fmt.Sprintf("%s/(.+)", s.GetBasePath())).FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			s.BulkDelete(regexp.MustCompile(submatch))
		}
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	w.Write(res)
}
