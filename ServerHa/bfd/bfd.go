package bfd

import (
	"fmt"
	"ha/config"
	"syscall"
)

// 定义回调函数
/*
 * 目标ip, 上一个状态, 当前状态
 */
func callBackBFDState(ipAddr string, preState, curState int) error {
	fmt.Println("ipAddr:", ipAddr, ",preState:", preState, ",curState:", curState)
	return nil
}

func test() {
	family := syscall.AF_INET // 默认ipv4
	local := "0.0.0.0"
	port := 3666

	passive := false                                  // 是否是被动模式
	rxInterval := config.Instance.Seriesne.Rtinterval // 接收速率400 毫秒
	txInterval := config.Instance.Seriesne.Rtinterval // 发送速率400 毫秒
	detectMult := config.Instance.Seriesne.Detectmult // 报文最大失效的个数

	fmt.Println("rxInterval:", rxInterval)
	fmt.Println("txInterval:", txInterval)
	fmt.Println("detectMult:", detectMult)

	// 启动
	control := NewControl(port, local, family)
	for _, value := range config.Instance.Seriesne.Vip {

		control.AddSession(value.Ip, passive, rxInterval, txInterval, detectMult, callBackBFDState)
		control.DelSession(value.Ip)
	}
	fmt.Println("seriesnestatus_vip end")

}
