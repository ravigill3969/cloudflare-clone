package ddos

import (
	"net"
	"sync"
)

type DDoS struct {
	IpMutex sync.RWMutex
	IpCount map[string]int
	limit   int
}

func NewDDoS(limit int) *DDoS {
	return &DDoS{
		IpCount: make(map[string]int),
		limit:   limit,
	}
}

func (d *DDoS) AddRequest(ip net.IP) {
	d.IpMutex.Lock()
	defer d.IpMutex.Unlock()
	d.IpCount[ip.String()]++
}

func (d *DDoS) IsSuspicious(ip net.IP) bool {
	d.IpMutex.RLock()
	defer d.IpMutex.RUnlock()
	return d.IpCount[ip.String()] > d.limit
}

func (d *DDoS) Stats() map[string]int {
	d.IpMutex.RLock()
	defer d.IpMutex.RUnlock()

	stats := make(map[string]int)
	for k, v := range d.IpCount {
		stats[k] = v
	}
	return stats
}
