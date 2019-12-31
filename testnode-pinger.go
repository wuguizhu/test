package main

import (
	"fmt"
	"testnode-pinger/process"
	_ "testnode-pinger/routers"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

func main() {
	filename := beego.AppConfig.String("ErrorLog")
	if filename != "" {
		if err := logs.SetLogger("file", fmt.Sprintf(`{"filename":"%s"}`, filename)); err != nil {
			panic(err)
		}
	}
	// set log level
	level := beego.AppConfig.String("ErrorLogLevel")
	switch level {
	case "debug":
		logs.SetLevel(beego.LevelDebug)
	case "info":
		logs.SetLevel(beego.LevelInformational)
	case "warn":
		logs.SetLevel(beego.LevelWarning)
	case "error":
		logs.SetLevel(beego.LevelError)
	case "critical":
		logs.SetLevel(beego.LevelCritical)
	default:
		logs.SetLevel(beego.LevelInformational)
	}
	if console, err := beego.AppConfig.Bool("DisableConsole"); err == nil && console == true {
		logs.GetBeeLogger().DelLogger("console")
	}
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	go process.PingProcess()
	logs.Info("Beego start to run")
	beego.Run()
}
