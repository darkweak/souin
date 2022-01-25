package prometheus

import (
	"net/http"

	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/configurationtypes"
)

const (
	counter = "counter"
	average = "average"

	requestCounter          = "request_counter"
	noCachedResponseCounter = "no_cached_response_counter"
	cachedResponseCounter   = "cached_response_counter"
	avgResponseTime         = "avg_response_time"
)

// PrometheusAPI object contains informations related to the endpoints
type PrometheusAPI struct {
	basePath string
	enabled  bool
	security *auth.SecurityAPI
}

// InitializePrometheus initialize the prometheus endpoints
func InitializePrometheus(configuration configurationtypes.AbstractConfigurationInterface, api *auth.SecurityAPI) *PrometheusAPI {
	basePath := configuration.GetAPI().Prometheus.BasePath
	enabled := configuration.GetAPI().Prometheus.Enable
	var security *auth.SecurityAPI
	if configuration.GetAPI().Souin.Security {
		security = api
	}
	if basePath == "" {
		basePath = "/metrics"
	}
	return &PrometheusAPI{
		basePath,
		enabled,
		security,
	}
}

// GetBasePath will return the basepath for this resource
func (p *PrometheusAPI) GetBasePath() string {
	return p.basePath
}

// IsEnabled will return enabled status
func (p *PrometheusAPI) IsEnabled() bool {
	return p.enabled
}

// HandleRequest will handle the request
func (p *PrometheusAPI) HandleRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Not enabled for Træfik due to unsafe usage inside the prometheus/client_golang dependency. They don't want to support it inside plugins."))
}

// Increment will increment the counter.
func Increment(name string) {}

// Increment will add the referred value the counter.
func Add(name string, value float64) {}

// Run populate and prepare the map with the default values.
func run() {}
