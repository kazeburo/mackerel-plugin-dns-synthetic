package main

import (
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func startTestDNSServer(t *testing.T, handler dns.Handler) (port string, shutdown func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen udp: %v", err)
	}
	server := &dns.Server{PacketConn: pc, Handler: handler}
	go func() {
		_ = server.ActivateAndServe()
	}()
	udpAddr, ok := pc.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("unexpected addr type: %T", pc.LocalAddr())
	}
	return strconv.Itoa(udpAddr.Port), func() {
		_ = server.Shutdown()
	}
}

func dnsHandler(qname string, rcode int, ip string) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		if len(r.Question) == 0 {
			m.Rcode = dns.RcodeFormatError
			_ = w.WriteMsg(m)
			return
		}
		if r.Question[0].Name != qname {
			m.Rcode = dns.RcodeNameError
			_ = w.WriteMsg(m)
			return
		}
		m.Rcode = rcode
		if rcode == dns.RcodeSuccess && ip != "" {
			rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", qname, ip))
			if err != nil {
				m.Rcode = dns.RcodeServerFailure
				_ = w.WriteMsg(m)
				return
			}
			m.Answer = []dns.RR{rr}
		}
		_ = w.WriteMsg(m)
	}
}

func TestResolveOnce(t *testing.T) {
	const qname = "example.com."
	port, shutdown := startTestDNSServer(t, dnsHandler(qname, dns.RcodeSuccess, "1.2.3.4"))
	defer shutdown()

	base := &Opt{
		Protocol: "udp",
		Port:     port,
		Question: qname,
		Timeout:  1 * time.Second,
	}

	t.Run("success", func(t *testing.T) {
		opt := *base
		opt.Expect = "1.2.3.4"
		rtt, err := opt.ResolveOnce("127.0.0.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rtt == nil {
			t.Fatalf("expected rtt, got nil")
		}
	})

	t.Run("expect mismatch", func(t *testing.T) {
		opt := *base
		opt.Expect = "9.9.9.9"
		_, err := opt.ResolveOnce("127.0.0.1")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestResolveOnceRcodeError(t *testing.T) {
	const qname = "bad.example."
	port, shutdown := startTestDNSServer(t, dnsHandler(qname, dns.RcodeServerFailure, ""))
	defer shutdown()

	opt := &Opt{
		Protocol: "udp",
		Port:     port,
		Question: qname,
		Timeout:  1 * time.Second,
	}
	_, err := opt.ResolveOnce("127.0.0.1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestFetchMetricsSuccess(t *testing.T) {
	const qname = "example.com."
	port, shutdown := startTestDNSServer(t, dnsHandler(qname, dns.RcodeSuccess, "1.2.3.4"))
	defer shutdown()

	opt := &Opt{
		Protocol: "udp",
		Port:     port,
		Hosts:    []string{"127.0.0.1"},
		Question: qname,
		Try:      2,
		Interval: 0,
		Timeout:  1 * time.Second,
	}

	metrics, err := opt.FetchMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics["available"] != 100 {
		t.Fatalf("available should be 100, got %v", metrics["available"])
	}
	if metrics["success-rate"] != 100 {
		t.Fatalf("success-rate should be 100, got %v", metrics["success-rate"])
	}
	if metrics["error"] != 0 {
		t.Fatalf("error should be 0, got %v", metrics["error"])
	}
	if metrics["success"] != 2 {
		t.Fatalf("success should be 2, got %v", metrics["success"])
	}
	if metrics["max"] < metrics["mean"] {
		t.Fatalf("max should be >= mean, got max=%v mean=%v", metrics["max"], metrics["mean"])
	}
}

func TestFetchMetricsFailure(t *testing.T) {
	const qname = "example.com."
	port, shutdown := startTestDNSServer(t, dnsHandler(qname, dns.RcodeServerFailure, ""))
	defer shutdown()

	opt := &Opt{
		Protocol: "udp",
		Port:     port,
		Hosts:    []string{"127.0.0.1"},
		Question: qname,
		Try:      2,
		Interval: 0,
		Timeout:  1 * time.Second,
	}

	metrics, err := opt.FetchMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics["available"] != 0 {
		t.Fatalf("available should be 0, got %v", metrics["available"])
	}
	if metrics["success-rate"] != 0 {
		t.Fatalf("success-rate should be 0, got %v", metrics["success-rate"])
	}
	if metrics["error"] != 2 {
		t.Fatalf("error should be 2, got %v", metrics["error"])
	}
	if metrics["success"] != 0 {
		t.Fatalf("success should be 0, got %v", metrics["success"])
	}
}
