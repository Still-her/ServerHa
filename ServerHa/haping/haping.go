package haping

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"ha/log"
	"math"
	"net"
	"os/exec"
	"time"
)

// tcp 报文前20个是报文头，后面的才是 ICMP 的内容。
// ICMP：组建 ICMP 首部（8 字节） + 我们要传输的内容
// ICMP 首部：type、code、校验和、ID、序号，1 1 2 2 2
// 回显应答：type = 0，code = 0
// 回显请求：type = 8, code = 0

// 回复应答（ICMP类型0）：ping命令用到该类型的数据包以测试TCP/IP连接；
// 目标不可达 （ICMP类型3）：用以知识目标网络、主机或者端口不可达；
// 源站抑制 （ICMP类型4）：当路由器处理IP数据的速度不够快时，会发送此类的消息。它的意思是让发送方降低发送数据的速率。Microsoft Windows NT或Windows 2000主机可以通过降低数据传输率来响应这种类型的消息；
// 重定向消息 （ICMP类型5）：用于将主机重新定向到一个不同的网络路径，该消息告诉路由器对于该数据包可以忽略它内部的路由表项；
// 回复请求（ICMP类型8）：ping命令用该类型的数据包测试TCP/IP连接；
// 路由器通告 （ICMP类型9）：以随机的时间间隔发送该数据包以响应
// 路由器请求 （ICMP类型10）：路由器发送该数据包来请求路由器通告的更新；
// 超时 （ICMP类型11）：指示数据包由于通过了太多的网段，其的生存时间（TTL）已经过期，Tracert命令用此消息来测试本地和远程主机之间的多个路由器；
// 参数问题 （ICMP类型12）：用以指示处理IP数据包头时出错
// ICMP 序号不能乱

type ICMP struct {
	Type        uint8  // 类型
	Code        uint8  // 代码
	CheckSum    uint16 // 校验和
	ID          uint16 // ID
	SequenceNum uint16 // 序号
}

// ICMP 序号不能乱
type PingNCMU struct {
	Dettmult   int
	Detectmult int
	PingConn   net.Conn
}

// 回调函数
type CallbackFunc func(ipAddr string, recv int)

type IpaddrDetectmult struct {
	Ip           string
	Dettmult     int
	Detectmult   int
	Resetnetconn bool
	Readagain    int
}

// 求校验和
func checkSum(data []byte) uint16 {
	// 第一步：两两拼接并求和
	length := len(data)
	index := 0
	var sum uint32
	for length > 1 {
		// 拼接且求和
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		length -= 2
		index += 2
	}
	// 奇数情况，还剩下一个，直接求和过去
	if length == 1 {
		sum += uint32(data[index])
	}

	// 第二部：高 16 位，低 16 位 相加，直至高 16 位为 0
	hi := sum >> 16
	for hi != 0 {
		sum = hi + uint32(uint16(sum))
		hi = sum >> 16
	}
	// 返回 sum 值 取反
	return uint16(^sum)
}

func CmdPing(addr []IpaddrDetectmult, pingback CallbackFunc) {

	for index, value := range addr {
		addr[index].Dettmult = value.Detectmult
		addr[index].Resetnetconn = true
		addr[index].Readagain = 0
	}

	for i := 0; i < math.MaxInt64; i++ {

		for index, value := range addr {
			var SendTimes int = 0
			var RecvTimes int = 0
			// 要ping的地址
			address := value.Ip
			// 构建ping命令
			cmd := exec.Command("ping", "-c", "1", "-t", "1", address)
			// 创建buffer来捕获标准输出
			var out bytes.Buffer
			cmd.Stdout = &out
			// 发送数 ++
			SendTimes++
			// 执行命令
			err := cmd.Start()
			if err != nil {
				log.Error("Ping start:", err.Error(), " address: ", address, " Dettmult: ", addr[index].Dettmult)
				addr[index].Dettmult--
				if addr[index].Dettmult <= 0 {
					pingback(address, 0)
					addr[index].Dettmult = addr[index].Detectmult
					SendTimes = 0
					RecvTimes = 0
				}
				continue
			}
			// 等待命令完成
			err = cmd.Wait()
			if err != nil {
				log.Error("Ping wait:", err.Error(), " address: ", address, " Dettmult: ", addr[index].Dettmult)
				addr[index].Dettmult--
				if addr[index].Dettmult <= 0 {
					pingback(address, 0)
					addr[index].Dettmult = addr[index].Detectmult
					SendTimes = 0
					RecvTimes = 0
				}
				continue
			}
			log.Error("Ping output address:", address)
			log.Error(out.String())
			// 接受数 ++
			RecvTimes++
			log.Error("Ping ok RecvTimes:", RecvTimes, " address: ", address)
			pingback(address, RecvTimes)
			addr[index].Dettmult = addr[index].Detectmult
		}
		time.Sleep(time.Second)
	}
}

