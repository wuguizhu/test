package controllers

import (
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

func (c *CleanerController) CleanUp() {
	defer c.ServeJSON()
	resMessage := ResSwitcher{
		Status: 0,
	}
	strSuccess := "Clean up all test files successfully!,pinger will exit after 5 seconds "
	strFail := "Warning:Pinger will exit after 5 seconds,BUT clean up failed. Please clean up the test files manually"
	command := `. ./shell/clean.sh>../logs/testnode-pinger.log`
	cmd := exec.Command("/bin/bash", "-c", command)
	signal := make(chan int)
	go Clean(signal)
	output, err := cmd.Output()
	if err != nil {
		logs.Error("Execute Shell:%s failed with error:%s\n%s", command, err.Error(), strFail)
		resMessage.Status = 1
		resMessage.Message = strFail
		c.Data["json"] = &resMessage
		signal <- 1
		return
	}
	logs.Info("Execute Shell:%s finished with output:\n%s\n%s", command, string(output), strSuccess)
	resMessage.Message = strSuccess
	c.Data["json"] = &resMessage
	signal <- 0
	return
}
func Clean(signal <-chan int) {
	s := <-signal
	time.Sleep(5 * time.Second)
	os.Exit(s)
}
