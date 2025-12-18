package internal

/*
#cgo CFLAGS:  -I../hacallback
#cgo LDFLAGS: -L../hacallback
#include "hacallback.h"
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

import (
	"fmt"
	"ha/config"
	"ha/log"
	"ha/vrrp"
	"net"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/vishvananda/netlink"
)

var (
	stavrrp       *vrrp.VirtualRouter
	stawithinsite Withinsiteinfo
)

// 给网管返回站点内的状态数据
type Withinsiteinfo struct {
	Statu    int    `gorm:"column:statu" json:"statu" form:"statu"`          //当前节点状态
	Nodepip  string `gorm:"column:nodepip" json:"nodepip" form:"nodepip"`    //主节点的物理ip
	When     string `gorm:"column:when" json:"when" form:"when"`             //上次切换时间
	User     string `gorm:"column:user" json:"user" form:"user"`             //上次切换用户
	Reason   string `gorm:"column:reason" json:"reason" form:"reason"`       //上次切换原因
	Routerid int    `gorm:"column:routerid" json:"routerid" form:"routerid"` //节点的优先级
	Preeempt bool   `gorm:"column:preeempt" json:"preeempt" form:"preeempt"` //节点是否抢占模式
	Priority int    `gorm:"column:priority" json:"priority" form:"priority"` //节点的优先级
	Netdev   string `gorm:"column:netdev" json:"netdev" form:"netdev"`       //网卡设备
	Itervip  string `gorm:"column:itervip" json:"itervip" form:"itervip"`    //虚拟IP
	Maskbit  string `gorm:"column:maskbit" json:"maskbit" form:"maskbit"`    //掩码位数
}

const (
	INTERNAL_OK        int = 0  //成功
	INTERNAL_INIT_NO   int = 1  //状态初始化
	INTERNAL_AUTO_STAU int = 20 //自动
	INTERNAL_HAND_STAU int = 21 //手动
)

const (
	START  int = 0
	BACKUP int = 1
	MASTER int = 2
	ERRSTA int = 3
)

func SendNmsInternalAlarmCode(alert int) {

	log.Error("Ha SendNmsInternalAlarmCode:", alert)
	alertcode := C.int(alert)
	C.InternalAlarmCodeCallback(alertcode)

}

func SendNmsInternalAlarmMsg(alert string) {
	log.Error("Ha SendNmsInternalAlarmMsg:", alert)
	alertmsg := C.CString(alert)
	C.InternalAlarmMsgCallback(alertmsg)
	C.free(unsafe.Pointer(alertmsg))
}

func InternalMaster() {
	log.Error("Ha InternalMaster")
	C.InternalMasterCallback()
}

func InternalBackup() {
	log.Error("ha InternalBackup")
	C.InternalBackupCallback()
}

func Getstawithinsitesta() Withinsiteinfo {
	return stawithinsite
}

func Getstatus() int {
	return stawithinsite.Statu
}

func GetVip() string {
	return stawithinsite.Itervip
}

// ARP数据包结构
type arpPacket struct {
	EthHdr  [14]byte // 以太网头部
	ARP_hdr [28]byte // ARP头部
}

// 字节序转换: 主机字节序到网络字节序 (16位)
func htons(host uint16) uint16 {
	return uint16((host<<8)&0xff00) | (host>>8)&0x00ff
}

// 获取网络接口的MAC地址
func getInterfaceMAC(interfaceName string) ([]byte, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("无法获取网络接口信息: %v\n提示: 检查接口名称是否正确", err)
	}
	return iface.HardwareAddr, nil
}

func bindbuildGARPPacket(interfaceName string, vip string) error {

	log.Error("bindbuildGARPPacket start")

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		fmt.Printf("网络接口不存在: %v\n", err)
		return err
	}
	if iface.Flags&net.FlagUp == 0 {
		fmt.Printf("网络接口 %s 未启用，请先启用网络接口\n命令示例: sudo ifconfig %s up", interfaceName, interfaceName)
	}
	log.Error("已确认网络接口 存在且处于活动状态: ", interfaceName)

	// 检查VIP地址是否有效
	vipAddr := net.ParseIP(vip)
	if vipAddr == nil {
		fmt.Printf("无效的VIP地址: %s", vip)
	}
	vip4 := vipAddr.To4()
	if vip4 == nil {
		fmt.Printf("VIP必须是IPv4地址: %s", vip)
	}

	// 获取网络接口的MAC地址
	macAddr, err := getInterfaceMAC(interfaceName)
	if err != nil {
		return fmt.Errorf("获取网络接口MAC地址失败: %v", err)
	}
	log.Error("成功获取网络接口MAC地址: ", macAddr)

	// 创建原始套接字

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ARP)))
	if err != nil {
		return fmt.Errorf("创建原始套接字失败: %v\n提示: 确保具有root权限，且内核支持原始套接字", err)
	}
	defer syscall.Close(fd)
	log.Error("成功创建原始套接字，文件描述符: ", fd)

	// 准备绑定地址结构
	var addr syscall.SockaddrLinklayer
	addr.Protocol = htons(syscall.ETH_P_ARP) // ARP协议类型
	addr.Ifindex = iface.Index               // 网络接口索引

	// 绑定到网络接口
	if err := syscall.Bind(fd, &addr); err != nil {
		var errMsg string
		var vmHint string
		if err.(syscall.Errno) == syscall.ENODEV || err.(syscall.Errno) == 6 {
			errMsg = "网络接口不存在或无效"
			vmHint = "\n提示1: 在虚拟机环境中，确保网络适配器设置为桥接模式而非NAT模式\n提示2: 检查网络接口名称是否正确，使用'ip link show'命令查看可用接口\n提示3: 确保网络接口已启用，使用'sudo ifconfig <接口名> up'命令启用"
		} else if err.(syscall.Errno) == syscall.EADDRNOTAVAIL {
			errMsg = "地址不可用"
			vmHint = "\n提示: 检查网络接口是否配置正确"
		} else {
			errMsg = err.Error()
			vmHint = "\n提示: 检查网络接口是否存在且已启用，在虚拟机中确认网络适配器设置"
		}
		return fmt.Errorf("绑定到网络接口 %s 失败: %s\n%s", interfaceName, errMsg, vmHint)
	}
	log.Error("已成功绑定到网络接口:", interfaceName)

	// 准备ARP数据包
	var packet arpPacket

	// 设置以太网头部
	// 目标MAC地址: 广播地址 (ff:ff:ff:ff:ff:ff)
	for i := 0; i < 6; i++ {
		packet.EthHdr[i] = 0xff
	}
	// 源MAC地址
	copy(packet.EthHdr[6:12], macAddr)
	// 以太网类型: ARP (0x0806)
	packet.EthHdr[12] = 0x08
	packet.EthHdr[13] = 0x06

	// 设置ARP头部
	// 硬件类型: 以太网 (1)
	packet.ARP_hdr[0] = 0x00
	packet.ARP_hdr[1] = 0x01
	// 协议类型: IPv4 (0x0800)
	packet.ARP_hdr[2] = 0x08
	packet.ARP_hdr[3] = 0x00
	// 硬件地址长度: 6
	packet.ARP_hdr[4] = 0x06
	// 协议地址长度: 4
	packet.ARP_hdr[5] = 0x04
	// 操作码: ARP请求 (1)
	packet.ARP_hdr[6] = 0x00
	packet.ARP_hdr[7] = 0x02
	// 源MAC地址
	copy(packet.ARP_hdr[8:14], macAddr)
	// 源IP地址 (VIP)
	copy(packet.ARP_hdr[14:18], vip4)
	// 目标MAC地址: 全0 (未知)
	for i := 18; i < 24; i++ {
		packet.ARP_hdr[i] = 0xff
	}
	// 目标IP地址 (VIP)
	//copy(packet.ARP_hdr[24:28], vip4)
	// 目标IP地址 (VIP)
	for i := 24; i < 28; i++ {
		packet.ARP_hdr[i] = 0x00
	}

	// 准备发送地址
	// 复用之前创建的addr变量
	// 设置目标MAC地址为广播地址
	// Addr字段是一个[8]byte数组，前6个字节用于MAC地址
	copy(addr.Addr[:6], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	// 初始化剩余的地址字节为0
	for i := 6; i < len(addr.Addr); i++ {
		addr.Addr[i] = 0
	}

	// 打印调试信息
	fmt.Printf("准备发送到网络接口 %s (索引: %d)\n", interfaceName, addr.Ifindex)
	fmt.Printf("调试: SockaddrLinklayer 详情:\n")
	fmt.Printf("  协议: 0x%x (ETH_P_ARP)\n", addr.Protocol)
	fmt.Printf("  接口索引: %d\n", addr.Ifindex)
	fmt.Printf("  目标MAC地址: %02x:%02x:%02x:%02x:%02x:%02x\n",
		addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3], addr.Addr[4], addr.Addr[5])

	// 平台特定的发送逻辑
	fmt.Printf("准备发送ARP数据包...\n")
	// 调试信息：打印数据包内容
	fmt.Printf("调试: 数据包长度: %d 字节\n", len(packet.EthHdr)+len(packet.ARP_hdr))
	fmt.Printf("调试: 源MAC: %v, 目标MAC: ff:ff:ff:ff:ff:ff\n", macAddr)
	fmt.Printf("调试: 源IP: %s, 目标IP: %s\n", vip, vip)

	// 发送ARP数据包
	log.Error("发送ARP数据包到广播地址 ff:ff:ff:ff:ff:ff...\n")

	for i := 0; i < 3; i++ {
		_, _, errno := syscall.Syscall6(uintptr(syscall.SYS_SENDTO),
			uintptr(fd),
			uintptr(unsafe.Pointer(&packet)),
			uintptr(len(packet.EthHdr)+len(packet.ARP_hdr)),
			uintptr(unsafe.Pointer(&addr)),
			0,
			0)
		if errno != 0 {
			// 详细的错误信息
			var errMsg string
			var vmHint string
			switch errno {
			case syscall.ENODEV:
				errMsg = "网络接口不存在或无效"
				vmHint = "\n提示1: 确认网络接口名称正确，使用'ip link show'命令查看\n提示2: 确保网络接口已启用，使用'sudo ifconfig <接口名> up'命令"
			case syscall.EADDRNOTAVAIL:
				errMsg = "地址不可用，请检查VIP是否有效且已绑定"
				vmHint = "\n提示1: 确认VIP地址格式正确且在网络接口的子网内\n提示2: 检查VIP是否已正确绑定，使用'ip addr show dev <接口名>'命令"
			case syscall.EPERM:
				errMsg = "权限不足，请以root权限运行"
				vmHint = "\n提示: 使用sudo命令以管理员权限运行程序"
			case 6:
				errMsg = "系统错误代码: 6 (no such device or address)"
				vmHint = "\n提示1: 这通常表示网络接口不存在或无法访问\n提示2: 在虚拟机中，确保网络适配器设置为桥接模式而非NAT模式\n提示3: 确认网络接口名称正确且已启用\n提示4: 检查虚拟机的网络连接是否正常"
			default:
				errMsg = fmt.Sprintf("系统错误代码: %d", errno)
				vmHint = "\n提示: 检查网络配置和权限设置"
			}
			return fmt.Errorf("发送ARP数据包失败: %s (%v)\n%s", errMsg, errno, vmHint)
		}

		log.Error("已发送ARP广播: ", vip)
	}
	return nil
}

func SetAutoStatus(newsta int, User string, Reason string) {

	log.Error("Internal SetAutoStatus:", newsta, " user:", User, " res:", Reason)

	stawithinsite.When = time.Now().Format("2006-01-02 15:04:05")
	stawithinsite.User = User
	stawithinsite.Reason = Reason

	if stawithinsite.Statu == newsta {
		log.Error("站点内操作自动动一个新的状态和旧状态一致 return:", newsta)
		return
	}
	stawithinsite.Statu = newsta

	if newsta == BACKUP {
		go InternalBackup()
	} else if newsta == MASTER {
		go InternalMaster()
	}
	return
}

func SetHandStatus(newsta int, User string, Reason string) {

	log.Error("Internal SetHandStatus:", newsta, " user:", User, " res:", Reason)

	stawithinsite.When = time.Now().Format("2006-01-02 15:04:05")
	stawithinsite.User = User
	stawithinsite.Reason = Reason

	if stawithinsite.Statu == newsta {
		log.Error("站点内操作手动一个新的状态和旧状态一致 return:", newsta)
		return
	}

	stawithinsite.Statu = newsta

	if newsta == BACKUP {
		stavrrp.SetMaster2Backup()
		go InternalBackup()
	} else if newsta == MASTER {
		stavrrp.SetBackup2Master()
		go InternalMaster()
	}

	stavrrp.SetPriority(byte(stawithinsite.Priority))
	stavrrp.SetPreempt(stawithinsite.Preeempt)
	return
}

func varinit() {
	stawithinsite.Nodepip = config.Instance.Internal.Iterpip
	stawithinsite.Routerid = config.Instance.Internal.Routerid
	stawithinsite.Priority = config.Instance.Internal.Priority
	stawithinsite.Preeempt = config.Instance.Internal.Preeempt
	stawithinsite.Netdev = config.Instance.Internal.Netdev
	stawithinsite.Itervip = config.Instance.Internal.Itervip
	stawithinsite.Maskbit = config.Instance.Internal.Maskbit
	SetAutoStatus(START, "init", "internal run start")
}

func PidDwonNetlindel() {
	netip := stawithinsite.Itervip + "/" + stawithinsite.Maskbit
	eth, _ := netlink.LinkByName(stawithinsite.Netdev)
	addr, _ := netlink.ParseAddr(netip)
	netlink.AddrDel(eth, addr)
}

func PidDwonNetliadd() {
	netip := stawithinsite.Itervip + "/" + stawithinsite.Maskbit
	eth, _ := netlink.LinkByName(stawithinsite.Netdev)
	addr, _ := netlink.ParseAddr(netip)
	netlink.AddrAdd(eth, addr)
	bindbuildGARPPacket(stawithinsite.Netdev, stawithinsite.Itervip)
}

func netlindel(reason string) {
	netip := stawithinsite.Itervip + "/" + stawithinsite.Maskbit
	eth, _ := netlink.LinkByName(stawithinsite.Netdev)
	addr, _ := netlink.ParseAddr(netip)
	netlink.AddrDel(eth, addr)
	SetAutoStatus(BACKUP, "vrrp", reason)
}

func netlinadd(reason string) {
	netip := stawithinsite.Itervip + "/" + stawithinsite.Maskbit
	eth, _ := netlink.LinkByName(stawithinsite.Netdev)
	addr, _ := netlink.ParseAddr(netip)
	netlink.AddrAdd(eth, addr)
	bindbuildGARPPacket(stawithinsite.Netdev, stawithinsite.Itervip)
	SetAutoStatus(MASTER, "vrrp", reason)
}

func vrrpinit() {
	stavrrp = vrrp.NewVirtualRouter(byte(stawithinsite.Routerid), stawithinsite.Netdev, false, vrrp.IPv4)

	stavrrp.SetPreemptMode(stawithinsite.Preeempt)
	if 0 == config.Instance.Internal.Testtime {
		stavrrp.SetAdvInterval(time.Millisecond * time.Duration(config.Instance.Internal.Interval))
	} else {
		stavrrp.SetAdvInterval(time.Millisecond * time.Duration(config.Instance.Internal.Testtime))
		log.Error("start vrrp test send time : ", config.Instance.Internal.Testtime)
	}

	stavrrp.SetPriorityAndMasterAdvInterval(byte(stawithinsite.Priority), time.Millisecond*time.Duration(config.Instance.Internal.Interval))

	ip := net.ParseIP(stawithinsite.Itervip)
	if ip == nil {
		log.Error("invalid IP address: ", stawithinsite.Itervip)
		return
	}
	ipv4 := ip.To16()
	if ipv4 == nil {
		log.Error("not an IPv4 address: ", stawithinsite.Itervip)
		return
	}
	stavrrp.AddIPvXAddr(ipv4)
}

func vrrprun() {
	stavrrp.Enroll(vrrp.Backup2Master, func() {
		netlinadd("vrrprun vrrp.Backup2Master, 挂载vip, 未收到vrrp协议网络包, 设定发送间隔时间: " + strconv.Itoa(config.Instance.Internal.Interval) + " ms")
	})
	stavrrp.Enroll(vrrp.Init2Master, func() {
		netlinadd("vrrprun vrrp.Init2Master, 挂载vip, 初始化")
	})
	stavrrp.Enroll(vrrp.Master2Init, func() {
		netlindel("vrrprun vrrp.Master2Init, 卸载vip, 下架节点")
	})
	stavrrp.Enroll(vrrp.Master2Backup, func() {
		netlindel("vrrprun vrrp.Master2Backup, 卸载vip, 其他节点未收到vrrp协议网络包, 已变成主, 设定发送间隔时间: " + strconv.Itoa(config.Instance.Internal.Interval) + " ms")
	})
	stavrrp.Enroll(vrrp.Init2Backup, func() {
		netlindel("vrrprun vrrp.Init2Backup, 卸载vip, 初始化")
	})
	stavrrp.Enroll(vrrp.Backup2Init, func() {
		netlindel("vrrprun vrrp.Backup2Init, 卸载vip, 下架节点")
	})
	go stavrrp.StartWithEventLoop()
}

func Init() {
	log.Error("Internal Init Run")
	varinit()
	vrrpinit()
	vrrprun()
}
