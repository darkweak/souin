package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/surrogate/providers"
)

// SouinAPI object contains informations related to the endpoints
type SouinAPI struct {
	basePath         string
	enabled          bool
	storer           storage.Storer
	surrogateStorage providers.SurrogateInterface
}

func initializeSouin(
	configuration configurationtypes.AbstractConfigurationInterface,
	storer storage.Storer,
	surrogateStorage providers.SurrogateInterface,
) *SouinAPI {
	basePath := configuration.GetAPI().Souin.BasePath
	if basePath == "" {
		basePath = "/souin"
	}
	return &SouinAPI{
		basePath,
		configuration.GetAPI().Souin.Enable,
		storer,
		surrogateStorage,
	}
}

// BulkDelete allow user to delete multiple items with regexp
func (s *SouinAPI) BulkDelete(key string) {
	s.storer.DeleteMany(key)
}

// Delete will delete a record into the provider cache system and will update the Souin API if enabled
func (s *SouinAPI) Delete(key string) {
	s.storer.Delete(key)
}

// GetAll will retrieve all stored keys in the provider
func (s *SouinAPI) GetAll() []string {
	return s.storer.ListKeys()
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
	compile := regexp.MustCompile(s.GetBasePath()+"/.+").FindString(r.RequestURI) != ""
	switch r.Method {
	case http.MethodGet:
		if regexp.MustCompile(s.GetBasePath()+"/surrogate_keys").FindString(r.RequestURI) != "" {
			res, _ = json.Marshal(s.surrogateStorage.List())
		} else if compile {
			search := regexp.MustCompile(s.GetBasePath()+"/(.+)").FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			res, _ = json.Marshal(s.listKeys(search))
			if len(res) == 2 {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			res, _ = json.Marshal(s.GetAll())
		}
		w.Header().Set("Content-Type", "application/json")
	case "PURGE":
		if compile {
			keysRg := regexp.MustCompile(s.GetBasePath() + "/(.+)")
			flushRg := regexp.MustCompile(s.GetBasePath() + "/flush$")

			if flushRg.FindString(r.RequestURI) != "" {
				s.storer.DeleteMany(".+")
				e := s.surrogateStorage.Destruct()
				if e != nil {
					fmt.Printf("Error while purging the surrogate keys: %+v.", e)
				}
				fmt.Println("Successfully clear the cache and the surrogate keys storage.")
			} else {
				submatch := keysRg.FindAllStringSubmatch(r.RequestURI, -1)[0][1]
				s.BulkDelete(submatch)
			}
		} else {
			ck, _ := s.surrogateStorage.Purge(r.Header)
			for _, k := range ck {
				s.storer.Delete(k)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	_, _ = w.Write(res)
}
