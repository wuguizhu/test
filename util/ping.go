package util

// Package fastping is an ICMP ping library inspired by AnyEvent::FastPing Perl
// module to send ICMP ECHO REQUEST packets quickly. Original Perl module is
// available at
// http://search.cpan.org/~mlehmann/AnyEvent-FastPing-2.01/
//
// It hasn't been fully implemented original functions yet.
//
// Here is an example:
//
//	p := fastping.NewPinger()
//	ra, err := net.ResolveIPAddr("ip4:icmp", os.Args[1])
//	if err != nil {
//		fmt.Println(err)
//		os.Exit(1)
//	}
//	p.AddIPAddr(ra)
//	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
//		fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
//	}
//	p.OnIdle = func() {
//		fmt.Println("finish")
//	}
//	err = p.Run()
//	if err != nil {
//		fmt.Println(err)
//	}
//
// It sends an ICMP packet and wait a response. If it receives a response,
// it calls "receive" callback. After that, MaxRTT time passed, it calls
// "idle" callback. If you need more example, please see "cmd/ping/ping.go".
//
// This library needs to run as a superuser for sending ICMP packets when
// privileged raw ICMP endpoints is used so in such a case, to run go test
// for the package, please run like a following
//
//	sudo go test
//

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/astaxie/beego/logs"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var bytePool sync.Pool

func init() {
	bytePool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 512)
			return b
		},
	}
}

const (
	TimeSliceLength  = 16
	ProtocolICMP     = 1
	ProtocolIPv6ICMP = 58
	MaxAddrCount     = 4096
)

var (
	ipv4Proto = map[string]string{"ip": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"ip": "ip6:ipv6-icmp", "udp": "udp6"}
)

func fillData(b []byte, n int) {
	for i := 0; i < n; i++ {
		b[i] = byte(16 + i)
	}
}

func putTimeIntoBytes(b []byte, t time.Time) {

	// 秒和剩余的微秒分开处理
	sec := t.Unix()
	usec := t.UnixNano() / 1000 % 1000000

	//秒，填充前8个字节
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((sec >> (i * 8)) & 0xff)
	}
	//剩余的微秒，填充后8个字节
	for i := uint8(8); i < 16; i++ {
		b[i] = byte((usec >> ((i - 8) * 8)) & 0xff)
	}
}

func bytesToTime(b []byte) time.Time {
	var sec, usec int64
	// 解析秒
	for i := uint8(0); i < 8; i++ {
		sec += int64(b[i]) << (i * 8)
	}
	// 解析微妙
	for i := uint8(8); i < 16; i++ {
		usec += int64(b[i]) << ((i - 8) * 8)
	}
	return time.Unix(sec, usec*1000)
}

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

type packet struct {
	bytes   []byte
	addr    net.Addr
	srcAddr string
}

// Pinger represents ICMP packet sender/receiver
type Pinger struct {
	// key string is IPAddr.String()
	ip2id            map[string]int
	seq              int
	result           map[string]time.Duration
	isIPv6           bool
	addrsChannel     chan net.IP
	curAddrCount     int       // the count of address already added
	repliedAddrCount int       // the count of ip address which get ping reply
	finish           chan bool // if all reply received, close this channel
	network          string
	source           string
	mu               sync.Mutex
	stop             chan bool
	wgSend           *sync.WaitGroup
	wgRecv           *sync.WaitGroup
	wgProcess        *sync.WaitGroup

	// Size in bytes of the payload to send
	Size int
	// Number of (nano,milli)seconds of an idle timeout. Once it passed,
	// the library calls an idle callback function. It is also used for an
	// interval time of RunLoop() method
	MaxRTT time.Duration
	// OnRecv is called with a response packet's source address and its
	// elapsed time when Pinger receives a response packet.
	// OnRecv func(*net.IPAddr, time.Duration)
	// OnIdle is called when MaxRTT time passed
	// OnIdle func()
	// If Debug is true, it prints debug messages to stdout.
	Debug bool

	gorouteCount int
}

