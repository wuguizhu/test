package util

import (
	"fmt"
	"net"
	"strings"

	"github.com/astaxie/beego/logs"
)

// GetAllLocalIP 获取本地网卡所有有效的IP地址，包括IPv4\IPv6
func GetAllLocalIP() ([]string, error) {
	ips := make([]string, 0, 1)

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("net.Interface error: %v", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			// 没有up或者是loopback,跳过
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && IsPublicIP(ipnet.IP) {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips, nil
}

//GetIPv4LocalIP 获取本地网卡所有有效的IPv4地址
func GetIPv4LocalIP() ([]string, error) {
	allLocalIP, err := GetAllLocalIP()
	if err != nil {
		return nil, err
	}
	logs.Debug("allLocalIPs:", allLocalIP)
	ipv4s := make([]string, 0, 1)
	for _, ip := range allLocalIP {
		if strings.Contains(ip, ":") {
			// 只取ipv4
			continue
		}
		ipv4s = append(ipv4s, ip)
	}
	return ipv4s, nil
}

//IsPublicIP 判断是否公网IP,支持IPv4,IPv6
func IsPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return false
	}
	// IPv4私有地址空间
	// A类：10.0.0.0到10.255.255.255
	// B类：172.16.0.0到172.31.255.255
	// C类：192.168.0.0到192.168.255.255
	if ip4 := ip.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	// IPv6私有地址空间：以前缀FEC0::/10开头
	if ip6 := ip.To16(); ip6 != nil {
		if ip6[0] == 15 && ip6[1] == 14 && ip6[2] <= 12 {
			return false
		}
		return true
	}
	return false
}

//
// func SetNofile(max uint64) error {

// }
// 随机挑选一个本地网卡有效的IPv4地址
func GetLocalIpAddr() (ip string, err error) {
	ip = "127.0.0.1"
	// ip = "111.201.146.13"
	ifaces, err := net.Interfaces()
	if err != nil {
		return ip, fmt.Errorf("net.Interfaces error:", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || strings.HasPrefix(iface.Name, "lo") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && IsPublicIP(ipnet.IP) {
				return ipnet.IP.String(), nil
			}
		}
	}
	return ip, nil
}