func Ping(addr []IpaddrDetectmult, pingback CallbackFunc) {

	conns := make([]net.Conn, len(addr), len(addr))
	errs := make([]error, len(addr), len(addr))

	for index, _ := range addr {
		addr[index].Dettmult = addr[index].Detectmult
		addr[index].Resetnetconn = true
		conns[index], errs[index] = net.DialTimeout("ip:icmp", addr[index].Ip, time.Second)
		if errs[index] != nil {
			log.Error("Ping net.DialTimeout return ping:", errs[index].Error())
			continue
		}
	}

	for i := 0; i < math.MaxInt64; i++ {

		for index, _ := range addr {

			if nil == conns[index] {
				log.Error("ping ip :", addr[index].Ip, " ping failed, try use ip route del blackhole")
				conns[index], errs[index] = net.DialTimeout("ip:icmp", addr[index].Ip, time.Second)
				if errs[index] != nil {
					addr[index].Dettmult--
					log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, err:%s ", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult, errs[index].Error()))
					if addr[index].Dettmult <= 0 {
						log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, this ip is bad", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
						pingback(addr[index].Ip, 0)
						addr[index].Dettmult = addr[index].Detectmult
						addr[index].Readagain = 0
					}
					continue
				}
			}

			conns[index].SetDeadline(time.Now().Add(time.Second))
			conns[index].SetReadDeadline(time.Now().Add(time.Second))
			conns[index].SetWriteDeadline(time.Now().Add(time.Second))

			// 构建请求
			icmp := &ICMP{
				Type:        8,
				Code:        0,
				CheckSum:    uint16(0),
				ID:          uint16(index),
				SequenceNum: uint16(index),
			}

			// 将请求转为二进制流
			var buffer bytes.Buffer
			binary.Write(&buffer, binary.BigEndian, icmp)
			// 请求的数据
			data := make([]byte, 32)
			data[28] = 8
			data[29] = 8
			data[30] = 8
			data[31] = 8
			// 将请求数据写到 icmp 报文头后
			buffer.Write(data)
			data = buffer.Bytes()
			// ICMP 请求签名（校验和）：相邻两位拼接到一起，拼接成两个字节的数
			checkSum := checkSum(data)
			// 签名赋值到 data
			data[2] = byte(checkSum >> 8)
			data[3] = byte(checkSum)
			startTime := time.Now()

			var SendTimes int = 0
			var RecvTimes int = 0
			// 远程地址
			remoteaddr := conns[index].RemoteAddr()
			// 将 data 写入连接中，
			n, err := conns[index].Write(data)
			if err != nil {
				//log.Error("Ping net.Write:", err.Error(), " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
				if true == addr[index].Resetnetconn {
					conns[index].Close()
					//log.Error("net.Write err and reset ping net.DialTimeout addr :", addr[index].Ip)
					conns[index], errs[index] = net.DialTimeout("ip:icmp", addr[index].Ip, time.Second)
					if errs[index] != nil {
						//log.Error("net.Write reset ping net.DialTimeout error:", errs[index].Error(), " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
					} else {
						//log.Error("net.Write reset ping net.DialTimeout ok", " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
						addr[index].Resetnetconn = false
					}
				}

				addr[index].Dettmult--
				log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, err:%s ", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult, err.Error()))
				if addr[index].Dettmult <= 0 {
					log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, this ip is bad", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
					pingback(addr[index].Ip, 0)
					addr[index].Dettmult = addr[index].Detectmult
					addr[index].Readagain = 0
					SendTimes = 0
					RecvTimes = 0
				}
				continue
			}
			// 发送数 ++
			SendTimes++
			// 接收响应
			RecvTimeStart := time.Now().Unix()

		READAGAIN:
			buf := make([]byte, 1024)
			n, err = conns[index].Read(buf)

			//fmt.Println(data)
			if err != nil {
				//log.Error("Ping net.Read:", err.Error(), " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
				if true == addr[index].Resetnetconn {
					conns[index].Close()
					//log.Error("net.Read err and reset ping net.DialTimeout addr :", addr[index].Ip)
					conns[index], errs[index] = net.DialTimeout("ip:icmp", addr[index].Ip, time.Second)
					if errs[index] != nil {
						//log.Error("net.Read reset ping net.DialTimeout error:", errs[index].Error(), " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
					} else {
						//log.Error("net.Read reset ping net.DialTimeout ok", " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
						addr[index].Resetnetconn = false
					}
				}

				addr[index].Dettmult--
				log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, err:%s ", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult, err.Error()))
				if addr[index].Dettmult <= 0 {
					log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, this ip is bad", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
					pingback(addr[index].Ip, 0)
					addr[index].Dettmult = addr[index].Detectmult
					addr[index].Readagain = 0
					SendTimes = 0
					RecvTimes = 0
				}

				continue
			} else {
				if n != 60 || buf[56] != 8 || buf[57] != 8 || buf[58] != 8 || buf[59] != 8 {
					addr[index].Readagain++
					//log.Error("net.Read recv ip: ", addr[index].Ip, " other data again try read and read times :", addr[index].Readagain, " now time: ", time.Now().Unix(), " readstartime: ", RecvTimeStart)
					if time.Now().Unix()-RecvTimeStart > 1 {
						addr[index].Dettmult--
						log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, err: n != 60 ", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
						if addr[index].Dettmult <= 0 {
							log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, this ip is bad", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
							pingback(addr[index].Ip, 0)
							addr[index].Dettmult = addr[index].Detectmult
							addr[index].Readagain = 0
							SendTimes = 0
							RecvTimes = 0
						}
						continue
					}
					goto READAGAIN
				}
				addr[index].Readagain = 0
				if buf[20] != 0 {
					//log.Info("Ping net.Read ok but type is :", buf[20], " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult)
					if true == addr[index].Resetnetconn {
						conns[index].Close()
						//log.Error("net.Read err and reset ping net.DialTimeout addr :", addr[index].Ip, " type: ", buf[20])
						conns[index], errs[index] = net.DialTimeout("ip:icmp", addr[index].Ip, time.Second)
						if errs[index] != nil {
							//log.Error("Ping net.Read ok but type reset ping net.DialTimeout error:", errs[index].Error(), " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult, " type: ", buf[20])
						} else {
							//log.Error("Ping net.Read ok but type reset ping net.DialTimeout ok", " IP: ", addr[index].Ip, " Dettmult: ", addr[index].Dettmult, " type: ", buf[20])
							addr[index].Resetnetconn = false
						}
					}
					addr[index].Dettmult--
					log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, err: buf[20] != 0 ", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
					if addr[index].Dettmult <= 0 {
						log.Error(fmt.Sprintf("Ping net.DialTimeout return ping ip: %s, times: %d, this ip is bad", addr[index].Ip, addr[index].Detectmult-addr[index].Dettmult))
						pingback(addr[index].Ip, 0)
						addr[index].Dettmult = addr[index].Detectmult
						addr[index].Readagain = 0
						SendTimes = 0
						RecvTimes = 0
					}
					continue
				}
			}
			var byteformat string
			for i := 0; i < n; i++ {
				byteformat += fmt.Sprintf("Ip:%s byte value[%d]:%x\n", addr[index].Ip, i, buf[i])
			}
			//log.Info("byteformat:\n", byteformat)

			// 接受数 ++
			RecvTimes++
			//log.Info("Ping ConnList index:", index, " ConnListaddr: ", conns[index])
			//log.Info("Ping ok RecvTimes:", RecvTimes, " nbyte: ", n, " ipaddr: ", conns[index].RemoteAddr(), " type: ", buf[20])
			addr[index].Resetnetconn = true
			pingback(remoteaddr.String(), RecvTimes)
			addr[index].Dettmult = addr[index].Detectmult
			t := time.Since(startTime).Milliseconds()
			msg := fmt.Sprintf("Ping 来自 %d.%d.%d.%d 的回复：字节=%d 时间=%d TTL=%d TYPE=%d\n", buf[12], buf[13], buf[14], buf[15], n-28, t, buf[8], buf[20])
			log.Info(msg)
		}
		time.Sleep(time.Second)
	}
}