// NewPinger returns a new Pinger struct pointer
func NewPinger(seq int, srcIP string) *Pinger {
	return &Pinger{
		ip2id:  make(map[string]int),
		seq:    seq,
		result: make(map[string]time.Duration),
		isIPv6: func() bool {
			// 源IP为IPv6地址
			if srcIP != "" && strings.IndexByte(srcIP, ':') != -1 {
				return true
			}
			return false
		}(),
		addrsChannel:     make(chan net.IP, MaxAddrCount+1),
		finish:           make(chan bool),
		curAddrCount:     0,
		repliedAddrCount: 0,
		network:          "ip",
		source:           srcIP,
		Size:             TimeSliceLength,
		MaxRTT:           time.Second,
		stop:             make(chan bool),
		wgSend:           new(sync.WaitGroup),
		wgRecv:           new(sync.WaitGroup),
		wgProcess:        new(sync.WaitGroup),
		Debug:            false,
		gorouteCount:     32,
	}
}

// AddIP adds an IP address to Pinger. ipaddr arg should be a string like
// "192.0.2.1".
func (p *Pinger) AddIP(ipaddr string, id int) error {
	if p.curAddrCount >= MaxAddrCount {
		return fmt.Errorf("cannot add IP because you have already add %d IPs, max count %d", p.curAddrCount, MaxAddrCount)
	}

	// add to map
	addr := net.ParseIP(ipaddr)
	if addr == nil {
		return fmt.Errorf("%s is not a valid textual representation of an IP address", ipaddr)
	}
	// 源IP是IPv6但目的IP是IPv4 或 源IP是IPv4但目的IP是IPv6
	if p.isIPv6 && isIPv4(addr) || !p.isIPv6 && !isIPv4(addr) {
		return fmt.Errorf("the type of srcIP is not same as dstIP")
	}

	p.ip2id[addr.String()] = id
	p.curAddrCount++
	// add to channel
	p.addrsChannel <- addr

	return nil
}

// Run invokes a single send/receive procedure. It sends packets to all hosts
// which have already been added by AddIP() etc. and wait those responses. When
// it receives a response, it calls "receive" handler registered by AddHander().
// After MaxRTT seconds, it calls "idle" handler and returns to caller with
// an error value. It means it blocks until MaxRTT seconds passed. For the
// purpose of sending/receiving packets over and over, use RunLoop().
func (p *Pinger) Run() (map[string]time.Duration, string, error) {
	conn, err := func() (*icmp.PacketConn, error) {
		if p.isIPv6 {
			return p.listen(ipv6Proto[p.network], p.source)
		}
		return p.listen(ipv4Proto[p.network], p.source)
	}()
	if err != nil {
		close(p.stop)
		close(p.finish)
		// logs.Error("func failed with error:", err)
		return nil, p.source, err
	}
	defer conn.Close()
	// 因为p.source可能为空，所以在这里更新源IP地址
	if p.source == "" {
		p.source = conn.LocalAddr().String()
	}

	close(p.addrsChannel) // must close it
	recv := make(chan *packet, 40960)

	for i := 0; i < 4; i++ {
		p.wgRecv.Add(1)
		go p.recvICMP(conn, recv)
	}

	for i := 0; i < p.gorouteCount; i++ {
		p.wgProcess.Add(1)
		go p.procRecv(recv)
	}

	for i := 0; i < 1; i++ {
		p.wgSend.Add(1)
		go p.sendICMP(conn)
	}
	p.wgSend.Wait()

	ticker := time.NewTicker(p.MaxRTT)

	select {
	case <-ticker.C:
		close(p.stop)
	case <-p.finish:
		close(p.stop)
	}
	ticker.Stop()

	p.wgRecv.Wait()
	close(recv)

	p.wgProcess.Wait()
	// 防止finish 的channel泄露
	select {
	case <-p.finish:
	default:
		close(p.finish)
	}

	return p.result, p.source, nil
}

func (p *Pinger) listen(netProto string, source string) (*icmp.PacketConn, error) {
	conn, err := icmp.ListenPacket(netProto, source)
	if err != nil {
		// logs.Error("ListenPacket failed with error:", err)
		return nil, err
	}
	return conn, nil
}

