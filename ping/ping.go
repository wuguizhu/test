package ping

import (
	"math"
	"math/rand"
	"testnode-pinger/util"
	"time"

	"github.com/astaxie/beego/logs"
)

type Ping struct {
	Station          string
	IPRegionURL      string //获取Reagon IP的url
	IPStationURL     string //获取Station IP的url
	ResultURL        string //上传ping结果的url
	PacketSize       int    //ping包的大小
	PacketCount      int
	ThreadCount      int
	TimeoutMs        int
	TickerMin        int
	SendPackWait     int
	SendPackInterMin int
	Interval         int
}

//PingStat is the status of ping
type PingStat struct {
	Times    int           //返回ICMP包结果的次数
	Duration time.Duration //返回所有ICMP包的rtt之和
	MaxRtt   time.Duration //最大rtt
	MinRtt   time.Duration //最小rtt
}
type PingResult struct {
	PacketCount int     `json:"package"`
	AverageRtt  float64 `json:"avgrtt"`
	MinRtt      float64 `json:"minrtt"`
	MaxRtt      float64 `json:"maxrtt"`
	LossCount   int     `json:"loss"`
	ProbeTime   float64 `json:"ctime"`
	PingAtTime  string
}

func NewPing(conf *util.Conf) *Ping {
	p := Ping{
		Station:          conf.Station,
		PacketSize:       conf.PingPacketSize,
		PacketCount:      conf.PingPacketCount,
		ThreadCount:      conf.PingThreadCount,
		TimeoutMs:        conf.PingTimeoutMs,
		TickerMin:        conf.PingTickerMin,
		SendPackWait:     conf.PingSendPackWaitMs,
		SendPackInterMin: conf.PingSendPackInterMinMs,
	}
	return &p
}
func (p *Ping) TestNodePing(ipsGetter util.IPsGetter, sip, sregion string) (result map[*util.PingIP]*PingResult, err error) {
	result = make(map[*util.PingIP]*PingResult)
	localCN, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logs.Error("time.LoadLocation failed with err:", err)
		return nil, err
	}
	pingTime := time.Now().In(localCN).Format("2006-01-02 15:04:05")
	pips := ipsGetter.GetIPs()
	numIP := len(pips)
	numGroup := int(math.Ceil(float64(numIP) / float64(p.ThreadCount)))
	// 分组ping
	ipMap := make(map[int][]*util.PingIP, numGroup)
	for index, pip := range pips {
		key := index % numGroup
		ipMap[key] = append(ipMap[key], pip)
	}
	for groupIndex, groupIPs := range ipMap {
		logs.Debug("ips total count:%d,groups:%d,current group:%d", numIP, numGroup, groupIndex)
		ips := make([]string, 0, len(pips))
		//保存最新的ping stat
		stat := make(map[string]*PingStat)

		for _, pip := range groupIPs {
			result[pip] = nil
			ips = append(ips, pip.IP)
		}
		if len(ips) > 0 {
			res, send, err := ping(p.PacketCount, p.PacketSize, p.TimeoutMs, p.SendPackInterMin, p.SendPackWait, ips, sip)
			if err != nil {
				logs.Error("pips ping fail gorup：%d,with error:%v ", groupIndex, err)
				return nil, err
			}
			// 更新有ping结果的IP
			for k, r := range res {
				delete(stat, k)
				stat[k] = r
			}
			for ip, r := range stat {
				if r == nil {
					logs.Debug("dst ip:%s,rtt:%d,loss rate 100%%", ip, util.MaxRtt)
				} else {
					logs.Debug("dst ip:%s,MaxRtt:%v,MinRtt:%v,Durition:%s", ip, r.MaxRtt, r.MinRtt, r.Duration.String())
				}
			}
			// 将stat计算出result
			for _, pip := range pips {
				pingResult := new(PingResult)
				pingResult.PacketCount = p.PacketCount
				pingResult.PingAtTime = pingTime
				r, ok := stat[pip.IP]
				if !ok {
					pingResult.LossCount = send
				} else {
					pingResult.LossCount = send - r.Times
					pingResult.AverageRtt = float64(r.Duration) / float64(time.Millisecond) / float64(r.Times)
					pingResult.MinRtt = float64(r.MinRtt) / float64(time.Millisecond)
					pingResult.MaxRtt = float64(r.MaxRtt) / float64(time.Millisecond)
					pingResult.ProbeTime = float64(r.Duration) / float64(time.Millisecond)
				}
				result[pip] = pingResult
				// 				logs.Debug("packages:%d,avgRtt:%.2f,minRtt:%.2f,maxRtt:%.2f,loss:%d,probeTime:%s", result[ip].PacketCount, result[ip].AverageRtt, result[ip].MinRtt, result[ip].MaxRtt, result[ip].LossCount, result[ip].ProbeTime)
			}

		}
	}

	return result, nil
}

// ping perform a ping operation
func ping(times, size, timeout, sendPackageInterMin, sendPackageWait int, ips []string, sip string) (result map[string]*PingStat, send int, err error) {
	result = make(map[string]*PingStat)
	//给每个ip生成随机id放入icmp报文，保证同一个ip的id相同
	ip2id := make(map[string]int)
	rand.Seed(time.Now().UnixNano())
	for _, ip := range ips {
		ip2id[ip] = rand.Intn(0xffff)
	}
	// 开始ping指定次数
	// startTime := time.Now()
	for send = 0; send < times; send++ {
		logs.Debug("ping round:%d start", send)
		pinger := util.NewPinger(send+1, "")
		for _, ip := range ips {
			err := pinger.AddIP(ip, ip2id[ip])
			if err != nil {
				logs.Error("p.AddIP(ip) for ip %s fails with error %s\n", ip, err)
				continue
			}
		}

		pinger.Size = size
		pinger.MaxRTT = time.Duration(timeout) * time.Millisecond
		timer := time.NewTimer(time.Duration(sendPackageInterMin) * time.Millisecond)
		now := time.Now()
		// 启动本次ping
		m, _, err := pinger.Run()
		if err != nil {
			logs.Error("ping run fails:%s", err)
			continue
		}
		// 本次ping,与之前ping所有结果打包
		// logs.Debug("ping result:", m)
		for ip, d := range m {
			r := result[ip]
			// 已经有结果，则更新
			if r != nil {
				r.Times++
				r.Duration += d
				if d >= r.MaxRtt {
					r.MaxRtt = d
				}
				if d < r.MinRtt {
					r.MinRtt = d
				}
			} else {
				// 首次ping
				result[ip] = &PingStat{
					Times:    1,
					Duration: d,
					MaxRtt:   d,
					MinRtt:   d,
				}
			}
		}
		diff := time.Since(now)
		select {
		case <-timer.C:
			logs.Debug("ping round:%d finished, %dip ping use time %s", send, len(ips), diff.String())
		default:
			logs.Debug("finish %d ip ping with %s,less than %d ms,so wait %d ms",
				len(ips), diff.String(), sendPackageInterMin, sendPackageWait)
			time.Sleep(time.Duration(sendPackageWait) * time.Millisecond)
		}
		timer.Stop()
	}
	return
}
