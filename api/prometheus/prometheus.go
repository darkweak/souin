package prometheus

import (
	"net/http"

	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	promhttp.Handler().ServeHTTP(w, r)
}

var registered map[string]interface{}

// Increment will increment the counter.
func Increment(name string) {
	registered[name].(prometheus.Counter).Inc()
}

// Increment will add the referred value the counter.
func Add(name string, value float64) {
	if c, ok := registered[name].(prometheus.Counter); ok {
		c.Add(value)
	}
	if g, ok := registered[name].(prometheus.Histogram); ok {
		g.Observe(value)
	}
}

func push(promType, name, help string) {
	switch promType {
	case counter:
		registered[name] = promauto.NewCounter(prometheus.CounterOpts{
			Name: name,
			Help: help,
		})

		return
	case average:
		avg := prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: name,
			Help: help,
		})
		prometheus.MustRegister(avg)
		registered[name] = avg
	}
}

// Run populate and prepare the map with the default values.
func Run() {
	registered = make(map[string]interface{})
	push(counter, requestCounter, "Total request counter")
	push(counter, noCachedResponseCounter, "No cached response counter")
	push(counter, cachedResponseCounter, "Cached response counter")
	push(average, avgResponseTime, "Average Bidswitch response time")
}
