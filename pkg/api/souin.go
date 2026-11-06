package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/darkweak/storages/core"
	"github.com/pquerna/cachecontrol/cacheobject"
	"google.golang.org/protobuf/proto"
)

// SouinAPI object contains informations related to the endpoints
type SouinAPI struct {
	basePath         string
	enabled          bool
	storers          []types.Storer
	surrogateStorage providers.SurrogateInterface
	allowedMethods   []string
	stale            time.Duration
	softByDefault    bool
	compiledBP       *regexp.Regexp
	extractArgsBP    *regexp.Regexp
}

type invalidationType string

const (
	uriInvalidationType       invalidationType = "uri"
	uriPrefixInvalidationType invalidationType = "uri-prefix"
	originInvalidationType    invalidationType = "origin"
	groupInvalidationType     invalidationType = "group"

	// SoftPurgeModeHeader switches a PURGE request from removing the matching
	// responses to making them stale.
	SoftPurgeModeHeader = "Souin-Purge-Mode"
	softPurgeModeValue  = "soft"
	hardPurgeModeValue  = "hard"
)

// IsSoftPurgeRequest tells whether a PURGE request asks for a soft purge.
func IsSoftPurgeRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get(SoftPurgeModeHeader), softPurgeModeValue)
}

// softPurgesSurrogateKeys tells whether a purge by surrogate key is a soft one:
// what the request says wins, and without it `cdn.strategy: soft` decides.
// That strategy is about purging by tag, so it does not reach the regexp purge
// nor the flush, which stay hard unless the request asks otherwise.
func (s *SouinAPI) softPurgesSurrogateKeys(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get(SoftPurgeModeHeader), hardPurgeModeValue) {
		return false
	}

	return IsSoftPurgeRequest(r) || s.softByDefault
}

type invalidation struct {
	Type      invalidationType `json:"type"`
	Selectors []string         `json:"selectors"`
	Groups    []string         `json:"groups"`
	Purge     bool             `json:"purge"`
}

func initializeSouin(
	configuration configurationtypes.AbstractConfigurationInterface,
	storers []types.Storer,
	surrogateStorage providers.SurrogateInterface,
) *SouinAPI {
	basePath := configuration.GetAPI().Souin.BasePath
	if basePath == "" {
		basePath = "/souin"
	}

	allowedMethods := configuration.GetDefaultCache().GetAllowedHTTPVerbs()
	if len(allowedMethods) == 0 {
		allowedMethods = []string{http.MethodGet, http.MethodHead}
	}

	return &SouinAPI{
		basePath,
		configuration.GetAPI().Souin.Enable,
		storers,
		surrogateStorage,
		allowedMethods,
		configuration.GetDefaultCache().GetStale(),
		// Only an explicit `soft`: the strategy is documented to default to it,
		// and nobody who did not ask for it gets their purges turned soft.
		configuration.GetDefaultCache().GetCDN().Strategy == "soft",
		regexp.MustCompile(basePath + "/.+"),
		regexp.MustCompile(basePath + "/(.+)"),
	}
}

// BulkDelete deletes the given key, its mapping and the keys referenced by that mapping.
// Every deletion is exact: the key is never interpreted as a regexp.
// Without purge the responses are kept and made stale instead, see SoftPurge.
func (s *SouinAPI) BulkDelete(key string, purge bool) {
	if !purge {
		s.SoftPurge(key)

		return
	}

	key, _ = strings.CutPrefix(key, core.MappingKeyPrefix)
	for _, current := range s.storers {
		if b := current.Get(core.MappingKeyPrefix + key); len(b) > 0 {
			var mapping core.StorageMapper

			if e := proto.Unmarshal(b, &mapping); e == nil {
				for k := range mapping.GetMapping() {
					current.Delete(k)
				}
			}
		}

		current.Delete(core.MappingKeyPrefix + key)
		current.Delete(key)
	}
}

// SoftPurge makes every response stored for the given key stale, without
// removing it. The key is exact and so is every response it designates.
//
// The stored expiry of each response moves to now, which is all the
// middleware needs: the next request goes down the stale path, is answered
// with the stored response and refreshes it in the background. Nothing else
// is touched, neither the mapping nor the surrogate index, since the
// responses they point to are still stored.
func (s *SouinAPI) SoftPurge(key string) {
	key, _ = strings.CutPrefix(key, core.MappingKeyPrefix)
	for _, current := range s.storers {
		b := storedMapping(current, key)
		if len(b) == 0 {
			continue
		}

		var mapping core.StorageMapper
		if e := proto.Unmarshal(b, &mapping); e != nil {
			continue
		}

		for k := range mapping.GetMapping() {
			s.softPurgeStored(current, k)
		}
	}
}

