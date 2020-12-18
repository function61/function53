package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
	"github.com/function61/gokit/throttle"
	"github.com/miekg/dns"
)

type Job struct {
	Request  *dns.Msg
	Response chan *dns.Msg
}

func NewJob(req *dns.Msg) *Job {
	return &Job{
		Request:  req,
		Response: make(chan *dns.Msg, 1),
	}
}

type ServerEndpoint struct {
	ServerName string
	Addr       string
}

// forwards requests (in encrypted form) to DNS server endpoints which will actually
// do the job of answering our queries
type ForwarderPool struct {
	Jobs                  chan *Job
	Reconnect             chan ServerEndpoint
	tlsClientSessionCache tls.ClientSessionCache
	logl                  *logex.Leveled
}

func NewForwarderPool(endpoints []ServerEndpoint, logger *log.Logger, stop *stopper.Stopper) *ForwarderPool {
	pool := &ForwarderPool{
		Jobs:                  make(chan *Job, 16),
		Reconnect:             make(chan ServerEndpoint, len(endpoints)),
		tlsClientSessionCache: tls.NewLRUClientSessionCache(0),
		logl:                  logex.Levels(logger),
	}

	for _, endpoint := range endpoints {
		pool.Reconnect <- endpoint
	}

	go func() {
		defer stop.Done()
		defer pool.logl.Info.Println("Stopped")

		twoTimesASecond, cancel := throttle.BurstThrottler(2, 1*time.Second)
		defer cancel()

		// this loop will reconnect all broken connections
		for {
			select {
			case endpoint := <-pool.Reconnect:
				twoTimesASecond(func() {
					pool.logl.Info.Printf("Reconnecting to %s", endpoint.Addr)

					go endpointWorker(endpoint, pool)
				})
			case <-stop.Signal:
				return
			}

		}
	}()

	return pool
}

// inspired by: https://github.com/artyom/dot
func endpointWorker(endpoint ServerEndpoint, pool *ForwarderPool) {
	reconnect := func(err error) {
		pool.logl.Error.Printf("Endpoint %s failed: %v", endpoint.Addr, err)

		pool.Reconnect <- endpoint
	}

	var d net.Dialer
	tcpConn, err := d.DialContext(context.TODO(), "tcp", endpoint.Addr)
	if err != nil {
		reconnect(err)
		return
	}

	if err := tcpkeepalive.Enable(tcpConn.(*net.TCPConn), 1*time.Second); err != nil {
		pool.logl.Error.Printf("tcpkeepalive: %v", err)
	}

	tlsConn := tls.Client(tcpConn, &tls.Config{
		ServerName:         endpoint.ServerName,
		ClientSessionCache: pool.tlsClientSessionCache,
	})

	for job := range pool.Jobs {
		resp, err := dnsRequestResponse(job.Request, tlsConn)
		if err != nil {
			reconnect(err)
			pool.Jobs <- job
			return
		}

		job.Response <- resp
	}
}

// there's an API in miekg/dns that does this, but it forcefully emits deprecation message
// to stderr and the design philosophy in the docs hints that we should implement things
// like these ourselves......
func dnsRequestResponse(req *dns.Msg, conn net.Conn) (*dns.Msg, error) {
	dnsConn := dns.Conn{
		Conn: conn,
	}

	if err := dnsConn.WriteMsg(req); err != nil {
		return nil, err
	}

	res, err := dnsConn.ReadMsg()
	if err != nil {
		return nil, err
	}

	if res.Id != req.Id {
		return nil, dns.ErrId
	}

	return res, nil
}
