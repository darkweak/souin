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

	i, ok := registered[RequestCounter]
	if !ok {
		t.Error("The registered array must have the souin_request_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_request_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[NoCachedResponseCounter]
	if !ok {
		t.Error("The registered array must have the souin_no_cached_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_no_cached_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[CachedResponseCounter]
	if !ok {
		t.Error("The registered array must have the souin_cached_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_cached_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[AvgResponseTime]
	if !ok {
		t.Error("The registered array must have the souin_avg_response_time key")
	}
	_, ok = i.(prometheus.Histogram)
	if !ok {
		t.Errorf("The souin_avg_response_time element must be a prometheus.Histogram object, %T given.", i)
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
	if getMetricValue(registered[RequestCounter].(prometheus.Counter), counter) != 0 {
		t.Errorf("The souin_request_counter value must be equal to 0 since it's not updated, %f given.", getMetricValue(registered[RequestCounter].(prometheus.Counter), counter))
	}
	Add(RequestCounter, 12.34)
	if getMetricValue(registered[RequestCounter].(prometheus.Counter), counter) != 12.34 {
		t.Errorf("The souin_request_counter value must be equal to 12.34 when it's updated with it, %f given.", getMetricValue(registered[RequestCounter].(prometheus.Counter), counter))
	}
	Add(RequestCounter, 1.2)
	if getMetricValue(registered[RequestCounter].(prometheus.Counter), counter) != 13.54 {
		t.Errorf("The souin_request_counter value must be equal to 13.54 when it's updated with +1.2, %f given.", getMetricValue(registered[RequestCounter].(prometheus.Counter), counter))
	}
	if getMetricValue(registered[AvgResponseTime].(prometheus.Histogram), average) != 0 {
		t.Errorf("The souin_avg_response_time value must be equal to 0 since it's not updated, %f given.", getMetricValue(registered[AvgResponseTime].(prometheus.Histogram), average))
	}
	Add(AvgResponseTime, 12.34)
	if getMetricValue(registered[AvgResponseTime].(prometheus.Histogram), average) != 12.34 {
		t.Errorf("The souin_avg_response_time value must be equal to 12.34 when it's updated with it, %f given.", getMetricValue(registered[AvgResponseTime].(prometheus.Histogram), average))
	}
	Add(AvgResponseTime, 1.2)
	if getMetricValue(registered[AvgResponseTime].(prometheus.Histogram), average) != 13.54 {
		t.Errorf("The souin_avg_response_time value must be equal to 13.54 when it's updated with +1.2, %f given.", getMetricValue(registered[AvgResponseTime].(prometheus.Histogram), average))
	}
}

func Test_Increment(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	run()
	if getMetricValue(registered[RequestCounter].(prometheus.Counter), counter) != 0 {
		t.Errorf("The souin_request_counter value must be equal to 0 since it's not incremented yet, %f given.", getMetricValue(registered[RequestCounter].(prometheus.Counter), counter))
	}
	Increment(RequestCounter)
	if getMetricValue(registered[RequestCounter].(prometheus.Counter), counter) != 1.00 {
		t.Errorf("The souin_request_counter value must be equal to 1.00 when it's incremented once, %f given.", getMetricValue(registered[RequestCounter].(prometheus.Counter), counter))
	}
	Increment(RequestCounter)
	if getMetricValue(registered[RequestCounter].(prometheus.Counter), counter) != 2.00 {
		t.Errorf("The souin_request_counter value must be equal to 2.00 when it's incremented twice, %f given.", getMetricValue(registered[RequestCounter].(prometheus.Counter), counter))
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
