package controllers

import (
	"encoding/json"
	"testnode-pinger/process"
	"testnode-pinger/util"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type StationIPsController struct {
	beego.Controller
}

type RspStation struct {
	Status int    `json:"code"`
	RspMsg string `json:"message"`
}

func (c *StationIPsController) Get() {
	c.Ctx.WriteString("Error:todo")
}
func (c *StationIPsController) HandleStationIPs() {
	defer c.ServeJSON()
	var req util.ReqStation
	rsp := RspStation{0, "success"}
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if err != nil {
		logs.Error("json.Unmarshal failed with error:", err)
		rsp.Status = 1
		rsp.RspMsg = err.Error()
		return
	}
	jsonData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		logs.Error("json marshal failed with error: ", err)
	}
	logs.Debug("get a station ips post:", string(jsonData))
	logs.Debug("get test station info:ip=%s station=%s", req.IP, req.Station)
	process.IPs.UpdateStationIPs(&req)
	// if process.IPs.SafeReadStationStatus() && process.IPs.SafeReadRegionStatus() {
	process.Switcher.UpdateSwitcherStatus(true)
	logs.Debug("IPs updated successful!,Begin to ping!")
	// }
	c.Data["json"] = &rsp
}
