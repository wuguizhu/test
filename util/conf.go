package util

import (
	"encoding/json"
	"io/ioutil"

	"github.com/astaxie/beego/logs"
)

// Conf .
type Conf struct {
	BaseConf
	OtherConf
}

// conf is a global var
var conf Conf

// BaseConf is the base config file
type BaseConf struct {
	Station string `json:"station"`
	// icmp包大小，56Bytes,不包含IP头
	PingPacketSize int `json:"ping_packet_size"`
	// icmp ping的包个数
	PingPacketCount int `json:"ping_packet_count"`
	//一次ping的ip数
	PingThreadCount int `json:"ping_thread_count"`
	// ping的超时时间
	PingTimeoutMs int `json:"ping_timeout_ms"`
	// ping一轮的时间
	PingTickerMin int `json:"ping_series_ticker_timeout_min"`
	// 同一个ip每个ping间隔时间若小于PingSendPackInterMinMs,则等待PingSendPackWaitMs
	PingSendPackWaitMs     int `json:"ping_send_package_wait_ms"`
	PingSendPackInterMinMs int `json:"ping_send_package_interval_min_ms"`
	//获取ip的间隔
	PingInterval int `json:"ping_interval"`
	// tcpping的参数
	TCPPingPacketSize     int    `json:"tcpping_packet_size"`
	TCPPingPacketCount    int    `json:"tcpping_packet_count"`
	TCPPingTimeoutMs      int    `json:"tcpping_timeout_ms"`
	TCPPingDestPort       uint16 `json:"tcpping_dest_port"`
	TCPPingConnBufferSize int    `json:"tcpping_conn_buffer_size"`
	TCPPingSendGoCount    int    `json:"tcpping_send_go_count"`
	TCPPingRecvGoCount    int    `json:"tcpping_receive_go_count"`
	TCPPingProcGoCount    int    `json:"tcpping_process_go_count"`
	TCPPingInterval       int    `json:"tcpping_interval"`
}

// OtherConf is the other config file that will fill in the future
type OtherConf struct{}

// GetConf gets configs into conf from file
func GetConf(baseFile string) (*Conf, error) {
	buf, err := ioutil.ReadFile(baseFile)
	if err != nil {
		logs.Error("ReadFile failed with error:", err)
		return nil, err
	}
	err = json.Unmarshal(buf, &conf.BaseConf)
	if err != nil {
		logs.Error("Unmarshal failed with error:", err)
		return nil, err
	}
	return &conf, nil
}

// GetConfInfo gets the global var conf,waring:it may not be assigned!
func GetConfInfo() *Conf {
	return &conf
}
