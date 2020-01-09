package util

import (
	"math/rand"
)

const (
	//MaxRtt 没有返回ping结果的rtt
	MaxRtt int = 1000
)

type ReqRegion struct {
	Station string      `json:"Station"`
	IP      string      `json:"Ip"`
	Region  string      `json:"Region"`
	IPs     []*RegionIP `json:"IPs"`
}
type RegionIP struct {
	IP     string `json:"ip"`
	Region string `json:"region"`
}

type ReqStation struct {
	Station string       `json:"Station"`
	IP      string       `json:"Ip"`
	Region  string       `json:"Region"`
	IPs     []*StationIP `json:"IPs"`
}
type StationIP struct {
	IP            string `json:"ip"`
	Region        string `json:"region"`
	TargetStation string `json:"station"`
	IPStatus      int    `json:"ipStatus"`
	IsPhyIP       int    `json:"isPhyIp"`
}
type PingIP struct {
	IP            string
	Region        string
	TargetStation string
	IPStatus      int
	IsPhyIP       int
}
type IPsGetter interface {
	GetIPs() []*PingIP
}

func (stationIPs *ReqStation) GetIPs() (ips []*PingIP) {
	ips = make([]*PingIP, 0, len(stationIPs.IPs))
	for _, stationIP := range stationIPs.IPs {
		// logs.Debug("station ip %d:%v", i, stationIP.IP)
		pingIP := &PingIP{
			IP:     stationIP.IP,
			Region: stationIP.Region,
		}
		ips = append(ips, pingIP)
	}
	return ips
}
func (regionIPs *ReqRegion) GetIPs() (ips []*PingIP) {
	ips = make([]*PingIP, 0, len(regionIPs.IPs))
	for _, regionIP := range regionIPs.IPs {
		// logs.Debug("reagion ip %d:%v", i, regionIP.IP)
		pingIP := &PingIP{
			IP:     regionIP.IP,
			Region: regionIP.Region,
		}
		ips = append(ips, pingIP)
	}
	return ips
}
func GetLocalPort() int {
	return rand.Intn(30000) + 30000
}
