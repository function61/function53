package main

import (
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/miekg/dns"
	"log"
	"net"
	"sync"
	"time"
)

type DnsQueryHandler struct {
	ReloadBlocklist chan Blocklist
	conf            Config
	clientPool      *ClientConnectionPool
	queryLogger     QueryLogger
	blocklist       Blocklist
	blocklistMu     sync.Mutex
	metrics         *metrics
	logl            *logex.Leveled
}

func NewDnsQueryHandler(
	clientPool *ClientConnectionPool,
	conf Config,
	blocklist Blocklist,
	logger *log.Logger,
	queryLogger QueryLogger,
	stop *stopper.Stopper,
) *DnsQueryHandler {
	qh := &DnsQueryHandler{
		ReloadBlocklist: make(chan Blocklist, 1),
		conf:            conf,
		clientPool:      clientPool,
		queryLogger:     queryLogger,
		metrics:         makeMetrics(),
		logl:            logex.Levels(logger),
	}

	// to trigger metrics calculation
	qh.replaceBlocklist(blocklist)

	go func() {
		defer stop.Done()

		for {
			select {
			case <-stop.Signal:
				return
			case blocklist := <-qh.ReloadBlocklist:
				qh.replaceBlocklist(blocklist)
			}
		}
	}()

	return qh
}

func (h *DnsQueryHandler) Handle(rw dns.ResponseWriter, req *dns.Msg) {
	started := time.Now()

	// we don't have to support len(req.Question) > 1:
	// https://serverfault.com/questions/742785/multi-query-multiple-dns-record-types-at-once
	if len(req.Question) != 1 {
		// don't bother sending a message, even handleRejection() code
		// doesn't work if len() == 0
		h.logl.Error.Printf("request dropped: question count != 1")
		return
	}

	h.blocklistMu.Lock()
	blocklisted := h.blocklist.Has(req.Question[0].Name)
	h.blocklistMu.Unlock()

	remoteIp := ipFromAddr(rw.RemoteAddr())

	reqStatusForLogging := ""

	if h.conf.RejectQueriesByClientAddr[remoteIp] {
		reqStatusForLogging = "REJECTED BY CLIENT"
		h.metrics.requestRejectedByClient.Inc()
		h.handleRejection(rw, req)
	} else if blocklisted {
		reqStatusForLogging = "BLOCKLISTED"
		h.metrics.requestBlocklisted.Inc()
		h.handleRejection(rw, req)
	} else {
		reqStatusForLogging = "OK"
		h.metrics.requestAccepted.Inc()
		job := NewJob(req)
		h.clientPool.Jobs <- job
		resp := <-job.Response

		if err := rw.WriteMsg(resp); err != nil {
			h.logl.Error.Printf("request dropped, error writing to client: %v", err)
		}
	}

	h.queryLogger.LogQuery(req.Question[0].Name, reqStatusForLogging, remoteIp)

	h.metrics.requestCount.Inc()
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
		h.logl.Error.Printf("Error writing message: %v", err)
	}
}

func (h *DnsQueryHandler) replaceBlocklist(blocklist Blocklist) {
	h.blocklistMu.Lock()
	defer h.blocklistMu.Unlock()

	h.blocklist = blocklist
	h.metrics.blocklistItems.Set(float64(len(blocklist)))

	h.logl.Info.Printf("Got blocklist with %d item(s)", len(blocklist))
}

func runServer(handler *DnsQueryHandler, stop *stopper.Stopper) error {
	defer stop.Done()

	udpHandler := dns.NewServeMux()
	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", handler.Handle)
	udpHandler.HandleFunc(".", handler.Handle)

	tcpServer := &dns.Server{
		Addr:         "0.0.0.0:53",
		Net:          "tcp",
		Handler:      tcpHandler,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

	udpServer := &dns.Server{
		Addr:         "0.0.0.0:53",
		Net:          "udp",
		Handler:      udpHandler,
		UDPSize:      65535,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

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

func ipFromAddr(addr net.Addr) string {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	default:
		panic("unexpected addr type")
	}
}
