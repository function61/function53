package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/function61/gokit/log/logex"
	"github.com/function61/gokit/sync/syncutil"
	"github.com/function61/gokit/sync/taskrunner"
	"github.com/miekg/dns"
)

type DnsQueryHandler struct {
	conf          Config
	forwarderPool *ForwarderPool
	queryLogger   QueryLogger
	blocklist     Blocklist
	blocklistMu   sync.Mutex
	metrics       *metrics
	logl          *logex.Leveled
}

func NewDnsQueryHandler(
	forwarderPool *ForwarderPool,
	conf Config,
	blocklist Blocklist,
	queryLogger QueryLogger,
	logger *log.Logger,
) *DnsQueryHandler {
	qh := &DnsQueryHandler{
		conf:          conf,
		forwarderPool: forwarderPool,
		queryLogger:   queryLogger,
		metrics:       makeMetrics(),
		logl:          logex.Levels(logger),
	}

	// to trigger metrics calculation
	qh.replaceBlocklist(blocklist)

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

	perClientConf, found := h.conf.OverridesByClientAddr[remoteIp]
	if !found {
		perClientConf = h.conf.DefaultOverridableConfig
	}

	if perClientConf.RejectAllQueries {
		reqStatusForLogging = "REJECTED BY CLIENT"
		h.metrics.requestRejectedByClient.Inc()
		h.handleRejection(rw, req)
	} else if blocklisted && !perClientConf.DisableBlocklisting {
		reqStatusForLogging = "BLOCKLISTED"
		h.metrics.requestBlocklisted.Inc()
		h.handleRejection(rw, req)
	} else {
		reqStatusForLogging = "OK"
		h.metrics.requestAccepted.Inc()
		job := NewJob(req)
		h.forwarderPool.Jobs <- job
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
	defer syncutil.LockAndUnlock(&h.blocklistMu)()

	h.blocklist = blocklist
	h.metrics.blocklistItems.Set(float64(len(blocklist)))

	h.logl.Info.Printf("Got blocklist with %d item(s)", len(blocklist))
}

func runDnsListener(ctx context.Context, handler *DnsQueryHandler, logger *log.Logger) error {
	listenAndServeDns := func(ctx context.Context, network string) error {
		srv := &dns.Server{
			Addr:         "0.0.0.0:53",
			Net:          network,
			Handler:      dns.HandlerFunc(handler.Handle),
			UDPSize:      65535,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		}

		go func() {
			<-ctx.Done()
			srv.Shutdown()
		}()

		return srv.ListenAndServe()
	}

	tasks := taskrunner.New(ctx, logger)

	tasks.Start("udp", func(ctx context.Context) error {
		return listenAndServeDns(ctx, "udp")
	})

	tasks.Start("tcp", func(ctx context.Context) error {
		return listenAndServeDns(ctx, "tcp")
	})

	return tasks.Wait()
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
