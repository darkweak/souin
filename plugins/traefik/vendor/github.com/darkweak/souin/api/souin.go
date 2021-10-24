package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
)

// SouinAPI object contains informations related to the endpoints
type SouinAPI struct {
	basePath         string
	enabled          bool
	provider         types.AbstractProviderInterface
	security         *auth.SecurityAPI
	ykeyStorage      *ykeys.YKeyStorage
	surrogateStorage providers.SurrogateInterface
}

func initializeSouin(
	configuration configurationtypes.AbstractConfigurationInterface,
	api *auth.SecurityAPI,
	transport types.TransportInterface,
) *SouinAPI {
	basePath := configuration.GetAPI().Souin.BasePath
	var security *auth.SecurityAPI
	if configuration.GetAPI().Souin.Security {
		security = api
	}
	if "" == basePath {
		basePath = "/souin"
	}
	return &SouinAPI{
		basePath,
		configuration.GetAPI().Souin.Enable,
		transport.GetProvider(),
		security,
		transport.GetYkeyStorage(),
		transport.GetSurrogateKeys(),
	}
}

// BulkDelete allow user to delete multiple items with regexp
func (s *SouinAPI) BulkDelete(key string) {
	s.provider.DeleteMany(key)
}

func (s *SouinAPI) invalidateFromYKey(keys []string) {
	if s.ykeyStorage == nil {
		return
	}
	urls := s.ykeyStorage.InvalidateTags(keys)
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
			_, _ = w.Write(res)
			return
		}
	}
	compile := regexp.MustCompile(s.GetBasePath()+"/.+").FindString(r.RequestURI) != ""
	switch r.Method {
	case http.MethodGet:
		if regexp.MustCompile(s.GetBasePath()+"/surrogate_keys").FindString(r.RequestURI) != "" {
			res, _ = json.Marshal(s.surrogateStorage.List())
		} else if compile {
			w.WriteHeader(http.StatusNotFound)
		} else {
			res, _ = json.Marshal(s.GetAll())
		}
		w.Header().Set("Content-Type", "application/json")
	case "PURGE":
		// surr := providers.ParseHeaders(r.Header.Get(providers.SurrogateKey))
		// fmt.Printf("%+v \n%+v \n", surr[0], surr)
		query := r.URL.Query()["ykey"]
		if len(query) > 0 {
			s.invalidateFromYKey(query)
		} else if compile {
			submatch := regexp.MustCompile(s.GetBasePath()+"/(.+)").FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			s.BulkDelete(submatch)
		}
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	_, _ = w.Write(res)
}