// storedMapping reads the mapping of a key. Some storers hand mappings back
// through MapKeys only, since SetMultiLevel keeps them apart from the values
// Get returns.
func storedMapping(current types.Storer, key string) []byte {
	if b := current.Get(core.MappingKeyPrefix + key); len(b) > 0 {
		return b
	}

	// The prefix scan returns every key sharing the prefix, the entry named
	// exactly like the key being the one with nothing left after the prefix.
	return []byte(current.MapKeys(core.MappingKeyPrefix + key)[""])
}

// SoftPurgeMatching makes stale every stored response whose key matches the
// regexp, the soft counterpart of the regexp purge.
func (s *SouinAPI) SoftPurgeMatching(pattern string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return
	}

	for _, current := range s.storers {
		for _, k := range current.ListKeys() {
			if re.MatchString(k) {
				s.softPurgeStored(current, k)
			}
		}
	}
}

func (s *SouinAPI) softPurgeStored(current types.Storer, storedKey string) {
	stored := current.Get(storedKey)
	if len(stored) == 0 {
		return
	}

	rewritten, ttl, err := rfc.SoftPurgeStored(stored, time.Now(), s.stale, s.retention)
	if err != nil {
		// Nothing left to refresh, or not a response to begin with. Only the
		// first one is a purge that has to take effect.
		if errors.Is(err, rfc.ErrSoftPurgeExpired) {
			current.Delete(storedKey)
		}

		return
	}

	_ = current.Set(storedKey, rewritten, ttl)
}

// retention is the extra time a response asks to stay available once stale.
func (s *SouinAPI) retention(h http.Header) time.Duration {
	if s.surrogateStorage == nil {
		return 0
	}

	name, _ := s.surrogateStorage.GetSurrogateControl(h)
	cc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(h, name))

	return rfc.RetentionWindow(cc)
}

// Delete will delete a record into the provider cache system and will update the Souin API if enabled
// The key can be a regexp to delete multiple items
func (s *SouinAPI) Delete(key string) {
	_, err := regexp.Compile(key)
	for _, current := range s.storers {
		if err != nil {
			current.Delete(key)
		} else {
			current.DeleteMany(key)
		}
	}
}

