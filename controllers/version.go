package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type VersionController struct {
	beego.Controller
}

type ResVersion struct {
	Status  int    `json:"code"`
	Message string `json:"message"`
}

func (c *VersionController) GetVersion() {
	c.Ctx.Output.Header("Access-Control-Allow-Origin", "*")
	logs.Info("Get a request from %s", c.Ctx.Request.RemoteAddr)
	defer c.ServeJSON()
	rsp := ResCleaner{
		Status: 200,
	}
	versionStr := beego.AppConfig.String("version")
	if len(versionStr) != 0 {
		rsp.Message = versionStr
	} else {
		rsp.Status = 123
		rsp.Message = "Get version error,Please check the conf file."
	}
	c.Data["json"] = &rsp
	return
}
