package controllers

import (
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type CleanerController struct {
	beego.Controller
}

// FailedResponseWithME is a response to fronend with message and error
type ResCleaner struct {
	Status  int    `json:"code"`
	Message string `json:"message"`
}
type ReqCleaner = PostReqWithTS

func (c *CleanerController) CleanUp() {
	c.Ctx.Output.Header("Access-Control-Allow-Origin", "*")
	logs.Info("Get a request from %s,request body:\n%s", c.Ctx.Request.RemoteAddr, c.Ctx.Input.RequestBody)
	defer c.ServeJSON()
	rsp := ResCleaner{
		Status: 200,
	}
	req := new(ReqCleaner)
	err := json.Unmarshal(c.Ctx.Input.RequestBody, req)
	if err != nil {
		logs.Error("json.Unmarshal failed with error:", err)
		rsp.Status = 1
		rsp.Message = err.Error()
		c.Data["json"] = &rsp
		return
	}
	timeStamp := time.Now().Unix()
	logs.Info("req_timestamp:%d,now_timestamp:%d,differ:%d", req.TimeStamp, timeStamp, timeStamp-req.TimeStamp)
	if req.TimeStamp < timeStamp-60*3 || req.TimeStamp > timeStamp+60*3 {
		strErr := "Illegal Request! Please check your request data."
		logs.Error(strErr)
		rsp.Status = 2
		rsp.Message = strErr
		c.Data["json"] = &rsp
		return
	}
	strSuccess := "Clean up all test files successfully!,pinger will exit after 5 seconds "
	strFail := "Warning:Pinger will exit after 5 seconds,BUT clean up failed. Please clean up the test files manually"
	command := `sudo chmod 777 ./shell/clean.sh &&. ./shell/clean.sh &>> ./logs/testnode-pinger.log`
	cmd := exec.Command("/bin/bash", "-c", command)
	signal := make(chan int)
	// make a goroutine to async exec exit,otherwise response will be broken off
	go AppExit(signal, 5*time.Second)
	output, err := cmd.Output()
	if err != nil {
		logs.Error("Execute Shell:%s failed with error:%s\n%s", command, err.Error(), strFail)
		rsp.Status = 3
		rsp.Message = strFail
		c.Data["json"] = &rsp
		signal <- 1
		return
	}
	logs.Info("Execute Shell:%s finished with output:\n%s\n%s", command, string(output), strSuccess)
	rsp.Message = strSuccess
	c.Data["json"] = &rsp
	signal <- 0
	return
}

// AppExit woule exit the current app self after 5,when receive a siginal
func AppExit(signal <-chan int, duration time.Duration) {
	s := <-signal
	time.Sleep(duration)
	os.Exit(s)
}
