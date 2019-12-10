package controllers

import (
	"encoding/json"
	"testnode-pinger/process"
	"testnode-pinger/util"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	// "testnode-pinger/models"
)

type RegionIPsController struct {
	beego.Controller
}

// RspRegion is a request to testnode-collecter
type RspRegion struct {
	Status int    `json:"code"`
	RspMsg string `json:"message"`
}

func (c *RegionIPsController) Get() {
	c.Ctx.WriteString("Error:todo")
}
func (c *RegionIPsController) HandleRegionIPs() {
	defer c.ServeJSON()
	var req util.ReqRegion
	rsp := RspRegion{0, "success"}
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if err != nil {
		logs.Error("json.Unmarshal failed with error:", err)
		rsp.Status = 1
		rsp.RspMsg = err.Error()
		return
	}
	jsonData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		logs.Error("json marshal failed with error:", err)
	}
	logs.Info("get a region ips post")
	logs.Debug("post body:", string(jsonData))
	logs.Debug("get local station info:ip=%s station=%s", req.IP, req.Station)
	process.IPs.UpdateRegionIPs(&req)
	process.Switcher.UpdateSwitcherStatus(true)
	logs.Info("region IPs updated successful!,Begin to ping!")
	c.Data["json"] = &rsp
}
