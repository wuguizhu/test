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
	Ping    *ResPing    `json:"ping"`
	TCPPing *ResTcpping `json:"tcpping"`
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

type ResultsController struct {
	beego.Controller
}

func (c *ResultsController) GetResults() {
	rand.Seed(time.Now().UnixNano())
	reqID := rand.Intn(0xffff)
	logs.Info("Get a request for results,reqID：", reqID)
	defer logs.Info("Return result successful,reqID：", reqID)
	err := errors.New("Failed to get results, the pinger may not be running")
	rspResult := &util.RspResults{
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
		if regionResStatus || stationResStatus || tcpResStatus {
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
			rspResult = process.Res2Rsp(regionRes, stationRes, stationTCPRes, sip, sRegion, sStation)
			c.Data["json"] = rspResult
			logs.Info("finish prepare successful result,reqID：", reqID)
		} else {
			rspResult.Status = 2
			rspResult.Error = "Results have not been updated, please try again later"
			logs.Warn("User requested unupdated data, request was rejected,reqID：", reqID)
			c.Data["json"] = rspResult
			logs.Info("finish prepare failed result,reqID：", reqID)
		}

	} else {
		logs.Error(err.Error())
		rspResult.Status = 3
		rspResult.Error = err.Error()
		c.Data["json"] = rspResult
	}

}
