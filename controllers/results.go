package controllers

import (
	"errors"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"testnode-pinger/process"
	"testnode-pinger/util"
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
	logs.Debug("a get request for results")
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
	logs.Debug("regionStatus:%v, stationStatus:%v, switchStatus:%v, regionResStatus:%v, stationResStatus:%v, tcpResStatus:%v", regionStatus, stationStatus, switchStatus, regionResStatus, stationResStatus, tcpResStatus)
	if switchStatus {
		if regionResStatus || stationResStatus || tcpResStatus {
			rspResult = process.IPs.Res2Rsp(regionRes, stationRes, stationTCPRes)
			logs.Debug(*rspResult)
			c.Data["json"] = rspResult
			logs.Info("Return result successful")
		} else {
			rspResult.Status = 2
			rspResult.Error = "Results have not been updated, please try again later"
			logs.Warn("User requested unupdated data, request was rejected")
			c.Data["json"] = rspResult
		}

	} else {
		logs.Error(err.Error())
		rspResult.Status = 3
		rspResult.Error = err.Error()
		c.Data["json"] = rspResult
	}

}
