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

type ResultsController struct {
	beego.Controller
}

func (c *ResultsController) GetResults() {
	rand.Seed(time.Now().UnixNano())
	reqID := rand.Intn(0xffff)
	logs.Info("REQUEST:%d,Get a request for results", reqID)
	defer logs.Info("REQUEST:%d,Finished return the results", reqID)
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
			rspResult = process.Res2Rsp(regionResStatus, stationResStatus, tcpResStatus, regionRes, stationRes, stationTCPRes, sip, sRegion, sStation)
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
