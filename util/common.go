package util

import (
	"math/rand"
	"net"
	"sync"
	"testnode-pinger/go-cache"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

const (
	//MaxRtt 没有返回ping结果的rtt
	MaxRtt int = 1000
)

var c = getNewCache()

func getNewCache() *cache.Cache {
	// create a new cache with a default 10min expiration duration and 5min cleanup interval.
	return cache.New(10*time.Minute, 5*time.Minute)
}

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
	maxDNSLimit := beego.AppConfig.DefaultInt64("max_DNS_limit", 512)
	ipChannel := make(chan *PingIP, len(speedtestIPs.IPs))
	wg := &sync.WaitGroup{}
	wg.Add(len(speedtestIPs.IPs))
	limitChan := make(chan bool, maxDNSLimit)
	defer close(limitChan)
	for _, speedtestIP := range speedtestIPs.IPs {
		limitChan <- true
		go Host2IP_Producer(speedtestIP, wg, ipChannel, limitChan)
	}
	wg.Wait()
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
func Host2IP_Producer(speedtestIP *SpeedtestIP, wg *sync.WaitGroup, ipChannel chan<- *PingIP, limitChan <-chan bool) {
	var ip string
	// get ip from cache firstly,if not exist,then do net.ResolveIPAddr and save result into cache
	// expire at random time (60~240min) to avoid cache avalanche
	ipInterface, found := c.Get(speedtestIP.Host)
	if found {
		ip = ipInterface.(string)
		logs.Debug("get ip %s:%s from cache successfully,no need Resolution host", speedtestIP.Host, ip)

	} else {
		IPAddr, err := net.ResolveIPAddr("ip", speedtestIP.Host)
		if err != nil {
			logs.Error("Resolution host:%s to ip error %s", speedtestIP.Host, err)
			return
		}
		ip = IPAddr.String()
		c.Set(speedtestIP.Host, ip, time.Duration((60+rand.Intn(180)))*time.Minute)
		logs.Debug("Resolution host:%s to ip successfully %s", speedtestIP.Host, IPAddr.String())
	}
	pingIP := &PingIP{
		IP:        ip,
		Host:      speedtestIP.Host,
		Provider:  speedtestIP.Provider,
		Country:   speedtestIP.Country,
		City:      speedtestIP.City,
		Port:      speedtestIP.Port,
		Longitude: speedtestIP.Longitude,
		Latitude:  speedtestIP.Latitude,
	}
	ipChannel <- pingIP
	wg.Done()
	<-limitChan
}
func Host2IP_Customer(ipChannel <-chan *PingIP) (ips []*PingIP) {
	// 使用for-range读取channel，这样既安全又便利，当channel关闭时，for循环会自动退出，无需主动监测channel是否关闭
	for pingIP := range ipChannel {
		ips = append(ips, pingIP)
		logs.Debug("append %v to ips", pingIP)
	}
	return
}
