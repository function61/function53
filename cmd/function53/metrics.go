package main

import (
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
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
			Name: "request_count_total",
			Help: "Total requests (accepted + rejected by client + blocklisted)",
		}),
		requestAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_count_accepted",
			Help: "Accepted requests",
		}),
		requestRejectedByClient: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_rejected_by_client",
			Help: "Requests made by clients that are not allowed to do DNS queries",
		}),
		requestBlocklisted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_count_blocklisted",
			Help: "Blocklisted requests (ads/malware etc.)",
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

func metricsServer(conf Config, logger *log.Logger, stop *stopper.Stopper) error {
	http.Handle("/metrics", promhttp.Handler())

	logl := logex.Levels(logger)

	addr := fmt.Sprintf(":%d", conf.MetricsPort)

	srv := http.Server{
		Addr: addr,
	}

	logl.Info.Printf("starting to listen at %s", addr)

	go func() {
		defer stop.Done()
		defer logl.Info.Println("stopped")

		<-stop.Signal

		srv.Shutdown(nil)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
