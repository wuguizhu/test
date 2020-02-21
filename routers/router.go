package routers

import (
	"testnode-pinger/controllers"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/plugins/cors"
)

func init() {
	//InsertFilter是提供一个过滤函数
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		//允许访问所有源
		AllowAllOrigins: true,
		//可选参数"GET", "POST", "PUT", "DELETE", "OPTIONS" (*为所有)
		//其中Options跨域复杂请求预检
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//指的是允许的Header的种类
		AllowHeaders: []string{"*"},
		//公开的HTTP标头列表
		ExposeHeaders: []string{"*"},
		//如果设置，则允许共享身份验证凭据，例如cookie
		AllowCredentials: true,
	}))
	// 响应后台post station IPs
	regionURL := beego.AppConfig.String("post_region_url")
	stationURL := beego.AppConfig.String("post_station_url")
	resultsURL := beego.AppConfig.String("get_results_url")
	pingStatusURL := beego.AppConfig.String("ping_status_url")
	cleanURL := beego.AppConfig.String("clean_files_url")
	beego.Router(regionURL, &controllers.RegionIPsController{}, "post:HandleRegionIPs")
	beego.Router(stationURL, &controllers.StationIPsController{}, "post:HandleStationIPs")
	beego.Router(resultsURL, &controllers.ResultsController{}, "get:GetResults")
	beego.Router(pingStatusURL, &controllers.SwitcherController{}, "post:SwitchOFF")
	beego.Router(cleanURL, &controllers.CleanerController{}, "post:CleanUp")
	beego.Router("/test/ips/version", &controllers.VersionController{}, "get:GetVersion")
	beego.Router("test/ips/heathcheck", &controllers.HealthController{}, "get:HealthCheck")
}
