package main

import (
	"github.com/function61/gokit/stopper"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type metrics struct {
	requestCount       prometheus.Counter
	requestAccepted    prometheus.Counter
	requestBlacklisted prometheus.Counter
	requestDuration    prometheus.Histogram
	blocklistItems     prometheus.Gauge
}

func makeMetrics() *metrics {
	// from 0.25ms to 8 seconds
	timeBuckets := prometheus.ExponentialBuckets(0.00025, 2, 16)

	metrics := &metrics{
		requestCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_count_total",
			Help: "Total requests",
		}),
		requestAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_count_accepted",
			Help: "Accepted (non-blacklisted) requests",
		}),
		requestBlacklisted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_count_blacklisted",
			Help: "Blacklisted requests",
		}),
		requestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Buckets: timeBuckets,
			Help:    "Histogram of the time (in seconds) each request took.",
		}),
		blocklistItems: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "blocklist_items",
			Help: "Number of items in the blocklist",
		}),
	}

	// TODO: write test to guarantee that this is in sync with the struct?
	allCollectors := []prometheus.Collector{
		metrics.requestCount,
		metrics.requestAccepted,
		metrics.requestBlacklisted,
		metrics.requestDuration,
		metrics.blocklistItems,
	}

	for _, collector := range allCollectors {
		prometheus.MustRegister(collector)
	}

	return metrics
}

func metricsServer(stop *stopper.Stopper) error {
	http.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Addr: ":80",
	}

	go func() {
		defer stop.Done()

		<-stop.Signal

		srv.Shutdown(nil)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
