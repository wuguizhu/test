package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"

	"github.com/astaxie/beego/logs"
)

const (
	MaxRtt int = 1000 //没有返回ping结果的rtt
)

// RespRegion is response fron /test/regionips
type RespRegion struct {
	Status  int        `json:"status"`
	Message *RegionIPs `json:"message"`
	Error   string     `json:"error"`
}
type RegionIPs struct {
	Station string      `json:"Station"`
	IP      string      `json:"Ip"`
	Region  string      `json:"Region"`
	IPs     []*RegionIP `json:"IPs"`
}
type RegionIP struct {
	IP     string `json:"ip"`
	Region string `json:"region"`
}

// RespStation is response fron /test/stationips
type RespStation struct {
	Status  int         `json:"status"`
	Message *StationIPs `json:"message"`
	Error   string      `json:"error"`
}
type StationIPs struct {
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

type ReqForm struct {
	Status int     `json:"status"`
	Msg    Message `json:"message"`
	Error  string  `json:"error"`
}
type Message struct {
	IP      string       `json:"ip"`
	Region  string       `json:"region"`
	Res     []ResMessage `json:"res"`
	Station string       `json:"station"`
}
type ResMessage struct {
	IPStatus      int           `json:"ipStatus"`
	IsPhyIP       int           `json:"isPhyIp"`
	Result        ResultMessage `json:"result"`
	TargetIP      string        `json:"targetIP"`
	TargetRegion  string        `json:"targetRegion"`
	TargetStation string        `json:"targetStation"`
	Type          string        `json:"type"`
}
type ResultMessage struct {
	Ping    ResPing    `json:"ping"`
	TCPPing ResTcpping `json:"tcpping"`
}
type ResPing struct {
	Avgrtt  float64 `json:"avgrtt"`
	Ctime   string  `json:"ctime"`
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
type PostRes struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}
type PingIP struct {
	IP     string
	Region string
}
type IPsGetter interface {
	GetIPs() []*PingIP
}

func (stationIPs *StationIPs) GetIPs() (ips []*PingIP) {
	ips = make([]*PingIP, 0, len(stationIPs.IPs))
	for _, stationIP := range stationIPs.IPs {
		logs.Debug(stationIP.IP)
		pingIP := &PingIP{
			IP:     stationIP.IP,
			Region: stationIP.Region,
		}
		ips = append(ips, pingIP)
	}
	return ips
}
func (regionIPs *RegionIPs) GetIPs() (ips []*PingIP) {
	logs.Debug(*regionIPs)
	ips = make([]*PingIP, 0, len(regionIPs.IPs))
	for _, regionIP := range regionIPs.IPs {
		logs.Debug(regionIP.IP)
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

func GetRegionIPs(url, staion, sip string) (*RegionIPs, error) {
	resp, err := http.Get(url + "?station=" + staion + "&sip=" + sip)
	if err != nil {
		logs.Error("httpGET failed with error:", err)
		return nil, err
	}
	defer resp.Body.Close()
	logs.Debug("resp:%s", url+"?station="+staion+"&sip="+sip)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error("ioutil.ReadAll failed with error:", err)
		return nil, err
	}
	var r RespRegion
	err = json.Unmarshal(body, &r)
	if err != nil {
		logs.Error(" json.Unmarshal failed with error:", err)
		return nil, err
	}
	if r.Status != 0 {
		err = fmt.Errorf("r.Status=%d and r.Err:%s", r.Status, err)
		logs.Error("resp failed with error:", err)
		return nil, err
	}
	if r.Message == nil {
		err = fmt.Errorf("r.Message is nil")
		logs.Error("resp failed with error:", err)
		return nil, err
	}
	return r.Message, nil
}

func GetStationIPs(url, staion, sip string) (*StationIPs, error) {
	resp, err := http.Get(url + "?station=" + staion + "&sip=" + sip)
	if err != nil {
		logs.Error("httpGET failed with error:", err)
		return nil, err
	}
	defer resp.Body.Close()
	logs.Debug("resp:%s", url+"?station="+staion+"&sip="+sip)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error("ioutil.ReadAll failed with error:", err)
		return nil, err
	}
	var r RespStation
	err = json.Unmarshal(body, &r)
	if err != nil {
		logs.Error(" json.Unmarshal failed with error:", err)
		return nil, err
	}
	if r.Status != 0 {
		err = fmt.Errorf("r.Status=%d and r.Err:%s", r.Status, err)
		logs.Error("resp failed with error:", err)
		return nil, err
	}
	if r.Message == nil {
		err = fmt.Errorf("r.Message is nil")
		logs.Error("resp failed with error:", err)
		return nil, err
	}
	return r.Message, nil
}
