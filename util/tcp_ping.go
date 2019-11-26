package util

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	DefaultPacketSize     = 56
	DefaultPacketCount    = 50
	DefaultTcpPort        = 80
	DefaultTimeoutMs      = 800
	DefaultBufferSize     = 10240000
	DefaultSendGoCount    = 1
	DefaultReceiveGoCout  = 2
	DefaultProcessGoGount = 4
	DefaultIntervalMs     = 500
)

type TcpPingOption struct {
	PacketSize     int    `json:"tcpping_packet_size"`  // 数据包大小
	Count          int    `json:"tcpping_packet_count"` // 数据包的总数量
	LocalIp        string // 本地IP
	DstPort        uint16 `json:"tcpping_dest_port"`  // 目的端口
	TimeoutMs      int    `json:"tcpping_timeout_ms"` // 超时时间
	ConnBufferSize int    `json:"tcpping_conn_buffer_size"`
	SendGoCount    int    `json:"tcpping_send_go_count"`
	ReceiveGoCount int    `json:"tcpping_receive_go_count"`
	ProcessGoCount int    `json:"tcpping_process_go_count"`
	IntervalMs     int    `json:"tcpping_interval_ms"` //间隔时间
}

type TcpPingResult struct {
	SentPackets int     `json:"sentPackets"` // 发送的数据包个数
	RecvPackets int     `json:"recvPackets"` // 收到的数据包个数
	LossPackets int     `json:"lossPackets"` // LossPackets = SentPackets - RecvPackets
	LossRate    float64 `json:"lossRate"`    // 丢包率 LossRate = LossPackets/SentPackets
	AvgRttMs    float64 `json:"avgRttMs"`    // 平均Rtt
	MaxRttMs    float64 `json:"maxRttMs"`    // 最大Rtt
	MinRttMs    float64 `json:"minRttMs"`    // 最小Rtt
	Mdev        float64 `json:"mdev"`        // Rtt的标准差
}

func (s *TcpPingResult) String() string {
	return fmt.Sprintf("%d packets transmitted, %d packets received, %.2f%% packet loss, "+
		"round-trip min/avg/max/mdev = %.3f/%.3f/%.3f/%.3f ms",
		s.SentPackets, s.RecvPackets, s.LossRate, s.MinRttMs, s.AvgRttMs, s.MaxRttMs, s.Mdev)
}

type DurationSlice []time.Duration

func (ds DurationSlice) Len() int {
	return len(ds)
}
func (ds DurationSlice) Less(i, j int) bool {
	return ds[i] < ds[j]
}
func (ds DurationSlice) Swap(i, j int) {
	ds[i], ds[j] = ds[j], ds[i]
}

type Statistics struct {
	TcpPingResult
	Rtts     DurationSlice // sorted
	TotalRtt time.Duration
}

func NewStatistics(count int) *Statistics {
	return &Statistics{
		Rtts: make([]time.Duration, 0, count),
	}
}

func (s *Statistics) Calculate() {
	if len(s.Rtts) > 0 {
		sort.Sort(s.Rtts)
		s.MaxRttMs = s.Rtts[len(s.Rtts)-1].Seconds() * 1000
		s.MinRttMs = s.Rtts[0].Seconds() * 1000
		avg := s.TotalRtt / time.Duration(len(s.Rtts))
		s.AvgRttMs = avg.Seconds() * 1000
		var sumSquares time.Duration
		for _, rtt := range s.Rtts {
			sumSquares += (rtt - avg) * (rtt - avg)
		}
		s.Mdev = time.Duration(math.Sqrt(float64(sumSquares/time.Duration(len(s.Rtts))))).Seconds() * 1000

		s.LossPackets = s.SentPackets - s.RecvPackets
		s.LossRate = float64(s.LossPackets*100) / float64(s.SentPackets)
	} else {
		s.LossPackets = s.SentPackets
		s.LossRate = 100
		s.AvgRttMs = float64(MaxRtt)
	}
}

func (s *Statistics) String() string {
	return fmt.Sprintf("%d packets transmitted, %d packets received, %.1f%% packet loss \n"+
		"round-trip min/avg/max/mdev = %.3f/%.3f/%.3f/%.3f ms",
		s.SentPackets, s.RecvPackets, s.LossRate, s.MinRttMs, s.AvgRttMs, s.MaxRttMs, s.Mdev)

}

type EchoResult struct {
	bytes     []byte
	recvTime  time.Time
	poolBytes []byte // 从内存池里面分配出来的内存
	Addr      string
	srcAddr   string // 源IP
	Seq       uint32
	Data      []byte
	Rtt       time.Duration
}

