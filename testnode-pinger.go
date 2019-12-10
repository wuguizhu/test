package main

import (
	"fmt"
	"testnode-pinger/process"
	_ "testnode-pinger/routers"

	"github.com/astaxie/beego"
)

func main() {
	filename := beego.AppConfig.String("ErrorLog")
	if filename != "" {
		if err := beego.SetLogger("file", fmt.Sprintf(`{"filename":"%s"}`, filename)); err != nil {
			panic(err)
		}
	}
	// set log level
	level := beego.AppConfig.String("ErrorLogLevel")
	switch level {
	case "debug":
		beego.SetLevel(beego.LevelDebug)
	case "info":
		beego.SetLevel(beego.LevelInformational)
	case "warn":
		beego.SetLevel(beego.LevelWarning)
	case "error":
		beego.SetLevel(beego.LevelError)
	case "critical":
		beego.SetLevel(beego.LevelCritical)
	default:
		beego.SetLevel(beego.LevelInformational)
	}
	if console, err := beego.AppConfig.Bool("DisableConsole"); err == nil && console == true {
		beego.BeeLogger.DelLogger("console")
	}
	go process.Process()
	beego.Info("Beego start to run")
	beego.Run()
}
