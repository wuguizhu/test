package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"testnode-pinger/ping"
	"testnode-pinger/util"
	"time"

	"github.com/astaxie/beego/logs"
)

var ips *IPs

type IPs struct {
	RegionIPs  *util.RegionIPs
	StationIPs *util.StationIPs
	RegionMu   *sync.RWMutex
	StationMu  *sync.RWMutex
}

func NewIPs(conf *util.Conf, sip string) (*IPs, error) {
	regionIPs, err := util.GetRegionIPs(conf.IPRegionGetUrl, conf.Station, sip)
	if err != nil {
		logs.Error("GetRegionIPs fails with error:", err)
		return nil, err
	}
	stationIPs, err := util.GetStationIPs(conf.IPStationGetUrl, conf.Station, sip)
	if err != nil {
		logs.Error("GetStationIPs fails with error:", err)
		return nil, err
	}
	return &IPs{
		RegionIPs:  regionIPs,
		StationIPs: stationIPs,
		RegionMu:   new(sync.RWMutex),
		StationMu:  new(sync.RWMutex),
	}, nil
}

func main() {
	configFile := "./conf/testnodeprober.json"
	conf, err := util.GetConf(configFile)
	if err != nil {
		logs.Error("GetConf fails with error:", err)
		return
	}
	logs.Debug("conf: %#v", conf)
	logs.SetLevel(conf.LogLevel)
	// get sip
	srcIPs, err := util.GetIPv4LocalIP()
	if err != nil {
		return
	}
	if len(srcIPs) == 0 {
		srcIPs = append(srcIPs, "")
	}
	sip := srcIPs[0]
	logs.Debug("local ip is ", sip)
	// request to update ips per 10s
	// regionIPs = new(util.RegionIPs)
	// stationIPs = new(util.StationIPs)
	// updateIPs(conf, sip)
	// start := make(chan bool)
	ips, _ := NewIPs(conf, sip)
	go ips.updateIPs(conf, sip)
	ips.pingRun(conf, sip)

}
func (ips *IPs) updateIPs(conf *util.Conf, sip string) {
	count := 0
	for {
		count++
		logs.Info("request times:", count)
		timer := time.NewTimer(time.Duration(conf.APIRequestInterval) * time.Second)
		// request regionIPs
		ips.RegionMu.Lock()
		var err error
		ips.RegionIPs, err = util.GetRegionIPs(conf.IPRegionGetUrl, conf.Station, sip)
		if err != nil {
			logs.Error("GetRegionIPs fails with error:", err)
			return
		}
		ips.RegionMu.Unlock()
		logs.Debug("get regionips:%#v", *ips.RegionIPs)

		// request stationIPs
		ips.StationMu.Lock()
		ips.StationIPs, err = util.GetStationIPs(conf.IPStationGetUrl, conf.Station, sip)
		if err != nil {
			logs.Error("GetStationIPs fails with error:", err)
			return
		}
		ips.StationMu.Unlock()
		logs.Debug("get stationips:%#v", *ips.StationIPs)
		select {
		case <-timer.C:
		default:
			logs.Info("finished a API request,waitting %s,will request next time ", time.Duration(conf.APIRequestInterval)*time.Second)
			time.Sleep(time.Duration(conf.APIRequestInterval) * time.Second)
		}
		timer.Stop()

	}
}
func (ips *IPs) pingRun(conf *util.Conf, sip string) {
	for {
		timer := time.NewTimer(time.Duration(conf.PingResultPostInterval) * time.Second)
		pinger := ping.NewPing(conf)
		logs.Debug("begion ping reagionIP")
		regionRes, err := pinger.PingAll(ips.RegionIPs, sip, ips.RegionIPs.Region)
		if err != nil {
			logs.Error("PingAll fails with error:", err)
			return
		}
		logs.Debug("reagion ping result:", regionRes)
		logs.Debug("begion ping stationIP")
		stationRes, err := pinger.PingAll(ips.StationIPs, sip, ips.StationIPs.Region)
		if err != nil {
			logs.Error("PingAll fails with error:", err)
			return
		}
		logs.Debug("station ping result:", stationRes)
		logs.Debug("begion tcpping stationIP")
		stationTCPRes := ping.TCPPing(ips.StationIPs, conf)
		logs.Debug("station tcpping result:", stationTCPRes)
		jsonData, err := ips.Res2Json(regionRes, stationRes, stationTCPRes)
		if err != nil {
			logs.Error("ips.Res2Json fail with err:", jsonData)
			return
		}
		logs.Info("json result:",string( jsonData))
		err = Post(jsonData, conf)
		if err != nil {
			logs.Error("Post fail with err:", err)
			return
		}
		logs.Info("json post success!")
		select {
		case <-timer.C:
		default:
			logs.Info("finished a ping task,waitting %s,will exec next time ", time.Duration(conf.PingResultPostInterval)*time.Second)
			time.Sleep(time.Duration(conf.PingResultPostInterval) * time.Second)
		}
		timer.Stop()
	}
}
func (ips *IPs) Res2Json(regionRes map[string]*ping.PingResult, stationRes map[string]*ping.PingResult, stationTCPRes map[string]*util.Statistics) ([]byte, error) {
	data := util.ReqForm{
		Status: 0,
		Error:  "",
	}
	if data.Msg.IP = ips.StationIPs.IP; data.Msg.IP == "" {
		data.Msg.IP = ips.RegionIPs.IP
	}
	if data.Msg.Region = ips.StationIPs.Region; data.Msg.Region == "" {
		data.Msg.Region = ips.RegionIPs.Region
	}
	if data.Msg.Station = ips.StationIPs.Station; data.Msg.Station == "" {
		data.Msg.Station = ips.RegionIPs.Station
	}

	// pips := ipsGetter.GetIPs()
	// ips := make([]string, 0, len(pips))
	// for _, pip := range pips {
	// 	ips = append(ips, pip.IP)
	// }
	res := make([]util.ResMessage, 0)
	for _, ip := range ips.RegionIPs.IPs {
		re := util.ResMessage{
			TargetIP:     ip.IP,
			TargetRegion: ip.Region,
			Type:         "region",
			Result: util.ResultMessage{
				Ping: util.ResPing{
					Avgrtt:  regionRes[ip.IP].AverageRtt,
					Ctime:   regionRes[ip.IP].ProbeTime,
					Loss:    regionRes[ip.IP].LossCount,
					Maxrtt:  regionRes[ip.IP].MaxRtt,
					Minrtt:  regionRes[ip.IP].MinRtt,
					Package: regionRes[ip.IP].PacketCount,
				},
			},
		}
		res = append(res, re)
	}
	for _, ip := range ips.StationIPs.IPs {
		re := util.ResMessage{
			TargetIP:      ip.IP,
			TargetRegion:  ip.Region,
			TargetStation: ip.TargetStation,
			IPStatus:      ip.IPStatus,
			IsPhyIP:       ip.IsPhyIP,
			Type:          "station",
			Result: util.ResultMessage{
				Ping: util.ResPing{
					Avgrtt:  stationRes[ip.IP].AverageRtt,
					Ctime:   stationRes[ip.IP].ProbeTime,
					Loss:    stationRes[ip.IP].LossCount,
					Maxrtt:  stationRes[ip.IP].MaxRtt,
					Minrtt:  stationRes[ip.IP].MinRtt,
					Package: stationRes[ip.IP].PacketCount,
				},
				TCPPing: util.ResTcpping{
					AvgRttMs:    stationTCPRes[ip.IP].AvgRttMs,
					LossPackets: stationTCPRes[ip.IP].LossPackets,
					LossRate:    stationTCPRes[ip.IP].LossRate,
					MaxRttMs:    stationTCPRes[ip.IP].MaxRttMs,
					Mdev:        stationTCPRes[ip.IP].Mdev,
					MinRttMs:    stationTCPRes[ip.IP].MinRttMs,
					RecvPackets: stationTCPRes[ip.IP].RecvPackets,
					SentPackets: stationTCPRes[ip.IP].SentPackets,
				},
			},
		}
		res = append(res, re)
	}
	data.Msg.Res = res
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logs.Error("json marshal failed with error:", err)
		return nil, err
	}

	return jsonData, nil
}
func Post(data []byte, conf *util.Conf) error {
	reader := bytes.NewReader(data)
	request, err := http.NewRequest("POST", conf.PingResultUrl, reader)
	if err != nil {
		logs.Error("http.NewRequest failed with error:", err)
		return err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		logs.Error("client.Do failed with error:", err)
		return err
	}
	var r util.PostRes
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logs.Error("ioutil.ReadAll failed with error:", err)
		return err
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		logs.Error("json.Unmarshal failed with error:", err)
		return err
	}
	if r.Status != 0 {
		err = errors.New(r.Error)
		logs.Error("Form post failed with error:", err)
		return err
	}
	return nil
}
