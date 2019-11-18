package filter

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"

	"github.com/miekg/dns"
)

type Filter struct {
	Next plugin.Handler
}

func (f Filter) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	nw := nonwriter.New(w)
	rcode, err := plugin.NextOrFailure(f.Name(), f.Next, ctx, nw, r)

	// only filter when success
	if rcode != dns.RcodeSuccess ||
		err != nil ||
		nw.Msg == nil {
		return rcode, err
	}

	answer := make([]dns.RR, 0, len(nw.Msg.Answer))
	Acount := 0 // MUST use Acount instead of len(answer)
	for _, record := range nw.Msg.Answer {
		rr, ok := record.(*dns.A)
		if ok {
			ip := rr.A
			if IsSlowIP(ip) { // filter slow
				continue
			} else {
				Acount++
			}
		}
		// MUST append() here -> some answers are CNAME
		// otherwise could lead to spurious NXDOMAIN
		answer = append(answer, record)
	}

	if Acount > 0 {
		nw.Msg.Answer = answer
	}

	w.WriteMsg(nw.Msg)

	return rcode, err
}

func (f Filter) Name() string { return "filter" }
