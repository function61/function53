package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/function61/gokit/log/logex"
	"github.com/function61/gokit/net/http/httputils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	requestCount            prometheus.Counter
	requestAccepted         prometheus.Counter
	requestBlocklisted      prometheus.Counter
	requestRejectedByClient prometheus.Counter
	requestDuration         prometheus.Histogram
	blocklistItems          prometheus.Gauge
}

func makeMetrics() *metrics {
	// from 0.25ms to 8 seconds
	timeBuckets := prometheus.ExponentialBuckets(0.00025, 2, 16)

	metrics := &metrics{
		requestCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "fn53_reqs_total",
			Help: "Total requests (accepted + rejected by client + blocklisted)",
		}),
		requestAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "fn53_reqs_accepted",
			Help: "Accepted requests",
		}),
		requestRejectedByClient: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "fn53_reqs_rejected_by_client",
			Help: "Requests made by clients that are not allowed to do DNS queries",
		}),
		requestBlocklisted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "fn53_reqs_blocklisted",
			Help: "Blocklisted requests (ads/malware etc.)",
		}),
		requestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "fn53_req_duration",
			Buckets: timeBuckets,
			Help:    "Histogram of the time (in seconds) each request took.",
		}),
		blocklistItems: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "fn53_blocklist_items",
			Help: "Number of items in the blocklist",
		}),
	}

	// TODO: write test to guarantee that this is in sync with the struct?
	allCollectors := []prometheus.Collector{
		metrics.requestCount,
		metrics.requestAccepted,
		metrics.requestRejectedByClient,
		metrics.requestBlocklisted,
		metrics.requestDuration,
		metrics.blocklistItems,
	}

	for _, collector := range allCollectors {
		prometheus.MustRegister(collector)
	}

	return metrics
}

func metricsServer(ctx context.Context, conf Config, logger *log.Logger) error {
	http.Handle("/metrics", promhttp.Handler())

	logl := logex.Levels(logger)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", conf.MetricsPort),
	}

	logl.Info.Printf("starting to listen at %s", srv.Addr)

	return httputils.CancelableServer(ctx, srv, func() error { return srv.ListenAndServe() })
}
