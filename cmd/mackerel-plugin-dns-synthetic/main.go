package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/miekg/dns"
	"github.com/montanaflynn/stats"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	StatusCodeOK      = 0
	StatusCodeWARNING = 1
)

// version by Makefile
var version string

type Opt struct {
	Version bool   `short:"v" long:"version" description:"Show version"`
	Prefix  string `long:"prefix" default:"dnsdist" description:"Metric key prefix"`

	Protocol string        `long:"protocol" required:"true" default:"udp" choice:"tcp" choice:"udp"`
	Port     string        `short:"p" long:"port" default:"53" description:"Port number"`
	Hosts    []string      `short:"H" long:"hostname" default:"127.0.0.1" description:"DNS server hostnames"`
	Question string        `short:"Q" long:"question" default:"example.com." description:"Question hostname"`
	Expect   string        `short:"E" long:"expect" default:"" description:"Expect string in result"`
	Timeout  time.Duration `long:"timeout" default:"5s" description:"Timeout"`
	Try      int           `long:"try" default:"1" description:"Number of resoluitions"`
}

func (o *Opt) MetricKeyPrefix() string {
	if o.Prefix == "" {
		return "dns-synthetic"
	}
	return o.Prefix + "-synthetic"
}

func (o *Opt) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := cases.Title(language.Und, cases.NoLower).String(o.MetricKeyPrefix())
	return map[string]mp.Graphs{
		"service": {
			Label: labelPrefix + ": Available",
			Unit:  mp.UnitPercentage,
			Metrics: []mp.Metrics{
				{Name: "available", Label: "available", Diff: false, Stacked: true},
			},
		},
		"status": {
			Label: labelPrefix + ": resolve status count",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "error", Label: "Error", Diff: false, Stacked: true},
				{Name: "success", Label: "Success", Diff: false, Stacked: true},
			},
		},
		"rtt": {
			Label: labelPrefix + ": RTT",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "max", Label: "Max", Diff: false, Stacked: false},
				{Name: "mean", Label: "Mean", Diff: false, Stacked: false},
			},
		},
	}
}

func (o *Opt) ResolveOnce(host string) (*time.Duration, error) {
	c := &dns.Client{Net: o.Protocol, Timeout: o.Timeout}
	address := net.JoinHostPort(host, o.Port)
	m := new(dns.Msg)
	m.SetQuestion(o.Question, dns.StringToType["A"])
	r, rtt, err := c.Exchange(m, address)
	if err != nil {
		return nil, err
	}
	if r.Rcode != dns.RcodeSuccess {
		return &rtt, fmt.Errorf("failed to resolve '%s'. rcode:%s",
			o.Question,
			dns.RcodeToString[r.Rcode],
		)
	}
	answer := make([]string, 0)
	for _, a := range r.Answer {
		if aa, ok := a.(*dns.A); ok {
			answer = append(answer, aa.A.String())
		}
	}
	if len(o.Expect) > 0 && !strings.Contains(strings.Join(answer, "|"), o.Expect) {
		return &rtt, fmt.Errorf("dns answer does not contain '%s' in '%s'",
			o.Expect,
			strings.Join(answer, "\t"))
	}
	return &rtt, nil
}

type response struct {
	rtt  *time.Duration
	host string
	err  error
}

func (o *Opt) FetchMetrics() (map[string]float64, error) {
	c := make(chan *response, len(o.Hosts)*o.Try)
	for _, host := range o.Hosts {
		h := host
		go func() {
			for i := 0; i < o.Try; i++ {
				rtt, err := o.ResolveOnce(h)
				c <- &response{
					rtt:  rtt,
					host: h,
					err:  err,
				}
			}
		}()
	}
	onError := float64(0)
	onSuccess := float64(0)
	rtts := make([]float64, 0)
	for i := 0; i < o.Try; i++ {
		for range o.Hosts {
			res := <-c
			if res.rtt != nil {
				rtts = append(rtts, float64(res.rtt.Milliseconds()))
			}
			if res.err != nil {
				log.Printf("%v on %s:%s", res.err, res.host, o.Port)
				onError++
				continue
			}
			onSuccess++
		}
	}
	result := map[string]float64{}
	if onSuccess > 0 {
		result["available"] = 100
	} else {
		result["available"] = 0
	}
	result["error"] = onError
	result["success"] = onSuccess
	if len(rtts) > 0 {
		result["mean"], _ = stats.Mean(rtts)
		result["max"], _ = stats.Max(rtts)
	}
	return result, nil
}

func (o *Opt) Run() {
	plugin := mp.NewMackerelPlugin(o)
	plugin.Run()
}

func main() {
	opt := &Opt{}
	psr := flags.NewParser(opt, flags.HelpFlag|flags.PassDoubleDash)
	_, err := psr.Parse()
	if opt.Version {
		fmt.Printf(`%s %s
Compiler: %s %s
`,
			os.Args[0],
			version,
			runtime.Compiler,
			runtime.Version())
		os.Exit(StatusCodeOK)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(StatusCodeWARNING)
	}

	opt.Run()
}
