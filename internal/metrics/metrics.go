package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	GRPCRequests     *prometheus.CounterVec
	GRPCDuration     *prometheus.HistogramVec
	ExchangeRequests *prometheus.CounterVec
	ExchangeDuration prometheus.Histogram
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		GRPCRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests by method and status code.",
		}, []string{"method", "code"}),
		GRPCDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		ExchangeRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "exchange_requests_total",
			Help: "Total number of order book requests to the exchange by result.",
		}, []string{"result"}),
		ExchangeDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "exchange_request_duration_seconds",
			Help:    "Duration of order book requests to the exchange in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	reg.MustRegister(m.GRPCRequests, m.GRPCDuration, m.ExchangeRequests, m.ExchangeDuration)
	return m
}