// GetAll will retrieve all stored keys in the provider
func (s *SouinAPI) GetAll() []string {
	keys := []string{}
	for _, current := range s.storers {
		keys = append(keys, current.ListKeys()...)
	}

	return keys
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

var storageToInfiniteTTLMap = map[string]time.Duration{
	"BADGER":                 types.OneYearDuration,
	"ETCD":                   types.OneYearDuration,
	"GO-REDIS":               0,
	"NUTS":                   0,
	"OLRIC":                  types.OneYearDuration,
	"OTTER":                  types.OneYearDuration,
	"REDIS":                  -1,
	"SIMPLEFS":               0,
	types.DefaultStorageName: types.OneYearDuration,
}

// maxMappingValueSize bounds the mapping values the eviction is willing to
// decode. Anything larger is dropped undecoded: unmarshalling a pathological
// value would materialize it in memory, and its response keys expire through
// their own TTLs anyway.
const maxMappingValueSize = 1 << 20

func EvictMapping(current types.Storer) {
	now := time.Now()
	infiniteStoreDuration := storageToInfiniteTTLMap[current.Name()]

	processMapping := func(k string, v []byte) bool {
		if len(v) > maxMappingValueSize {
			fmt.Println("Deleting the oversized mapping", core.MappingKeyPrefix+k)
			current.Delete(core.MappingKeyPrefix + k)

			return true
		}

		mapping := &core.StorageMapper{}

		e := proto.Unmarshal(v, mapping)
		if e != nil {
			current.Delete(core.MappingKeyPrefix + k)
			return true
		}

		updated := false
		for key, val := range mapping.GetMapping() {
			if now.Sub(val.FreshTime.AsTime()) > 0 && now.Sub(val.StaleTime.AsTime()) > 0 {
				current.Delete(key)
				current.Delete(mapping.GetMapping()[key].RealKey)

				delete(mapping.GetMapping(), key)
				updated = true
			}
		}

		if updated {
			v, e := proto.Marshal(mapping)
			if e != nil {
				fmt.Println("Impossible to re-encode the mapping", core.MappingKeyPrefix+k)
				current.Delete(core.MappingKeyPrefix + k)
			}
			_ = current.Set(core.MappingKeyPrefix+k, v, infiniteStoreDuration)
		}

		if len(mapping.GetMapping()) == 0 {
			current.Delete(core.MappingKeyPrefix + k)
		}

		return true
	}

	// Stream the mapping index in bounded batches when the storer supports
	// it, so the whole index is never materialized in memory at once.
	if walker, ok := current.(core.MappingWalker); ok {
		_ = walker.WalkMappings(core.MappingKeyPrefix, processMapping)

		return
	}

	for k, v := range current.MapKeys(core.MappingKeyPrefix) {
		processMapping(k, []byte(v))
	}
}

func (s *SouinAPI) purgeMapping() {
	for _, current := range s.storers {
		EvictMapping(current)
	}

	fmt.Println("Successfully clear the mappings.")
}

// HandleRequest will handle the request
func (s *SouinAPI) HandleRequest(w http.ResponseWriter, r *http.Request) {
	res := []byte{}
	compile := s.compiledBP.FindString(r.RequestURI) != ""
	switch r.Method {
	case http.MethodGet:
		if strings.Contains(r.RequestURI, s.GetBasePath()+"/surrogate_keys") {
			res, _ = json.Marshal(s.surrogateStorage.List())
		} else if compile {
			search := s.extractArgsBP.FindAllStringSubmatch(r.RequestURI, -1)[0][1]
			res, _ = json.Marshal(s.listKeys(search))
			if len(res) == 2 {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			res, _ = json.Marshal(s.GetAll())
		}
		w.Header().Set("Content-Type", "application/json")
	case http.MethodPost:
		var invalidator invalidation
		defer func() {
			_ = r.Body.Close()
		}()
		err := json.NewDecoder(r.Body).Decode(&invalidator)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		keysToInvalidate := []string{}
		switch invalidator.Type {
		case groupInvalidationType:
			var surrogateKeys []string
			keysToInvalidate, surrogateKeys = s.surrogateStorage.Purge(http.Header{"Surrogate-Key": invalidator.Groups})
			for _, k := range surrogateKeys {
				keysToInvalidate = append(keysToInvalidate, "SURROGATE_"+k)
			}
		case uriPrefixInvalidationType, uriInvalidationType:
			bodyKeys := []string{}
			listedKeys := s.GetAll()
			for _, k := range invalidator.Selectors {
				if !strings.Contains(k, "//") {
					rq, err := http.NewRequest(http.MethodGet, "//"+k, nil)
					if err != nil {
						continue
					}

					bodyKeys = append(bodyKeys, rq.Host+"-"+rq.URL.Path)
				}
			}

			for _, allKey := range listedKeys {
				for _, bk := range bodyKeys {
					if invalidator.Type == uriInvalidationType {
						if strings.Contains(allKey, bk) && strings.Contains(allKey, bk+"-") && strings.HasSuffix(allKey, bk) {
							keysToInvalidate = append(keysToInvalidate, allKey)
							break
						}
					} else {
						if strings.Contains(allKey, bk) &&
							(strings.Contains(allKey, bk+"-") || strings.Contains(allKey, bk+"?") || strings.Contains(allKey, bk+"/") || strings.HasSuffix(allKey, bk)) {
							keysToInvalidate = append(keysToInvalidate, allKey)
							break
						}
					}
				}
			}
		case originInvalidationType:
			bodyKeys := []string{}
			listedKeys := s.GetAll()
			for _, k := range invalidator.Selectors {
				if !strings.Contains(k, "//") {
					rq, err := http.NewRequest(http.MethodGet, "//"+k, nil)
					if err != nil {
						continue
					}

					bodyKeys = append(bodyKeys, rq.Host)
				}
			}

			for _, allKey := range listedKeys {
				for _, bk := range bodyKeys {
					if strings.Contains(allKey, bk) {
						keysToInvalidate = append(keysToInvalidate, allKey)
						break
					}
				}
			}
		}

		for _, k := range keysToInvalidate {
			s.BulkDelete(k, invalidator.Purge)
		}
		w.WriteHeader(http.StatusOK)
	case "PURGE":
		softPurge := IsSoftPurgeRequest(r)
		if compile {
			if strings.HasSuffix(r.RequestURI, s.GetBasePath()+"/flush") {
				if softPurge {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte("soft purge is not supported for flush"))

					return
				}
				for _, current := range s.storers {
					current.DeleteMany(".+")
				}
				e := s.surrogateStorage.Destruct()
				if e != nil {
					fmt.Printf("Error while purging the surrogate keys: %+v.", e)
				}
				fmt.Println("Successfully clear the cache and the surrogate keys storage.")
			} else if strings.HasSuffix(r.RequestURI, s.GetBasePath()+"/mapping") {
				s.purgeMapping()
			} else {
				submatch := s.extractArgsBP.FindAllStringSubmatch(r.RequestURI, -1)[0][1]
				if softPurge {
					s.SoftPurgeMatching(submatch)
				} else {
					for _, current := range s.storers {
						current.DeleteMany(submatch)
					}
				}
			}
		} else {
			if s.softPurgesSurrogateKeys(r) {
				// The tags keep pointing at the responses: they are still stored.
				ck, _ := s.surrogateStorage.Lookup(r.Header)
				for _, k := range ck {
					s.SoftPurge(k)
				}
				w.WriteHeader(http.StatusNoContent)

				return
			}

			ck, surrogateKeys := s.surrogateStorage.Purge(r.Header)
			for _, k := range ck {
				s.BulkDelete(k, true)
			}
			for _, k := range surrogateKeys {
				for _, current := range s.storers {
					current.Delete("SURROGATE_" + k)
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	_, _ = w.Write(res)
}
