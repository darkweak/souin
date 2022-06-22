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
	if basePath == "" {
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

func (s *SouinAPI) listKeys(search string) []string {
	res := []string{}
	re, err := regexp.Compile(search)
	if err != nil {
		return res
	}
	for _, key := range s.GetAll() {
		if re.MatchString(key) {
			res = append(res, key)
		}
	}

	return res
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
			search := regexp.MustCompile(s.GetBasePath()+"/(.+)").FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			res, _ = json.Marshal(s.listKeys(search))
			if res == nil {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			res, _ = json.Marshal(s.GetAll())
		}
		w.Header().Set("Content-Type", "application/json")
	case "PURGE":
		if compile {
			submatch := regexp.MustCompile(s.GetBasePath()+"/(.+)").FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			s.BulkDelete(submatch)
		} else {
			ck, _ := s.surrogateStorage.Purge(r.Header)
			for _, k := range ck {
				s.provider.Delete(k)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	_, _ = w.Write(res)
}
