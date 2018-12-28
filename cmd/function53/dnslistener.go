package main

import (
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/miekg/dns"
	"log"
	"sync"
	"time"
)

type DnsQueryHandler struct {
	clientPool  *ClientConnectionPool
	queryLogger QueryLogger
	blocklist   Blocklist
	metrics     *metrics
	logl        *logex.Leveled
}

func NewDnsQueryHandler(clientPool *ClientConnectionPool, blocklist Blocklist, logger *log.Logger, queryLogger QueryLogger) *DnsQueryHandler {
	return &DnsQueryHandler{
		clientPool:  clientPool,
		blocklist:   blocklist,
		queryLogger: queryLogger,
		metrics:     makeMetrics(),
		logl:        logex.Levels(logger),
	}
}

// we don't have to support len(req.Question) > 1:
// https://serverfault.com/questions/742785/multi-query-multiple-dns-record-types-at-once
func (h *DnsQueryHandler) Handle(rw dns.ResponseWriter, req *dns.Msg) {
	started := time.Now()

	h.queryLogger.LogQuery(req.Question[0].Name, rw.RemoteAddr().String())

	h.metrics.requestCount.Inc()

	if h.blocklist.Has(req.Question[0].Name) {
		h.metrics.requestBlacklisted.Inc()
		h.handleRejection(rw, req)
	} else {
		h.metrics.requestAccepted.Inc()
		job := NewJob(req)
		h.clientPool.Jobs <- job
		resp := <-job.Response

		if err := rw.WriteMsg(resp); err != nil {
			h.logl.Error.Printf("request dropped, error writing to client: %v", err)
		}
	}

	h.metrics.requestDuration.Observe(time.Since(started).Seconds())
}

func (h *DnsQueryHandler) handleRejection(rw dns.ResponseWriter, req *dns.Msg) {
	msg := &dns.Msg{}
	msg.SetReply(req)
	msg.SetRcode(req, dns.RcodeNameError)
	msg.Authoritative = true
	msg.RecursionAvailable = false

	// Add a useful TXT record
	header := dns.RR_Header{
		Name:   req.Question[0].Name,
		Class:  dns.ClassINET,
		Rrtype: dns.TypeTXT,
	}
	msg.Ns = []dns.RR{&dns.TXT{
		Hdr: header,
		Txt: []string{"Rejected query based on matched filters"},
	}}

	if err := rw.WriteMsg(msg); err != nil {
		// debugMsg("Error writing message: ", err)
	}
}

func runServer(handler *DnsQueryHandler, stop *stopper.Stopper) error {
	defer stop.Done()

	udpHandler := dns.NewServeMux()
	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", handler.Handle)
	udpHandler.HandleFunc(".", handler.Handle)

	tcpServer := &dns.Server{
		Addr:    "0.0.0.0:53",
		Net:     "tcp",
		Handler: tcpHandler,
		// ReadTimeout:  2 * time.Second, // FIXME: incorrect?
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	// WriteTimeout: 2 * time.Second}

	udpServer := &dns.Server{
		Addr:    "0.0.0.0:53",
		Net:     "udp",
		Handler: udpHandler,
		UDPSize: 65535,
		// ReadTimeout:  2 * time.Second, // FIXME: incorrect?
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	// WriteTimeout: 2 * time.Second}

	serverErrored := make(chan error, 1)

	once := sync.Once{}
	serverErroredOnce := func(err error) {
		once.Do(func() { serverErrored <- err })
	}

	go listenAndServeDns(udpServer, serverErroredOnce)
	go listenAndServeDns(tcpServer, serverErroredOnce)

	select {
	case <-stop.Signal:
		udpServer.Shutdown()
		tcpServer.Shutdown()

		return nil
	case err := <-serverErrored:
		udpServer.Shutdown()
		tcpServer.Shutdown()

		return err
	}
}

func listenAndServeDns(ds *dns.Server, serverErrored func(error)) {
	if err := ds.ListenAndServe(); err != nil {
		serverErrored(fmt.Errorf("Start %s listener on %s failed: %v", ds.Net, ds.Addr, err))
	}
}
