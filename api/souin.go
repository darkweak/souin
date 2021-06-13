package api

import (
	"encoding/json"
	"fmt"
	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"regexp"
)

// SouinAPI object contains informations related to the endpoints
type SouinAPI struct {
	basePath string
	enabled  bool
	provider types.AbstractProviderInterface
	security *auth.SecurityAPI
	ykeyStorage *ykeys.YKeyStorage
}

func initializeSouin(provider types.AbstractProviderInterface, configuration configurationtypes.AbstractConfigurationInterface, api *auth.SecurityAPI, ykeyStorage *ykeys.YKeyStorage) *SouinAPI {
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
		provider,
		security,
		ykeyStorage,
	}
}

// BulkDelete allow user to delete multiple items with regexp
func (s *SouinAPI) BulkDelete(key string) {
	s.provider.DeleteMany(key)
}

func (s *SouinAPI) invalidateFromYKey(key string) {
	urls := s.ykeyStorage.InvalidateTags([]string{key})
	for _, u := range urls {
		s.provider.Delete(u)
	}
}

// Delete will delete a record into the provider cache system and will update the Souin API if enabled
func (s *SouinAPI) Delete(key string) {
	s.provider.Delete(key)
}

// GetAll will retrieve all stored keys in the provider
func (s *SouinAPI) GetAll() []string {
	return s.provider.ListKeys()
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
		query := r.URL.Query().Get("ykey")
		if query != "" {
			s.invalidateFromYKey(query)
		} else if compile {
			submatch := regexp.MustCompile(fmt.Sprintf("%s/(.+)", s.GetBasePath())).FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			s.BulkDelete(submatch)
		}
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	w.Write(res)
}