func getLocalIP(dstip net.IP) (localIP net.IP, err error) {
	serverAddr, err := net.ResolveUDPAddr("udp", dstip.String()+":12345")
	if err != nil {
		return
	}
	// don't actually connect, but determine ip what source ip to use based on destination
	for i := 0; i < 3; i++ {
		if con, err := net.DialUDP("udp", nil, serverAddr); err == nil {
			if udpaddr, ok := con.LocalAddr().(*net.UDPAddr); ok {
				return udpaddr.IP, nil
			}
		}
	}
	err = errors.New("can not find localIP and localPort")
	return
}

type TcpPinger struct {
	remoteAddr       *net.IPAddr   // 目的IP
	srcPort, dstPort uint16        // 端口
	packetSize       int           // 数据包大小, max size is 1460 = 1500(MTU) - 20(IP Header) - 20(Tcp Header)
	count            int           // 总的数据包个数, count <= 0 代表没限制
	interval         time.Duration // 每个包的间隔时间(最小可设置的值为10ms)
	timeout          time.Duration // 总的超时时间
	isFlood          bool          // 是否需要无间隔发送(总的发送包数还是由Count指定)
	localIp          string        // 本地IP
	endRecvWaitS     int           // 最后一个包的等待时间
	bufferSize       int           // 底层连接的读缓冲区的大小
	curAddrCount     int
	repliedAddrCount int
	addrs            map[string]bool // 需要ping的地址列表
	addrsChannel     chan string
	timeMutex        sync.Mutex
	sendGoCount      int
	receiveGoCount   int
	processGoCount   int
	resultChannel    chan *EchoResult
	wgSend           *sync.WaitGroup
	wgRecv           *sync.WaitGroup
	wgProcess        *sync.WaitGroup
	stop             chan bool
	allReceived      chan bool
	sendTimes        map[string]time.Time
	rttMap           map[string]time.Duration
	seq              uint32
	srcMap           map[string]string // 源IP的映射关系
}

func NewTcpPinger(pingOpt *TcpPingOption) (*TcpPinger, error) {
	localAddr, err := getLocalIP(net.ParseIP("114.114.114.114"))
	if err != nil {
		logs.Error("getLocalIP failed.%s", err.Error())
		return nil, err
	}
	if pingOpt.TimeoutMs == 0 {
		pingOpt.TimeoutMs = DefaultTimeoutMs
	}
	if pingOpt.IntervalMs == 0 {
		pingOpt.IntervalMs = DefaultIntervalMs
	}
	if pingOpt.Count == 0 {
		pingOpt.Count = DefaultPacketCount
	}
	if pingOpt.PacketSize == 0 {
		pingOpt.PacketSize = DefaultPacketSize
	}
	if pingOpt.DstPort == 0 {
		pingOpt.DstPort = DefaultTcpPort
	}
	if pingOpt.ConnBufferSize == 0 {
		pingOpt.ConnBufferSize = DefaultBufferSize
	}
	if pingOpt.SendGoCount == 0 {
		pingOpt.SendGoCount = DefaultSendGoCount
	}
	if pingOpt.ReceiveGoCount == 0 {
		pingOpt.ReceiveGoCount = DefaultReceiveGoCout
	}
	if pingOpt.ProcessGoCount == 0 {
		pingOpt.ProcessGoCount = DefaultProcessGoGount
	}
	if pingOpt.LocalIp == "" {
		logs.Debug("pingOpt.LocalIp is nil,get a localIP:", localAddr)
		pingOpt.LocalIp = localAddr.String()
	} else {
		if net.ParseIP(pingOpt.LocalIp) == nil {
			details := fmt.Sprintf("%s is not an ip address", pingOpt.LocalIp)
			logs.Warn(details)
			err := errors.New(details)
			return nil, err
		}
	}

	pinger := TcpPinger{
		srcPort:        uint16(GetLocalPort()),
		timeout:        time.Duration(pingOpt.TimeoutMs) * time.Millisecond,
		dstPort:        pingOpt.DstPort,
		packetSize:     pingOpt.PacketSize,
		localIp:        pingOpt.LocalIp,
		sendGoCount:    pingOpt.SendGoCount,
		receiveGoCount: pingOpt.ReceiveGoCount,
		processGoCount: pingOpt.ProcessGoCount,
		interval:       time.Duration(pingOpt.IntervalMs) * time.Millisecond,
		bufferSize:     pingOpt.ConnBufferSize,
		addrs:          make(map[string]bool, 128),
		sendTimes:      make(map[string]time.Time, 128),
		rttMap:         make(map[string]time.Duration, 128),
		seq:            rand.Uint32()%332528 + 332527,
		addrsChannel:   make(chan string, MaxAddrCount+1),
		wgSend:         new(sync.WaitGroup),
		wgRecv:         new(sync.WaitGroup),
		wgProcess:      new(sync.WaitGroup),
		resultChannel:  make(chan *EchoResult, 1024000),
		stop:           make(chan bool),
		allReceived:    make(chan bool),
		srcMap:         make(map[string]string),
	}

	return &pinger, nil
}

