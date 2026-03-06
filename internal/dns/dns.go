package dns

import (
	"fmt"
	"net"
	"strings"

	mdns "github.com/miekg/dns"
)

// Server is an embedded DNS server that resolves *.test domains to 127.0.0.1.
type Server struct {
	udpServer *mdns.Server
	tcpServer *mdns.Server
}

// handler responds to DNS queries.
type handler struct{}

func (h *handler) ServeDNS(w mdns.ResponseWriter, r *mdns.Msg) {
	msg := new(mdns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	for _, q := range r.Question {
		if q.Qtype == mdns.TypeA && strings.HasSuffix(strings.ToLower(q.Name), ".test.") {
			msg.Answer = append(msg.Answer, &mdns.A{
				Hdr: mdns.RR_Header{
					Name:   q.Name,
					Rrtype: mdns.TypeA,
					Class:  mdns.ClassINET,
					Ttl:    60,
				},
				A: net.ParseIP("127.0.0.1"),
			})
		}
	}

	if len(msg.Answer) == 0 {
		msg.Rcode = mdns.RcodeNameError
	}

	_ = w.WriteMsg(msg)
}

// New creates a DNS server that listens on the given address (e.g., "127.0.0.1:53").
func New(addr string) *Server {
	h := &handler{}

	return &Server{
		udpServer: &mdns.Server{Addr: addr, Net: "udp", Handler: h},
		tcpServer: &mdns.Server{Addr: addr, Net: "tcp", Handler: h},
	}
}

// Start begins serving DNS on both UDP and TCP. It blocks until the servers are ready.
func (s *Server) Start() error {
	udpReady := make(chan struct{})
	tcpReady := make(chan struct{})
	errCh := make(chan error, 2)

	s.udpServer.NotifyStartedFunc = func() { close(udpReady) }
	s.tcpServer.NotifyStartedFunc = func() { close(tcpReady) }

	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("udp dns server: %w", err)
		}
	}()

	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("tcp dns server: %w", err)
		}
	}()

	// Wait for both servers to be ready, or for an error.
	for i := 0; i < 2; i++ {
		select {
		case <-udpReady:
			udpReady = nil
		case <-tcpReady:
			tcpReady = nil
		case err := <-errCh:
			return err
		}
	}

	return nil
}

// Stop gracefully shuts down both UDP and TCP servers.
func (s *Server) Stop() error {
	var firstErr error
	if err := s.udpServer.Shutdown(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := s.tcpServer.Shutdown(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}
