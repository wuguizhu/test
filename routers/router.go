package routers

import (
	"github.com/astaxie/beego"
	"testnode-pinger/controllers"
)

func init() {
	// 响应后台post station IPs
	regionURL := beego.AppConfig.String("post_region_url")
	stationURL := beego.AppConfig.String("post_station_url")
	resultsURL := beego.AppConfig.String("get_results_url")
	pingStatusURL := beego.AppConfig.String("ping_status_url")
	cleanURL := beego.AppConfig.String("clean_files_url")
	beego.Router(regionURL, &controllers.RegionIPsController{}, "post:HandleRegionIPs")
	beego.Router(stationURL, &controllers.StationIPsController{}, "post:HandleStationIPs")
	beego.Router(resultsURL, &controllers.ResultsController{}, "get:GetResults")
	beego.Router(pingStatusURL, &controllers.SwitcherController{}, "get:SwitchOFF")
	beego.Router(cleanURL, &controllers.CleanerController{}, "get:CleanUp")
}