func (p *TcpPinger) AddIP(ipaddr string) error {
	if p.curAddrCount >= MaxAddrCount {
		return fmt.Errorf("cannot add IP because you have already add %d IPs, max count %d",
			p.curAddrCount, MaxAddrCount)
	}

	// add to map
	addr := net.ParseIP(ipaddr)
	if addr == nil {
		return fmt.Errorf("%s is not a valid textual representation of an IP address", ipaddr)
	}
	p.addrs[addr.String()] = true

	p.curAddrCount += 1

	// add to channel
	p.addrsChannel <- ipaddr

	return nil
}

func (p *TcpPinger) Run() (result map[string]time.Duration, srcMap map[string]string, err error) {
	start := time.Now()
	close(p.addrsChannel) // must close it

	conn, err := p.listen()
	if err != nil {
		close(p.stop)
		return
	}
	defer conn.Close()

	if len(p.addrs) == 0 {
		msg := "there is no IP to tcp ping"
		logs.Error(msg)
		err = errors.New(msg)
		close(p.stop)
		return
	}
	p.curAddrCount = len(p.addrs)
	err = conn.SetReadBuffer(p.bufferSize)
	if err != nil {
		logs.Error("SetReadBuffer to %d fails with error %s", p.bufferSize, err)
		close(p.stop)
		return
	}
	conn.SetReadDeadline(time.Now().Add(p.timeout))

	for i := 0; i < p.processGoCount; i++ {
		p.wgProcess.Add(1)
		go p.processTcpMsg()
	}

	for i := 0; i < p.receiveGoCount; i++ {
		p.wgRecv.Add(1)
		go p.recvTcpMsg(conn)
	}

	for i := 0; i < p.sendGoCount; i++ {
		p.wgSend.Add(1)
		go p.sendTcpMsg(conn)
	}

	ticker := time.NewTicker(p.timeout)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		close(p.stop)
	case <-p.allReceived:
		//fmt.Printf("all tcp ping reply received %s \n", time.Now())
		close(p.stop)
	}

	p.wgRecv.Wait()
	close(p.resultChannel)

	p.wgSend.Wait()
	p.wgProcess.Wait()

	// 防止allReceived 的channel泄露
	select {
	case <-p.allReceived:
	default:
		close(p.allReceived)
	}

	// 接收到所有的回包了， 但是没有到timeout的时间点， 我们sleep到timeout时间点
	// 用来防止发包太快
	end := time.Now()
	used := end.Sub(start)
	if p.timeout-used > 0 {
		logs.Debug("timeout ms:%d, used ms:%d, sleep ms %d",
			p.timeout/time.Millisecond, used/time.Millisecond, (p.timeout-used)/time.Millisecond)
		time.Sleep(p.timeout - used)
	}

	return p.rttMap, p.srcMap, nil
}

func (p *TcpPinger) listen() (*net.IPConn, error) {
	conn, err := net.ListenPacket("ip4:tcp", p.localIp)
	if err != nil {
		logs.Error("listening Tcp packets failed:", err)
		return nil, err
	} else {
		return conn.(*net.IPConn), nil
	}
}

func (p *TcpPinger) sendTcpMsg(conn *net.IPConn) error {
	defer p.wgSend.Done()
	for ip := range p.addrsChannel {
		for {
			msg, err := p.constructSynMsg(ip)
			if err != nil {
				logs.Error("serialize syn message failed:", err)
				return err
			}

			remoteAddr, err := net.ResolveIPAddr("ip4", ip)
			if err != nil {
				logs.Error("ResolveIPAddr failed.%s", err.Error())
				return err
			}

			p.timeMutex.Lock()
			p.sendTimes[ip] = time.Now()
			p.timeMutex.Unlock()
			if _, err := conn.WriteTo(msg, remoteAddr); err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Err == syscall.ENOBUFS {
						continue
					}
					err = neterr
				}
				logs.Error("TCP send package failed:", err)
				return err
			}
			break
		}
	}
	return nil
}

