package filter

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/miekg/dns"
)

const (
	DEFAULT_IP_GROUP = iota
	DEFAULT_LOCAL_IP_GROUP
)

type Filter struct {
	Next         plugin.Handler
	localIP      uint32 // data race but fine, read/write to int type wrong cause severe data inconsistent
	localIPGroup uint64 // data race but fine

	// iptable is only updted on startup, since then, it became read only
	// write to iptable might cause index out of range panic
	// if really need to, all reads to IPTable MUST use a new symbol to reference IPtable before access the data inside
	IPTable []IPInfo
}

type IPInfo struct {
	start uint32
	end   uint32
	group uint64 // alignment
}

var _ plugin.Handler = &Filter{}

func (f *Filter) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	nw := nonwriter.New(w)
	rcode, err := plugin.NextOrFailure(f.Name(), f.Next, ctx, nw, r)

	// only filter when success
	if rcode != dns.RcodeSuccess ||
		err != nil ||
		nw.Msg == nil {
		log.Error("forward to query plugin error:", err)
		return rcode, err
	}

	answer := make([]dns.RR, 0, len(nw.Msg.Answer))
	IPcount := 0 // MUST use IPcount instead of len(answer)
	for _, record := range nw.Msg.Answer {

		if record.Header().Class == dns.TypeA {
			rr := record.(*dns.A)
			ip := rr.A
			if f.IsGroupX(ip, f.localIPGroup) { // filter slow
				IPcount++
			} else {
				continue
			}
		}
		// MUST append() here -> some answers are CNAME
		// otherwise could lead to spurious NXDOMAIN
		answer = append(answer, record)
	}

	if IPcount > 0 {
		nw.Msg.Answer = answer
	}

	w.WriteMsg(nw.Msg)

	return rcode, err
}

func (f *Filter) Name() string { return "filter" }

type myIP struct {
	IP string `json:"ip,omitempty"`
}

func (f *Filter) localIPUpdator(interval time.Duration) {
	ticker := time.Tick(interval)
	for {
		f.updateLocalIP()
		<-ticker
	}
}

func (f *Filter) updateLocalIP() {
	// not using lock will cause data race, but won't affect correctness because assigning to uint32 is atomic
	for i := 0; i < 3; i++ {
		resp, err := http.Get(MYIP)
		if err != nil {
			log.Errorf("call ip api error: %v", err)
			time.Sleep(10 * time.Second)
		} else if resp.StatusCode != 200 {
			log.Errorf("call ip api statuscode: %d", resp.StatusCode)
			resp.Body.Close()
			time.Sleep(10 * time.Second)
		} else {
			myip := myIP{}

			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&myip)
			resp.Body.Close()

			if err == nil { // call api succeed, update local ip
				ip := net.ParseIP(myip.IP)
				f.localIP = IP2Int(ip)
				localIPGroup := f.GetGroupOfIP(ip)

				// seperate default local group and defautl group
				// ensures that if local group is not is configuration, no dns answer is gonna be filter
				if localIPGroup != DEFAULT_IP_GROUP {
					f.localIPGroup = localIPGroup
				} else {
					f.localIPGroup = DEFAULT_LOCAL_IP_GROUP
				}

				break
			} else {
				log.Errorf("update ip unmarshal error: %v", err)
			}
		}
	}
}

// GetGroupOfIP returns group of ip base on config
// default is DEFAULT_IP_GROUP(i.e. 0)
func (f *Filter) GetGroupOfIP(ip net.IP) uint64 {
	uIP := IP2Int(ip)

	var l = 0
	var r = len(f.IPTable) - 1
	for l <= r {
		var mid = int((l + r) / 2)
		if uIP < f.IPTable[mid].start {
			r = mid - 1
		} else if uIP > f.IPTable[mid].end {
			l = mid + 1
		} else {
			return f.IPTable[mid].group
		}
	}

	return DEFAULT_IP_GROUP
}

func (f *Filter) IsGroupX(ip net.IP, group uint64) bool {
	g := f.GetGroupOfIP(ip)
	return g == group
}
