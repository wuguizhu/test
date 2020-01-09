package controllers

import (
	"errors"
	"math/rand"
	"testnode-pinger/process"
	"testnode-pinger/util"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

// RspResults is a  response from testnode-pinger: /test/result
type RspResults struct {
	Status int      `json:"status"`
	Msg    *Message `json:"message,omitempty"`
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
	Result        *ResultMessage `json:"result,omitempty"`
	TargetIP      string         `json:"targetIP"`
	TargetRegion  string         `json:"targetRegion"`
	TargetStation string         `json:"targetStation"`
	Type          string         `json:"type"`
}
type ResultMessage struct {
	Ping    *ResPing    `json:"ping,omitempty"`
	TCPPing *ResTcpping `json:"tcpping,omitempty"`
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

type ResultsController struct {
	beego.Controller
}

func (c *ResultsController) GetResults() {
	rand.Seed(time.Now().UnixNano())
	reqID := rand.Intn(0xffff)
	logs.Info("REQUEST:%d,Get a request for results", reqID)
	defer logs.Info("REQUEST:%d,Finished return the results", reqID)
	err := errors.New("Failed to get results, the pinger may not be running")
	rspResult := &RspResults{
		Status: 0,
		Msg:    nil,
	}
	defer c.ServeJSON()

	regionStatus := process.IPs.SafeReadRegionStatus()
	stationStatus := process.IPs.SafeReadStationStatus()
	switchStatus := process.Switcher.SafeReadSwitcherStatus()
	regionResStatus := process.Res.SafeReadRegionResStatus()
	stationResStatus := process.Res.SafeReadStationResStatus()
	tcpResStatus := process.Res.SafeReadTCPResStatus()
	regionRes := process.Res.SafeReadRegionResults()
	stationRes := process.Res.SafeReadStationResults()
	stationTCPRes := process.Res.SafeReadTCPResults()
	logs.Debug("reqID：%v,regionStatus:%v, stationStatus:%v, switchStatus:%v, regionResStatus:%v, stationResStatus:%v, tcpResStatus:%v", reqID, regionStatus, stationStatus, switchStatus, regionResStatus, stationResStatus, tcpResStatus)
	if switchStatus {
		regionIPs := new(util.ReqRegion)
		stationIPs := new(util.ReqStation)
		if regionStatus {
			regionIPs = process.IPs.SafeReadRegionIPs()
		}
		if stationStatus {
			stationIPs = process.IPs.SafeReadStationIPs()
		}
		var sip, sRegion, sStation string
		if sip = stationIPs.IP; sip == "" {
			sip = regionIPs.IP
		}
		if sRegion = stationIPs.Region; sRegion == "" {
			sRegion = regionIPs.Region
		}
		if sStation = stationIPs.Station; sStation == "" {
			sStation = regionIPs.Station
		}
		if regionResStatus || (stationResStatus && tcpResStatus) {
			rspResult = process.Res2Rsp(regionRes, stationRes, stationTCPRes, sip, sRegion, sStation)
			c.Data["json"] = rspResult
			logs.Info("REQUEST:%d,finish prepare successful result", reqID)
		} else {
			rspResult.Status = 2
			rspResult.Error = "Results have not been updated, please try again later"
			logs.Warn("REQUEST:%d,User requested unupdated data, request was rejected", reqID)
			c.Data["json"] = rspResult
			logs.Info("REQUEST:%d,finish prepare failed result", reqID)
		}

	} else {
		logs.Error(err.Error())
		rspResult.Status = 3
		rspResult.Error = err.Error()
		c.Data["json"] = rspResult
	}

}
