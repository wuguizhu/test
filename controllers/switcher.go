package controllers

import (
	"testnode-pinger/process"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type SwitcherController struct {
	beego.Controller
}

// FailedResponseWithME is a response to fronend with message and error
type ResSwitcher struct {
	Status  int    `json:"code"`
	Message string `json:"message"`
}

func (c *SwitcherController) SwitchOFF() {
	logs.Info("get a switch off request")
	defer c.ServeJSON()
	resMessage := ResSwitcher{
		Status: 0,
	}
	strSuccess := "switch off successfuly!stop running ping in seconds"
	strFail := "switch off failed,ping is not running,No need to switch off"
	if process.Switcher.SafeReadSwitcherStatus() {
		process.Switcher.UpdateSwitcherStatus(false)
		process.IPs.UpdateRegionStatus(false)
		process.IPs.UpdateStationStatus(false)
		logs.Info(strSuccess)
		resMessage.Message = strSuccess
	} else {
		logs.Info(strFail)
		resMessage.Message = strFail

	}

	c.Data["json"] = &resMessage

}
