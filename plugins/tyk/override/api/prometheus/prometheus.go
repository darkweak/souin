package prometheus

import (
	"net/http"

	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/configurationtypes"
)

const (
	counter = "counter"
	average = "average"

	RequestCounter          = "souin_request_counter"
	NoCachedResponseCounter = "souin_no_cached_response_counter"
	CachedResponseCounter   = "souin_cached_response_counter"
	AvgResponseTime         = "souin_avg_response_time"
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

	if registered == nil {
		run()
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
	return false
}

// HandleRequest will handle the request
func (p *PrometheusAPI) HandleRequest(w http.ResponseWriter, r *http.Request) {}

var registered map[string]interface{}

// Increment will increment the counter.
func Increment(_ string) {}

// Increment will add the referred value the counter.
func Add(_ string, _ float64) {}

// Run populate and prepare the map with the default values.
func run() {
	registered = make(map[string]interface{})
}
