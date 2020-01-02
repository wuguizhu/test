package controllers

import (
	"encoding/json"
	"testnode-pinger/process"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type SwitcherController struct {
	beego.Controller
}

// RspSwitcher is a response to fronend with message and error
type RspSwitcher struct {
	Status  int    `json:"code"`
	Message string `json:"message"`
}
type ReqSwitcher = PostReqWithTS

func (c *SwitcherController) SwitchOFF() {
	logs.Info("Get a request from %s,request body:\n%s", c.Ctx.Request.RemoteAddr, c.Ctx.Input.RequestBody)
	defer c.ServeJSON()
	rsp := RspSwitcher{
		Status: 200,
	}
	req := new(ReqSwitcher)
	err := json.Unmarshal(c.Ctx.Input.RequestBody, req)
	if err != nil {
		logs.Error("json.Unmarshal failed with error:", err)
		rsp.Status = 1
		rsp.Message = err.Error()
		c.Data["json"] = &rsp
		return
	}
	timeStamp := time.Now().Unix()
	logs.Info("req_timestamp:%d,now_timestamp:%d,differ:%d", req.TimeStamp, timeStamp, timeStamp-req.TimeStamp)
	if req.TimeStamp < timeStamp-60*3 || req.TimeStamp > timeStamp+60*3 {
		strErr := "Illegal Request! Please check your request data."
		logs.Error(strErr)
		rsp.Status = 2
		rsp.Message = strErr
		c.Data["json"] = &rsp
		return
	}
	strSuccess := "switch off successfuly!stop running ping in seconds"
	strFail := "switch off failed,ping is not running,No need to switch off"
	if process.Switcher.SafeReadSwitcherStatus() {
		process.Switcher.UpdateSwitcherStatus(false)
		process.IPs.UpdateRegionStatus(false)
		process.IPs.UpdateStationStatus(false)
		logs.Info(strSuccess)
		rsp.Message = strSuccess
	} else {
		logs.Info(strFail)
		rsp.Status = 3
		rsp.Message = strFail
	}

	c.Data["json"] = &rsp

}
