package filter

import (
	"bufio"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
)

const (
	MAXU32 = ^uint32(0)

	MYIP = "https://api.ipify.org?format=json"
)

func init() {
	plugin.Register("filter", setup)
	log.D.Set()
}

func setup(c *caddy.Controller) error {
	c.Next() // skip plugin name

	filter := &Filter{}

	if !c.Next() {
		return c.ArgErr()
	}
	timeInterval := c.Val()
	interval, err := strconv.Atoi(timeInterval)
	if err != nil {
		return err
	}

	// if Filter contains variable length data structure
	// updates Filter concurrently might cause data inconsistent
	filter.updateLocalIP()

	files := c.RemainingArgs()

	var ipTable []IPInfo
	for i, f := range files {
		ipinfos, err := extractIPs(f)
		if err != nil {
			return err
		}

		group := uint64(i + 1)
		for i := range ipinfos {
			ipinfos[i].group = group
		}

		ipTable = append(ipTable, ipinfos...)
	}

	filter.IPTable = ipTable
	sort.Sort(filter)

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return filter
	})

	go filter.localIPUpdator(time.Duration(interval) * time.Second)
	return nil
}

func extractIPs(filename string) ([]IPInfo, error) {
	// TODO: add ip file format validation
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	var result []IPInfo
	r := bufio.NewReader(f)
	var line string
	for err == nil {
		line, err = r.ReadString('\n')
		if len(line) < 4 {
			continue
		}
		line = line[:len(line)-1]

		ipLine := strings.Split(line, "/")
		ipInfo := IP2IPInterval(ipLine[0], ipLine[1])
		result = append(result, ipInfo)
	}

	if err != io.EOF {
		return nil, err
	}

	return result, nil
}

func IP2IPInterval(ipString string, maskStr string) IPInfo {
	// ip := strings.Split(ipString, ".")
	ipInfo := IPInfo{}
	mask, _ := strconv.Atoi(maskStr)

	var curr uint32 = 0
	for _, r := range ipString {
		switch r {
		case '.':
			ipInfo.start <<= 8
			ipInfo.start += curr
			curr = 0
		default:
			curr *= 10
			curr += uint32(r - '0')
		}
	}
	ipInfo.start <<= 8
	ipInfo.start += curr
	ipInfo.start = ipInfo.start >> (32 - mask) << (32 - mask)

	ipInfo.end = ipInfo.start | (MAXU32<<mask)>>mask

	return ipInfo
}

func (f *Filter) Len() int           { return len(f.IPTable) }
func (f *Filter) Swap(i, j int)      { f.IPTable[i], f.IPTable[j] = f.IPTable[j], f.IPTable[i] }
func (f *Filter) Less(i, j int) bool { return f.IPTable[i].start < f.IPTable[j].start }
