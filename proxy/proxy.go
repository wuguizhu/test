package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

// Run start a proxyServer
func Run() {
	proxyPort := beego.AppConfig.String("proxyport")
	l, err := net.Listen("tcp", ":"+proxyPort)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Info("http proxy server Running on http://:" + proxyPort)
	// 监听连接，并起协程处理
	for {
		client, err := l.Accept()
		if err != nil {
			logs.Error(err)
			return
		}
		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}
	defer client.Close()

	var b [1024]byte
	n, err := client.Read(b[:])
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Debug("get a proxy request,len:", n)
	var method, host, address string
	fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		logs.Error(err)
		return
	}

	if hostPortURL.Opaque == "443" { //https访问
		address = hostPortURL.Scheme + ":443"
	} else { //http访问
		if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
			address = hostPortURL.Host + ":80"
		} else {
			address = hostPortURL.Host
		}
	}

	//开始拨号
	server, err := net.Dial("tcp", address)
	if err != nil {
		logs.Error(err)
		return
	}
	if method == "CONNECT" {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		server.Write(b[:n])
	}
	//进行转发
	go io.Copy(server, client)
	io.Copy(client, server)
}
