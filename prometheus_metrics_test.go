package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

type mockRegistry struct {
	mu         sync.Mutex
	collectors map[string]prometheus.Collector
	descs      []*prometheus.Desc
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		collectors: make(map[string]prometheus.Collector),
	}
}

func (m *mockRegistry) Register(c prometheus.Collector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		c.Describe(ch)
		close(ch)
	}()

	for desc := range ch {
		if desc == nil {
			continue
		}
		m.descs = append(m.descs, desc)
		m.collectors[desc.String()] = c
	}

	return nil
}

func (m *mockRegistry) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		if err := m.Register(c); err != nil {
			panic(err)
		}
	}
}

func (m *mockRegistry) Unregister(c prometheus.Collector) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		c.Describe(ch)
		close(ch)
	}()

	found := false
	for desc := range ch {
		if _, ok := m.collectors[desc.String()]; ok {
			delete(m.collectors, desc.String())
			found = true
		}
	}

	return found
}

func (m *mockRegistry) Gather() ([]*dto.MetricFamily, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	families := make([]*dto.MetricFamily, 0, len(m.collectors))
	for _, c := range m.collectors {
		ch := make(chan prometheus.Metric, 10)
		go func() {
			c.Collect(ch)
			close(ch)
		}()

		for metric := range ch {
			dtoMetric := &dto.Metric{}
			err := metric.Write(dtoMetric)
			if err != nil {
				return nil, err
			}

			desc := metric.Desc()
			name := desc.String() // This is not ideal, but we don't have access to the actual name
			family := &dto.MetricFamily{
				Name:   &name,
				Metric: []*dto.Metric{dtoMetric},
			}
			families = append(families, family)
		}
	}
	return families, nil
}

func Test_initPrometheusMetrics(t *testing.T) {
	mockReg := newMockRegistry()

	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = mockReg

	initPrometheusMetrics()

	prometheus.DefaultRegisterer = originalRegistry

	if len(mockReg.descs) != 2 {
		t.Fatalf("Expected 2 descriptors to be registered, got %d", len(mockReg.descs))
	}

	expectedMetrics := map[string]bool{
		"http_requests_total":           false,
		"http_request_duration_seconds": false,
	}

	for _, desc := range mockReg.descs {
		name := desc.String()
		if strings.Contains(name, "http_requests_total") {
			expectedMetrics["http_requests_total"] = true
		} else if strings.Contains(name, "http_request_duration_seconds") {
			expectedMetrics["http_request_duration_seconds"] = true
		}
	}

	for name, found := range expectedMetrics {
		if !found {
			t.Errorf("Expected metric %s not found", name)
		}
	}
}

func Test_prometheusMiddleware(t *testing.T) {
	mockReg := newMockRegistry()
	prometheus.DefaultRegisterer = mockReg
	initPrometheusMetrics()

	handler := prometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	families, _ := mockReg.Gather()
	for _, family := range families {
		if strings.Contains(*family.Name, "http_requests_total") {
			for _, metric := range family.Metric {
				if metric.GetCounter().GetValue() != 1 {
					t.Errorf("Expected counter to be incremented")
				}
			}
		}
	}
}

func Test_handlePrometheusMetrics(t *testing.T) {
	mockReg := newMockRegistry()
	prometheus.DefaultRegisterer = mockReg
	initPrometheusMetrics()

	handlePrometheusMetrics()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := promhttp.HandlerFor(mockReg, promhttp.HandlerOpts{})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	body := rr.Body.String()
	expectedMetrics := []string{"http_requests_total", "http_request_duration_seconds"}
	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Response does not contain expected metric: %s", metric)
		}
	}
}
