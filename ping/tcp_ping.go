package ping

import (
	"github.com/astaxie/beego/logs"
	"testnode-pinger/util"
	"time"
)

func TcpOptionFromConf(conf *util.Conf) *util.TcpPingOption {
	return &util.TcpPingOption{
		PacketSize:     conf.TCPPingPacketSize,
		Count:          conf.TCPPingPacketCount,
		DstPort:        conf.TCPPingDestPort,
		TimeoutMs:      conf.TCPPingTimeoutMs,
		ConnBufferSize: conf.TCPPingConnBufferSize,
		SendGoCount:    conf.TCPPingSendGoCount,
		ReceiveGoCount: conf.TCPPingRecvGoCount,
		ProcessGoCount: conf.TCPPingProcGoCount,
		IntervalMs:     conf.TCPPingInterval,
	}
}
func TCPPing(ipsGetter util.IPsGetter, conf *util.Conf) map[*util.PingIP]*util.Statistics {
	localCN, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logs.Error("time.LoadLocation failed with err:", err)
		return nil
	}
	pingTime := time.Now().In(localCN).Format("2006-01-02 15:04:05")
	pips := ipsGetter.GetIPs()
	// ips := make([]string, 0, len(pips))
	// for _, pip := range pips {
	// 	ips = append(ips, pip.IP)
	// }
	tcppingOption := TcpOptionFromConf(conf)
	times := tcppingOption.Count
	allRes := make(map[*util.PingIP]*util.Statistics)
	for _, ip := range pips {
		s := util.NewStatistics(times)
		allRes[ip] = s
		s.TcpPingResult.PingAtTime = pingTime
	}
	sipMap := make(map[string]string)
	for i := 0; i < times; i++ {
		logs.Debug("tcp ping round:%d start", i)
		start := time.Now()
		tcpp, err := util.NewTcpPinger(tcppingOption)
		if err != nil {
			logs.Error("NewTCPPinger fails with error:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		for _, pip := range pips {
			err := tcpp.AddIP(pip.IP)
			if err != nil {
				logs.Error("TCPPing add ip fails with error:", err)
				time.Sleep(10 * time.Second)
				continue
			}
		}
		res, srcMap, err := tcpp.Run()
		if err != nil {
			logs.Error("TCPPing Run fails with error:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		sipMap = srcMap
		//此遍历没有统计丢包的rtt
		for pip, s := range allRes {
			s.SentPackets++
			rtt, ok := res[pip.IP]
			if !ok {
				s.LossPackets++
				continue
			}
			s.RecvPackets++
			s.TotalRtt += rtt
			s.Rtts = append(s.Rtts, rtt)
		}
		for pip, s := range allRes {
			sip := sipMap[pip.IP]
			s.Calculate()
			logs.Debug("ip %s,srcIP %s,result %s", pip.IP, sip, s.TcpPingResult.String())
		}
		logs.Debug("tcp ping round:%d finished,ip count %d,use time %s", i, len(pips), time.Now().Sub(start))
	}
	return allRes
}