func (p *TcpPinger) constructSynMsg(strIp string) ([]byte, error) {
	ip := &layers.IPv4{
		SrcIP:    net.ParseIP(p.localIp),
		DstIP:    net.ParseIP(strIp),
		Protocol: layers.IPProtocolTCP,
	}

	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(p.srcPort),
		DstPort: layers.TCPPort(p.dstPort),
		SYN:     true,
		Seq:     p.seq,
		// some server may filter packets by mss option and window size
		Window: 1024,
		// set mss 1460
		Options: []layers.TCPOption{{layers.TCPOptionKindMSS, 4, []byte{5, 180}}},
	}
	tcp.SetNetworkLayerForChecksum(ip)
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}
	if err := gopacket.SerializeLayers(buf, opts, tcp, gopacket.Payload(make([]byte, p.packetSize))); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *TcpPinger) constructRstMsg(seq, ack uint32) ([]byte, error) {
	ip := &layers.IPv4{
		SrcIP:    net.ParseIP(p.localIp),
		DstIP:    p.remoteAddr.IP,
		Protocol: layers.IPProtocolTCP,
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(p.srcPort),
		DstPort: layers.TCPPort(p.dstPort),
		RST:     true,
		ACK:     true,
		Seq:     seq,
		Ack:     ack,
		// some server may filter packets by mss option and window size
		Window: 1024,
		// set mss 1460
		Options: []layers.TCPOption{{layers.TCPOptionKindMSS, 4, []byte{5, 180}}},
	}
	tcp.SetNetworkLayerForChecksum(ip)
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}
	if err := gopacket.SerializeLayers(buf, opts, tcp, gopacket.Payload(make([]byte, p.packetSize))); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *TcpPinger) recvTcpMsg(conn *net.IPConn) {
	defer p.wgRecv.Done()

	for {
		select {
		case <-p.stop:
			return
		default:
			//general 40 is ok (IP header 20 + TCP header 20)
			// ipMsg := make([]byte, 64)
			ipMsg := bytePool.Get().([]byte)
			n, addr, err := conn.ReadFrom(ipMsg)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						// Read timeout
						conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
						bytePool.Put(ipMsg)
						continue
					}
				}
				logs.Error("conn.ReadFrom(ipMsg) fails with error %s", err)
				bytePool.Put(ipMsg)
				return
			}

			strIP := addr.String()
			if p.addrs[strIP] {
				echo := &EchoResult{
					bytes:     ipMsg[:n],
					recvTime:  time.Now(),
					Addr:      strIP,
					srcAddr:   conn.LocalAddr().String(),
					poolBytes: ipMsg,
				}
				p.resultChannel <- echo
			} else {
				bytePool.Put(ipMsg)
			}
		}
	}
}

func (p *TcpPinger) processTcpMsg() {
	defer p.wgProcess.Done()
	for echoResult := range p.resultChannel {
		packet := gopacket.NewPacket(echoResult.bytes, layers.LayerTypeTCP, gopacket.Lazy)
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			bytePool.Put(echoResult.poolBytes)
			logs.Error("message isn't a tcp packet")
			continue
		}
		tcp, _ := tcpLayer.(*layers.TCP)
		if uint16(tcp.DstPort) != p.srcPort {
			bytePool.Put(echoResult.poolBytes)
			continue
		}
		if tcp.SYN && tcp.ACK {
			echoResult.Seq = tcp.Ack - 1
		} else if tcp.ACK && tcp.RST {
			echoResult.Seq = tcp.Ack - uint32(p.packetSize) - 1
		} else {
			bytePool.Put(echoResult.poolBytes)
			continue
		}

		if echoResult.Seq != p.seq {
			// logs.Debug("sequence not equal")
			bytePool.Put(echoResult.poolBytes)
			continue
		}

		// logs.Debug("receive tcp ping packet, ip %s, seq %d", echoResult.Addr, p.seq)

		p.timeMutex.Lock()
		p.repliedAddrCount += 1
		if p.repliedAddrCount == p.curAddrCount {
			close(p.allReceived)
			logs.Debug("all packet received, p.repliedAddrCount %d", p.repliedAddrCount)
		}
		p.rttMap[echoResult.Addr] = echoResult.recvTime.Sub(p.sendTimes[echoResult.Addr])
		p.srcMap[echoResult.Addr] = echoResult.srcAddr
		p.timeMutex.Unlock()
		bytePool.Put(echoResult.poolBytes)
	}
	return
}
