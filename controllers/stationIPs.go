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
		c.Data["json"] = &rsp
		return
	}
	jsonData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		logs.Error("json marshal failed with error: ", err)
	}
	logs.Info("get a station ips post")
	logs.Debug("post body length:", len(jsonData))
	logs.Debug("get local station info:ip=%s station=%s", req.IP, req.Station)
	// when get same ips,return with nothing todo
	if !process.IPs.StationIPsAreChanged(&req) {
		// 虽然无需更新ip，但不要忘记更新stationip的update状态
		process.IPs.UpdateStationStatus(true)
		// 打开整个探测开关(有被手动关闭的情况)
		process.Switcher.UpdateSwitcherStatus(true)
		logs.Info("Station ips from request is same with the exsited station ips, Ping and tcpping directly!")
		c.Data["json"] = &rsp
		return
	}
	// 需更新ip，内部实现已经同步更新stationip的update状态，无需重复更新
	process.IPs.UpdateStationIPs(&req)
	// 打开整个探测开关
	process.Switcher.UpdateSwitcherStatus(true)
	logs.Info("station IPs updated successful!,,switcher is opened,Get ready to ping and tcpping!")
	//有了新ip，需要屏蔽旧ip的探测结果
	process.Res.UpdateStationResStatus(false)
	process.Res.UpdateTCPResStatus(false)
	logs.Info("Change all stationRes、TCPRes status  to false successfully,Because of station ips updating")
	c.Data["json"] = &rsp
}
