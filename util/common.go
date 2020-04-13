package util

import (
	"math/rand"
	"net"

	"github.com/astaxie/beego/logs"
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
type ReqSpeedtest struct {
	Station string         `json:"Station,omitempty"`
	IP      string         `json:"Ip,omitempty"`
	Region  string         `json:"Region,omitempty"`
	IPs     []*SpeedtestIP `json:"IPs,omitempty"`
}
type SpeedtestIP struct {
	Country   string `json:"country,omitempty"`
	Provider  string `json:"provider,omitempty"`
	City      string `json:"city,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      string `json:"port,omitempty"`
	Latitude  string `json:"latitude,omitempty"`
	Longitude string `json:"longitude,omitempty"`
}
type RspSpeedtestResults struct {
	Status int               `json:"status"`
	Msg    *SpeedtestMessage `json:"message,omitempty"`
	Error  string            `json:"error,omitempty"`
}
type SpeedtestMessage struct {
	IP         string                 `json:"ip,omitempty"`
	Region     string                 `json:"region,omitempty"`
	Res        []*SpeedtestResMessage `json:"res,omitempty"`
	Station    string                 `json:"station,omitempty"`
	PingSpTime string                 `json:"pingSpTime,omitempty"`
}
type SpeedtestResMessage struct {
	Result    *ResultMessage `json:"result,omitempty"`
	Country   string         `json:"country,omitempty"`
	Provider  string         `json:"provider,omitempty"`
	City      string         `json:"city,omitempty"`
	Host      string         `json:"host,omitempty"`
	Port      string         `json:"port,omitempty"`
	IP        string         `json:"ip,omitempty"`
	Latitude  string         `json:"latitude,omitempty"`
	Longitude string         `json:"longitude,omitempty"`
	Type      string         `json:"type,omitempty"`
}
type RspResults struct {
	Status int      `json:"status"`
	Msg    *Message `json:"message,omitempty"`
	Error  string   `json:"error"`
}
type Message struct {
	IP           string        `json:"ip"`
	Region       string        `json:"region"`
	Res          []*ResMessage `json:"res"`
	Station      string        `json:"station"`
	PingRTime    string        `json:"pingRTime,omitempty"`
	PingSTime    string        `json:"pingSTime,omitempty"`
	TCPPingSTime string        `json:"tcppingSTime,omitempty"`
}
type ResMessage struct {
	IPStatus      int            `json:"ipStatus,omitempty"`
	IsPhyIP       int            `json:"isPhyIp,omitempty"`
	Result        *ResultMessage `json:"result"`
	TargetIP      string         `json:"targetIP"`
	TargetRegion  string         `json:"targetRegion"`
	TargetStation string         `json:"targetStation,omitempty"`
	Type          string         `json:"type"`
}

// 此处使用指针，故注意要初始化零值
type ResultMessage struct {
	Ping    *ResPing    `json:"ping,omitempty"`
	TCPPing *ResTcpping `json:"tcpping,omitempty"`
}
type ResPing struct {
	Avgrtt  float64 `json:"avgrtt"`
	Ctime   float64 `json:"ctime"`
	Loss    int     `json:"loss"`
	Maxrtt  float64 `json:"maxrtt"`
	Minrtt  float64 `json:"minrtt"`
	Package int     `json:"package"`
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
}

type PingIP struct {
	IP            string
	Region        string
	TargetStation string
	IPStatus      int
	IsPhyIP       int
	Country       string
	Provider      string
	City          string
	Host          string
	Port          string
	Latitude      string
	Longitude     string
}
type IPsGetter interface {
	GetIPs() []*PingIP
}

func (stationIPs *ReqStation) GetIPs() (ips []*PingIP) {
	ips = make([]*PingIP, 0, len(stationIPs.IPs))
	for _, stationIP := range stationIPs.IPs {
		// logs.Debug("station ip %d:%v", i, stationIP.IP)
		pingIP := &PingIP{
			IP:            stationIP.IP,
			Region:        stationIP.Region,
			TargetStation: stationIP.TargetStation,
			IPStatus:      stationIP.IPStatus,
			IsPhyIP:       stationIP.IsPhyIP,
		}
		ips = append(ips, pingIP)
	}
	return ips
}
func (speedtestIPs *ReqSpeedtest) GetIPs() (ips []*PingIP) {
	ipChannel := make(chan PingIP, len(speedtestIPs.IPs))
	for _, speedtestIP := range speedtestIPs.IPs {
		go Host2IP_Producer(speedtestIP, ipChannel)
	}
	close(ipChannel)
	ips = Host2IP_Customer(ipChannel)
	logs.Info("Finish Parsing %d host to ip", len(ips))
	return
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
func Host2IP_Producer(speedtestIP *SpeedtestIP, ipChannel chan<- PingIP) {
	IPAddr, err := net.ResolveIPAddr("ip", speedtestIP.Host)
	if err != nil {
		logs.Error("Resolution host:%s to ip error %s", speedtestIP.Host, err)
	} else {
		logs.Debug("Resolution host:%s to ip successfully %s", speedtestIP.Host, IPAddr.String())
	}
	pingIP := PingIP{
		IP:        IPAddr.String(),
		Host:      speedtestIP.Host,
		Provider:  speedtestIP.Provider,
		Country:   speedtestIP.Country,
		City:      speedtestIP.City,
		Port:      speedtestIP.Port,
		Longitude: speedtestIP.Longitude,
		Latitude:  speedtestIP.Latitude,
	}
	ipChannel <- pingIP
}
func Host2IP_Customer(ipChannel <-chan PingIP) (ips []*PingIP) {
	for pingIP := range ipChannel {
		ips = append(ips, &pingIP)
	}
	return
}
