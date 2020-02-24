package controllers

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type HealthController struct {
	beego.Controller
}

type ResHealth struct {
	Status    int    `json:"code"`
	StampData int64  `json:"data,omitempty"`
	Message   string `json:"message"`
}

func (c *HealthController) HealthCheck() {
	c.Ctx.Output.Header("Access-Control-Allow-Origin", "*")
	logs.Info("Get a request from %s", c.Ctx.Request.RemoteAddr)
	defer c.ServeJSON()
	rsp := ResHealth{
		Status: 200,
	}
	localCN, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logs.Error("time.LoadLocation failed with err:", err)
		rsp.Status = 421
		rsp.Message = "获取时间戳错误"
	}
	rsp.StampData = time.Now().In(localCN).UnixNano()
	rsp.Message = "success"
	c.Data["json"] = &rsp
	return
}
