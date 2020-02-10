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
	logs.Info("get a station ips post")
	logs.Debug("post body:", string(jsonData))
	logs.Debug("get local station info:ip=%s station=%s", req.IP, req.Station)
	// when get same ips,return with nothing todo
	if !process.IPs.StationIPsAreChanged(&req) {
		logs.Info("Station ips from request is same with the exsited station ips, Ping and tcpping directly!")
		c.Data["json"] = &rsp
		return
	}
	process.IPs.UpdateStationIPs(&req)
	process.Switcher.UpdateSwitcherStatus(true)
	logs.Info("station IPs updated successful!,Get ready to ping and tcpping!")
	//need to change all res status to false
	process.Res.UpdateStationResStatus(false)
	process.Res.UpdateTCPResStatus(false)
	logs.Info("Change all stationRes、TCPRes status  to false successfully,Because of station ips updating")
	c.Data["json"] = &rsp
}
