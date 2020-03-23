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

type SpeedtestResultsController struct {
	beego.Controller
}

func (c *SpeedtestResultsController) GetResults() {
	// set api version
	c.Ctx.Output.Header("Api-Version", "v2")
	rand.Seed(time.Now().UnixNano())
	reqID := rand.Intn(0xffff)
	logs.Info("REQUEST:%d,Get a request for speedtest results", reqID)
	defer logs.Info("REQUEST:%d,Finished return the speedtest results", reqID)
	err := errors.New("Failed to get speedtest results, the pinger may not be running")
	rspResult := &util.RspSpeedtestResults{
		Status: 0,
		Msg:    nil,
	}
	defer c.ServeJSON()

	speedtestStatus := process.IPs.SafeReadSpeedtestStatus()
	switchStatus := process.Switcher.SafeReadSwitcherStatus()
	speedtestResStatus := process.Res.SafeReadSpeedtestResStatus()
	speedtestRes := process.Res.SafeReadSpeedtestResults()
	logs.Debug("reqID：%v, speedtestStatus:%v, switchStatus:%v, speedtestResStatus:%v", reqID, speedtestStatus, switchStatus, speedtestResStatus)
	if switchStatus {
		speedtestIPs := new(util.ReqSpeedtest)
		if speedtestStatus {
			speedtestIPs = process.IPs.SafeReadSpeedtestIPs()
		}
		sip := speedtestIPs.IP
		sRegion := speedtestIPs.Region
		sStation := speedtestIPs.Station
		if speedtestResStatus {
			rspResult = process.SpeedtestRes2Rsp(speedtestResStatus, speedtestRes, sip, sRegion, sStation)
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
