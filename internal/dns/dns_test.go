package dns

import (
	"fmt"
	"net"
	"testing"

	mdns "github.com/miekg/dns"
)

func startTestServer(t *testing.T) (string, func()) {
	t.Helper()

	// Use port 0 to let the OS pick a free port.
	// miekg/dns doesn't support port 0 directly, so we find a free port first.
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve udp addr: %v", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	port := udpConn.LocalAddr().(*net.UDPAddr).Port
	_ = udpConn.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	srv := New(addr)

	if err := srv.Start(); err != nil {
		t.Fatalf("start dns server: %v", err)
	}

	return addr, func() {
		_ = srv.Stop()
	}
}

func query(t *testing.T, addr, name string, qtype uint16, net string) *mdns.Msg {
	t.Helper()
	c := new(mdns.Client)
	c.Net = net
	m := new(mdns.Msg)
	m.SetQuestion(mdns.Fqdn(name), qtype)
	r, _, err := c.Exchange(m, addr)
	if err != nil {
		t.Fatalf("dns exchange (%s): %v", net, err)
	}
	return r
}

func TestDotTestResolvesToLoopback_UDP(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	r := query(t, addr, "myapp.test", mdns.TypeA, "udp")

	if len(r.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(r.Answer))
	}

	a, ok := r.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", r.Answer[0])
	}
	if !a.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("expected 127.0.0.1, got %s", a.A)
	}
}

func TestDotTestResolvesToLoopback_TCP(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	r := query(t, addr, "myapp.test", mdns.TypeA, "tcp")

	if len(r.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(r.Answer))
	}

	a, ok := r.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", r.Answer[0])
	}
	if !a.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("expected 127.0.0.1, got %s", a.A)
	}
}

func TestSubdomainResolvesToLoopback(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	r := query(t, addr, "api.myapp.test", mdns.TypeA, "udp")

	if len(r.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(r.Answer))
	}

	a, ok := r.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", r.Answer[0])
	}
	if !a.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("expected 127.0.0.1, got %s", a.A)
	}
}

func TestNonTestDomainReturnsNXDOMAIN(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	r := query(t, addr, "example.com", mdns.TypeA, "udp")

	if r.Rcode != mdns.RcodeNameError {
		t.Errorf("expected NXDOMAIN (rcode %d), got rcode %d", mdns.RcodeNameError, r.Rcode)
	}
	if len(r.Answer) != 0 {
		t.Errorf("expected 0 answers, got %d", len(r.Answer))
	}
}

func TestNonAQueryReturnsNXDOMAIN(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	r := query(t, addr, "myapp.test", mdns.TypeAAAA, "udp")

	if r.Rcode != mdns.RcodeNameError {
		t.Errorf("expected NXDOMAIN (rcode %d), got rcode %d", mdns.RcodeNameError, r.Rcode)
	}
	if len(r.Answer) != 0 {
		t.Errorf("expected 0 answers, got %d", len(r.Answer))
	}
}

func TestCaseInsensitive(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	r := query(t, addr, "MyApp.TEST", mdns.TypeA, "udp")

	if len(r.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(r.Answer))
	}

	a, ok := r.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", r.Answer[0])
	}
	if !a.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("expected 127.0.0.1, got %s", a.A)
	}
}
