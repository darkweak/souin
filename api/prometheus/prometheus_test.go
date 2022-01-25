package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func Test_Run(t *testing.T) {
	if len(registered) != 0 {
		t.Error("The registered additional metrics array must be empty.")
	}

	run()
	if len(registered) != 4 {
		t.Error("The registered additional metrics array must have 4 items.")
	}

	i, ok := registered[requestCounter]
	if !ok {
		t.Error("The registered array must have the request_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The request_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[noCachedResponseCounter]
	if !ok {
		t.Error("The registered array must have the no_cached_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The no_cached_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[cachedResponseCounter]
	if !ok {
		t.Error("The registered array must have the cached_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The cached_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[avgResponseTime]
	if !ok {
		t.Error("The registered array must have the avg_response_time key")
	}
	_, ok = i.(prometheus.Histogram)
	if !ok {
		t.Errorf("The avg_response_time element must be a prometheus.Histogram object, %T given.", i)
	}
}

func getMetricValue(col prometheus.Collector, t string) float64 {
	c := make(chan prometheus.Metric, 1)
	col.Collect(c)
	m := dto.Metric{}
	_ = (<-c).Write(&m)
	switch t {
	case average:
		return *m.Histogram.SampleSum
	default:
		return *m.Counter.Value
	}
}

func Test_Add(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	run()
	if getMetricValue(registered[requestCounter].(prometheus.Counter), counter) != 0 {
		t.Errorf("The request_counter value must be equal to 0 since it's not updated, %f given.", getMetricValue(registered[requestCounter].(prometheus.Counter), counter))
	}
	Add(requestCounter, 12.34)
	if getMetricValue(registered[requestCounter].(prometheus.Counter), counter) != 12.34 {
		t.Errorf("The request_counter value must be equal to 12.34 when it's updated with it, %f given.", getMetricValue(registered[requestCounter].(prometheus.Counter), counter))
	}
	Add(requestCounter, 1.2)
	if getMetricValue(registered[requestCounter].(prometheus.Counter), counter) != 13.54 {
		t.Errorf("The request_counter value must be equal to 13.54 when it's updated with +1.2, %f given.", getMetricValue(registered[requestCounter].(prometheus.Counter), counter))
	}
	if getMetricValue(registered[avgResponseTime].(prometheus.Histogram), average) != 0 {
		t.Errorf("The avg_response_time value must be equal to 0 since it's not updated, %f given.", getMetricValue(registered[avgResponseTime].(prometheus.Histogram), average))
	}
	Add(avgResponseTime, 12.34)
	if getMetricValue(registered[avgResponseTime].(prometheus.Histogram), average) != 12.34 {
		t.Errorf("The avg_response_time value must be equal to 12.34 when it's updated with it, %f given.", getMetricValue(registered[avgResponseTime].(prometheus.Histogram), average))
	}
	Add(avgResponseTime, 1.2)
	if getMetricValue(registered[avgResponseTime].(prometheus.Histogram), average) != 13.54 {
		t.Errorf("The avg_response_time value must be equal to 13.54 when it's updated with +1.2, %f given.", getMetricValue(registered[avgResponseTime].(prometheus.Histogram), average))
	}
}

func Test_Inc(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	run()
	if getMetricValue(registered[requestCounter].(prometheus.Counter), counter) != 0 {
		t.Errorf("The request_counter value must be equal to 0 since it's not incremented yet, %f given.", getMetricValue(registered[requestCounter].(prometheus.Counter), counter))
	}
	Increment(requestCounter)
	if getMetricValue(registered[requestCounter].(prometheus.Counter), counter) != 1.00 {
		t.Errorf("The request_counter value must be equal to 1.00 when it's incremented once, %f given.", getMetricValue(registered[requestCounter].(prometheus.Counter), counter))
	}
	Increment(requestCounter)
	if getMetricValue(registered[requestCounter].(prometheus.Counter), counter) != 2.00 {
		t.Errorf("The request_counter value must be equal to 2.00 when it's incremented twice, %f given.", getMetricValue(registered[requestCounter].(prometheus.Counter), counter))
	}
}

func Test_push(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	registered = make(map[string]interface{})
	push("UNKNOWN_TYPE", "dummy", "")
	if i := registered["dummy"]; i != nil {
		t.Error("The push method must not create a prometheus object while given an unknown type.")
	}

	push(average, "dummy_average", "")
	i := registered["dummy_average"]
	if i == nil {
		t.Error("The push method must create a prometheus object while given an average type.")
	}
	if _, ok := i.(prometheus.Histogram); !ok {
		t.Errorf("The dummy entry must be a type of prometheus.Histogram when the average type is set, %T given.", i)
	}

	push(counter, "dummy_counter", "")
	i = registered["dummy_counter"]
	if i == nil {
		t.Error("The push method must create a prometheus object while given a counter type.")
	}
	if _, ok := i.(prometheus.Counter); !ok {
		t.Errorf("The dummy_counter entry must be a type of prometheus.Counter when the average type is set, %T given.", i)
	}
}
