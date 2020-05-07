package controllers

import (
	"encoding/json"
	"testnode-pinger/process"
	"testnode-pinger/util"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type SpeedtestIPsController struct {
	beego.Controller
}

type RspSpeedtest struct {
	Status int    `json:"code"`
	RspMsg string `json:"message"`
}

func (c *SpeedtestIPsController) Get() {
	c.Ctx.WriteString("Error:todo")
}
func (c *SpeedtestIPsController) HandleSpeedtestIPs() {
	defer c.ServeJSON()
	var req util.ReqSpeedtest
	rsp := RspSpeedtest{0, "success"}
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
	logs.Info("get a speedtest ips post")
	logs.Debug("post body length:", len(jsonData))
	logs.Debug("get local station info:ip=%s station=%s", req.IP, req.Station)
	// when get same ips,return with nothing todo
	if !process.IPs.SpeedtestIPsAreChanged(&req) {
		// 虽然无需更新ip，但不要忘记更新speedtestip的update状态
		process.IPs.UpdateSpeedtestStatus(true)
		// 打开整个探测开关(有被手动关闭的情况)
		process.Switcher.UpdateSwitcherStatus(true)
		logs.Info("speedtest ips from request is same with the exsited station ips, Ping and tcpping directly!")
		c.Data["json"] = &rsp
		return
	}
	// 需更新ip，内部实现已经同步更新stationip的update状态，无需重复更新
	process.IPs.UpdateSpeedtestIPs(&req)
	// 打开整个探测开关
	process.Switcher.UpdateSwitcherStatus(true)
	logs.Info("speedtest IPs updated successful!,switcher is opened,Get ready to ping and tcpping!")
	//有了新ip，需要屏蔽旧ip的探测结果
	// process.Res.UpdateSpeedtestResStatus(false)
	// logs.Info("Change speedtestRes status  to false successfully,Because of speedtest ips updated")
	c.Data["json"] = &rsp
}
