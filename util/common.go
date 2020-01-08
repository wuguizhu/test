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

type RspResults struct {
	Status int      `json:"status"`
	Msg    *Message `json:"message"`
	Error  string   `json:"error"`
}
type Message struct {
	IP      string        `json:"ip"`
	Region  string        `json:"region"`
	Res     []*ResMessage `json:"res"`
	Station string        `json:"station"`
}
type ResMessage struct {
	IPStatus      int            `json:"ipStatus"`
	IsPhyIP       int            `json:"isPhyIp"`
	Result        *ResultMessage `json:"result"`
	TargetIP      string         `json:"targetIP"`
	TargetRegion  string         `json:"targetRegion"`
	TargetStation string         `json:"targetStation"`
	Type          string         `json:"type"`
}
type ResultMessage struct {
	Ping    ResPing    `json:"ping"`
	TCPPing ResTcpping `json:"tcpping"`
}
type ResPing struct {
	Avgrtt     float64 `json:"avgrtt"`
	Ctime      float64 `json:"ctime"`
	Loss       int     `json:"loss"`
	Maxrtt     float64 `json:"maxrtt"`
	Minrtt     float64 `json:"minrtt"`
	Package    int     `json:"package"`
	PingAtTime string  `json:"pingAtTime"`
}
type ResTcpping struct {
	AvgRttMs    float64 `json:"avgRttMs"`
	LossPackets int     `json:"lossPackets"`
	LossRate    float64 `json:"lossRate"`
	MaxRttMs    float64 `json:"maxRttMs"`
	Mdev        float64 `json:"mdev"`
	MinRttMs    float64 `json:"minRttMs"`
	RecvPackets int     `json:"recvPackets"`
	SentPackets int     `json:"sentPackets"`
	PingAtTime  string  `json:"pingAtTime"`
}

type PingIP struct {
	IP     string
	Region string
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
