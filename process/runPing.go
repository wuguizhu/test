package process

import (
	"sync"
	"testnode-pinger/ping"
	"testnode-pinger/util"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

var (
	Switcher = NewSwitcherUpdater()
	IPs      = NewIPsUpdater()
	Res      = NewResults()
)

func PingProcess() {
	configFile := "./conf/testnodeprober.json"
	conf, err := util.GetConf(configFile)
	if err != nil {
		logs.Error("GetConf fails with error:", err)
		return
	}
	logs.Debug("conf: %#v", conf)
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

	go IPs.pingRun(conf, sip)
}

type SwitcherUpdater struct {
	IsOk   bool
	IsOkMu *sync.RWMutex
}

func NewSwitcherUpdater() *SwitcherUpdater {
	return &SwitcherUpdater{
		IsOk:   false,
		IsOkMu: new(sync.RWMutex),
	}
}
func (switcher *SwitcherUpdater) UpdateSwitcherStatus(status bool) {
	switcher.IsOkMu.Lock()
	switcher.IsOk = status
	switcher.IsOkMu.Unlock()
}
func (switcher *SwitcherUpdater) SafeReadSwitcherStatus() (status bool) {
	switcher.IsOkMu.RLock()
	status = switcher.IsOk
	switcher.IsOkMu.RUnlock()
	return
}

type IPsUpdater struct {
	RegionIPs      *util.ReqRegion
	StationIPs     *util.ReqStation
	RegionUpdated  bool
	StationUpdated bool
	RegionMu       *sync.RWMutex
	StationMu      *sync.RWMutex
	// InitedMu *sync.RWMutex
	RegionUpdatedMu  *sync.RWMutex
	StationUpdatedMu *sync.RWMutex
}

func NewIPsUpdater() *IPsUpdater {
	return &IPsUpdater{
		RegionIPs:        new(util.ReqRegion),
		StationIPs:       new(util.ReqStation),
		RegionUpdated:    false,
		StationUpdated:   false,
		RegionMu:         new(sync.RWMutex),
		StationMu:        new(sync.RWMutex),
		RegionUpdatedMu:  new(sync.RWMutex),
		StationUpdatedMu: new(sync.RWMutex),
	}

}
func (ips *IPsUpdater) UpdateRegionIPs(req *util.ReqRegion) {
	ips.RegionMu.Lock()
	ips.RegionIPs = req
	ips.RegionMu.Unlock()
	ips.UpdateRegionStatus(true)
}
func (ips *IPsUpdater) UpdateRegionStatus(status bool) {
	ips.RegionUpdatedMu.Lock()
	ips.RegionUpdated = status
	ips.RegionUpdatedMu.Unlock()
}
func (ips *IPsUpdater) UpdateStationIPs(req *util.ReqStation) {
	ips.StationMu.Lock()
	ips.StationIPs = req
	ips.StationMu.Unlock()
	ips.UpdateStationStatus(true)
}
func (ips *IPsUpdater) UpdateStationStatus(status bool) {
	ips.StationUpdatedMu.Lock()
	ips.StationUpdated = status
	ips.StationUpdatedMu.Unlock()
}
func (ips *IPsUpdater) SafeReadRegionIPs() (regionIPs *util.ReqRegion) {
	ips.RegionMu.RLock()
	regionIPs = ips.RegionIPs
	ips.RegionMu.RUnlock()
	return
}
func (ips *IPsUpdater) SafeReadStationIPs() (stationIPs *util.ReqStation) {
	ips.StationMu.RLock()
	stationIPs = ips.StationIPs
	ips.StationMu.RUnlock()
	return
}
func (ips *IPsUpdater) SafeReadRegionStatus() (regionUpdated bool) {
	ips.RegionUpdatedMu.RLock()
	regionUpdated = ips.RegionUpdated
	ips.RegionUpdatedMu.RUnlock()
	return
}
func (ips *IPsUpdater) SafeReadStationStatus() (stationUpdated bool) {
	ips.StationUpdatedMu.RLock()
	stationUpdated = ips.StationUpdated
	ips.StationUpdatedMu.RUnlock()
	return
}

// pingRun run a ping task (ping,tcping then update Results)
func (ips *IPsUpdater) pingRun(conf *util.Conf, sip string) {

	for {
		// 每隔一定时间检查一次开关状态
		checkStatus, err := beego.AppConfig.Int("check_status_interval_s")
		if err != nil {
			logs.Error("checkStatus err:", err)
			return
		}
		if !Switcher.SafeReadSwitcherStatus() {
			logs.Debug("ping is not running,beacuse ping switcher status is false. Check status another time after:", time.Duration(checkStatus)*time.Second)
			time.Sleep(time.Duration(checkStatus) * time.Second)
			continue
		}
		interval, err := beego.AppConfig.Int("ping_interval_s")
		if err != nil {
			logs.Error("beego.AppConfig.Int failed with error:", err)
			continue
		}
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		pinger := ping.NewPing(conf)
		// ping regionIPs
		if ips.SafeReadRegionStatus() {
			logs.Info("begin ping reagionIP")
			regionips := ips.SafeReadRegionIPs()
			regionRes, err := pinger.TestNodePing(regionips, sip, regionips.Region)
			if err != nil {
				logs.Error("TestNodePing fails with error:", err)
				continue
			}
			Res.UpdateRegionResults(regionRes)
			Res.UpdateRegionResStatus(true)
			logs.Info("finish ping reagionIP")

		}

		// ping stationIPs
		if ips.SafeReadStationStatus() {
			logs.Info("begin ping stationIP")
			stationips := ips.SafeReadStationIPs()
			stationRes, err := pinger.TestNodePing(stationips, sip, stationips.Region)
			if err != nil {
				logs.Error("TestNodePing fails with error:", err)
				continue
			}
			Res.UpdateStationResStatus(true)
			Res.UpdateStationResults(stationRes)
			logs.Info("finish ping stationIP")

			// tcpping stationIPs
			logs.Info("begin tcpping stationIP")
			stationTCPRes := ping.TCPPing(stationips, conf)
			Res.UpdateTCPResults(stationTCPRes)
			Res.UpdateTCPResStatus(true)
			logs.Info("finish tcpping stationIP")

		}

		select {
		case <-timer.C:
		default:
			logs.Info("finished a ping task,waitting %s,will exec next time ", time.Duration(interval)*time.Second)
			time.Sleep(time.Duration(interval) * time.Second)
		}
		timer.Stop()

	}

}

// Res2Rsp convert all res to RspResults
func (ips *IPsUpdater) Res2Rsp(regionRes map[string]*ping.PingResult, stationRes map[string]*ping.PingResult, stationTCPRes map[string]*util.Statistics) *util.RspResults {
	data := util.RspResults{
		Status: 0,
		Msg:    new(util.Message),
	}
	regionIPs := new(util.ReqRegion)
	stationIPs := new(util.ReqStation)
	if ips.SafeReadRegionStatus() {
		regionIPs = ips.SafeReadRegionIPs()

	}
	if ips.SafeReadStationStatus() {
		stationIPs = ips.SafeReadStationIPs()

	}
	if data.Msg.IP = stationIPs.IP; data.Msg.IP == "" {
		data.Msg.IP = regionIPs.IP
	}
	if data.Msg.Region = stationIPs.Region; data.Msg.Region == "" {
		data.Msg.Region = regionIPs.Region
	}
	if data.Msg.Station = stationIPs.Station; data.Msg.Station == "" {
		data.Msg.Station = regionIPs.Station
	}
	res := make([]*util.ResMessage, 0)

	if regionRes != nil {
		for _, ip := range regionIPs.IPs {
			re := util.ResMessage{
				TargetIP:     ip.IP,
				TargetRegion: ip.Region,
				Type:         "region",
				Result:       new(util.ResultMessage),
			}

			re.Result.Ping = util.ResPing{
				Avgrtt:     regionRes[ip.IP].AverageRtt,
				Ctime:      regionRes[ip.IP].ProbeTime,
				Loss:       regionRes[ip.IP].LossCount,
				Maxrtt:     regionRes[ip.IP].MaxRtt,
				Minrtt:     regionRes[ip.IP].MinRtt,
				Package:    regionRes[ip.IP].PacketCount,
				PingAtTime: regionRes[ip.IP].PingAtTime,
			}
			res = append(res, &re)
		}
	}
	if stationRes != nil && stationTCPRes != nil {
		for _, ip := range stationIPs.IPs {
			re := util.ResMessage{
				TargetIP:      ip.IP,
				TargetRegion:  ip.Region,
				TargetStation: ip.TargetStation,
				IPStatus:      ip.IPStatus,
				IsPhyIP:       ip.IsPhyIP,
				Type:          "station",
				Result:        new(util.ResultMessage),
			}
			if _, ok := stationRes[ip.IP]; ok {
				re.Result.Ping = util.ResPing{
					Avgrtt:     stationRes[ip.IP].AverageRtt,
					Ctime:      stationRes[ip.IP].ProbeTime,
					Loss:       stationRes[ip.IP].LossCount,
					Maxrtt:     stationRes[ip.IP].MaxRtt,
					Minrtt:     stationRes[ip.IP].MinRtt,
					Package:    stationRes[ip.IP].PacketCount,
					PingAtTime: stationRes[ip.IP].PingAtTime,
				}
			}

			if _, ok := stationTCPRes[ip.IP]; ok {
				re.Result.TCPPing = util.ResTcpping{
					AvgRttMs:    stationTCPRes[ip.IP].AvgRttMs,
					LossPackets: stationTCPRes[ip.IP].LossPackets,
					LossRate:    stationTCPRes[ip.IP].LossRate,
					MaxRttMs:    stationTCPRes[ip.IP].MaxRttMs,
					Mdev:        stationTCPRes[ip.IP].Mdev,
					MinRttMs:    stationTCPRes[ip.IP].MinRttMs,
					RecvPackets: stationTCPRes[ip.IP].RecvPackets,
					SentPackets: stationTCPRes[ip.IP].SentPackets,
					PingAtTime:  stationTCPRes[ip.IP].PingAtTime,
				}
			}

			res = append(res, &re)
		}
		// ips.UpdateStationStatus(false)
	}
	data.Msg.Res = res
	return &data
}

type Results struct {
	RegionRes              map[string]*ping.PingResult
	StationRes             map[string]*ping.PingResult
	StationTCPRes          map[string]*util.Statistics
	RegionResUpdated       bool
	StationResUpdated      bool
	StationTCPResUpdated   bool
	regionResMu            *sync.RWMutex
	stationResMu           *sync.RWMutex
	stationTCPResMu        *sync.RWMutex
	RegionResUpdatedMu     *sync.RWMutex
	StationResUpdatedMu    *sync.RWMutex
	StationTCPResUpdatedMu *sync.RWMutex
}

func NewResults() *Results {
	return &Results{
		make(map[string]*ping.PingResult),
		make(map[string]*ping.PingResult),
		make(map[string]*util.Statistics),
		false,
		false,
		false,
		new(sync.RWMutex),
		new(sync.RWMutex),
		new(sync.RWMutex),
		new(sync.RWMutex),
		new(sync.RWMutex),
		new(sync.RWMutex),
	}
}

// updateResults update PingResults from the ping results
func (res *Results) UpdateTCPResults(stationTCPRes map[string]*util.Statistics) {
	res.stationTCPResMu.Lock()
	res.StationTCPRes = stationTCPRes
	res.stationTCPResMu.Unlock()
	res.UpdateTCPResStatus(true)
}
func (res *Results) UpdateTCPResStatus(status bool) {
	res.StationTCPResUpdatedMu.Lock()
	res.StationTCPResUpdated = status
	res.StationTCPResUpdatedMu.Unlock()
}
func (res *Results) UpdateRegionResults(regionRes map[string]*ping.PingResult) {
	res.regionResMu.Lock()
	res.RegionRes = regionRes
	res.regionResMu.Unlock()
	res.UpdateRegionResStatus(true)
}
func (res *Results) UpdateRegionResStatus(status bool) {
	res.RegionResUpdatedMu.Lock()
	res.RegionResUpdated = status
	res.RegionResUpdatedMu.Unlock()
}
func (res *Results) UpdateStationResults(stationRes map[string]*ping.PingResult) {
	res.stationResMu.Lock()
	res.StationRes = stationRes
	res.stationResMu.Unlock()
	res.UpdateStationResStatus(true)
}
func (res *Results) UpdateStationResStatus(status bool) {
	res.StationResUpdatedMu.Lock()
	res.StationResUpdated = status
	res.StationResUpdatedMu.Unlock()

}

// updateResults update PingResults from the ping results
func (res *Results) SafeReadTCPResults() (stationTCPRes map[string]*util.Statistics) {
	res.stationTCPResMu.RLock()
	stationTCPRes = res.StationTCPRes
	res.stationTCPResMu.RUnlock()
	return
}
func (res *Results) SafeReadTCPResStatus() (status bool) {
	res.StationTCPResUpdatedMu.RLock()
	status = res.StationTCPResUpdated
	res.StationTCPResUpdatedMu.RUnlock()
	return
}
func (res *Results) SafeReadRegionResults() (regionRes map[string]*ping.PingResult) {
	res.regionResMu.RLock()
	regionRes = res.RegionRes
	res.regionResMu.RUnlock()
	return
}
func (res *Results) SafeReadRegionResStatus() (status bool) {
	res.RegionResUpdatedMu.RLock()
	status = res.RegionResUpdated
	res.RegionResUpdatedMu.RUnlock()
	return
}
func (res *Results) SafeReadStationResults() (stationRes map[string]*ping.PingResult) {
	res.stationResMu.RLock()
	stationRes = res.StationRes
	res.stationResMu.RUnlock()
	return
}
func (res *Results) SafeReadStationResStatus() (status bool) {
	res.StationResUpdatedMu.RLock()
	status = res.StationResUpdated
	res.StationResUpdatedMu.RUnlock()
	return
}
