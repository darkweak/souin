//go:build wasi || wasm

package prometheus

import (
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	counter = "counter"
	average = "average"

	RequestCounter             = "souin_request_upstream_counter"
	RequestRevalidationCounter = "souin_request_revalidation_counter"
	NoCachedResponseCounter    = "souin_no_cached_response_counter"
	CachedResponseCounter      = "souin_cached_response_counter"
	AvgResponseTime            = "souin_avg_response_time"
)

// PrometheusAPI object contains informations related to the endpoints
type PrometheusAPI struct {
	basePath string
	enabled  bool
}

// InitializePrometheus initialize the prometheus endpoints
func InitializePrometheus(configuration configurationtypes.AbstractConfigurationInterface) *PrometheusAPI {
	basePath := configuration.GetAPI().Prometheus.BasePath
	enabled := configuration.GetAPI().Prometheus.Enable
	if basePath == "" {
		basePath = "/metrics"
	}

	if registered == nil {
		run()
	}
	return &PrometheusAPI{
		basePath,
		enabled,
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
func (p *PrometheusAPI) HandleRequest(_ http.ResponseWriter, _ *http.Request) {}

var registered map[string]interface{}

// Increment will increment the counter.
func Increment(_ string) {}

// Increment will add the referred value the counter.
func Add(_ string, _ float64) {}

func push(_, _, _ string) {}

// Run populate and prepare the map with the default values.
func run() {
	registered = make(map[string]interface{})
}