func (p *Pinger) sendICMP(conn *icmp.PacketConn) error {
	defer p.wgSend.Done()
	var msgType icmp.Type

	for ip := range p.addrsChannel {
		if p.isIPv6 {
			msgType = ipv6.ICMPTypeEchoRequest
		} else {
			msgType = ipv4.ICMPTypeEcho
		}

		buf := bytePool.Get().([]byte)
		putTimeIntoBytes(buf, time.Now())

		if p.Size-TimeSliceLength != 0 {
			fillData(buf[TimeSliceLength:], p.Size-TimeSliceLength)
		}
		data := buf[0:p.Size]
		bytes, err := (&icmp.Message{
			Type: msgType, Code: 0,
			Body: &icmp.Echo{
				ID: p.ip2id[ip.String()], Seq: p.seq,
				Data: data,
			},
		}).Marshal(nil)

		if err != nil {
			bytePool.Put(buf)
			// logs.Error("Message Marshal failed with error:", err)
			return err
		}

		var dst net.Addr = &net.IPAddr{IP: ip}
		if p.network == "udp" {
			dst = &net.UDPAddr{IP: ip}
		}

		for {
			if _, err := conn.WriteTo(bytes, dst); err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Err == syscall.ENOBUFS {
						continue
					}
				}
				// logs.Error("conn.WriteTo failed with error:", err)
			}
			break
		}
		bytePool.Put(buf)
	}

	return nil
}

func (p *Pinger) recvICMP(conn *icmp.PacketConn, recv chan<- *packet) {
	defer p.wgRecv.Done()
	conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	for {
		select {
		case <-p.stop:
			return
		default:
		}

		// bytes := make([]byte, 128)
		bytes := bytePool.Get().([]byte)
		_, ra, err := conn.ReadFrom(bytes)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
				}
			}
			// logs.Error("conn.ReadFrom failed with error:", err)

			bytePool.Put(bytes)
			continue
		}

		select {
		case recv <- &packet{bytes: bytes, addr: ra, srcAddr: conn.LocalAddr().String()}:
		case <-p.stop:
			bytePool.Put(bytes)
			return
		}
	}
}

func (p *Pinger) procRecv(recv <-chan *packet) {
	defer p.wgProcess.Done()
	var bytes []byte
	for packet := range recv {
		bytes = packet.bytes

		var ipaddr *net.IPAddr
		switch adr := packet.addr.(type) {
		case *net.IPAddr:
			ipaddr = adr
		case *net.UDPAddr:
			ipaddr = &net.IPAddr{IP: adr.IP, Zone: adr.Zone}
		default:
			bytePool.Put(bytes)
			continue
		}

		addr := ipaddr.String()
		if _, ok := p.ip2id[addr]; !ok {
			bytePool.Put(bytes)
			continue
		}

		var m *icmp.Message
		var err error
		if m, err = icmp.ParseMessage(func() int {
			if p.isIPv6 {
				return ProtocolIPv6ICMP
			}
			return ProtocolICMP
		}(), bytes); err != nil {
			bytePool.Put(bytes)
			logs.Error("ParseMessage failed with error:", err)
			continue
		}

		if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
			bytePool.Put(bytes)
			continue
		}

		var rtt time.Duration
		switch pkt := m.Body.(type) {
		case *icmp.Echo:
			if pkt.ID == p.ip2id[addr] && pkt.Seq == p.seq {
				rtt = time.Since(bytesToTime(pkt.Data[:TimeSliceLength]))
				// fmt.Printf("process icmp id: %d(%04x), seq: %d(%04x)\n", pkt.ID, pkt.ID, pkt.Seq, pkt.Seq)
			} else {
				bytePool.Put(bytes)
				continue
			}
		default:
			bytePool.Put(bytes)
			continue
		}

		// in very rare case(system time is changed when pinging), the rtt is less than 0
		if rtt < 0 {
			rtt = 1 * time.Millisecond
		}

		p.mu.Lock()
		p.repliedAddrCount++
		if p.repliedAddrCount == p.curAddrCount {
			close(p.finish)
		}
		p.result[addr] = rtt
		p.mu.Unlock()
		bytePool.Put(bytes)
	}
}
