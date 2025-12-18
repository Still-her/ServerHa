package station

/*
#cgo CFLAGS:  -I../hacallback
#cgo LDFLAGS: -L../hacallback
#include "hacallback.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"ha/config"
	"ha/haping"
	"ha/internal"
	"ha/log"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/Still-her/calm"

	"github.com/go-resty/resty/v2"
)

const (
	START  int = 0
	BACKUP int = 1
	MASTER int = 2
	ERRSTA int = 3
)

// 并联切换模式
const (
	MODE_PARALLEL_AUTO  int = iota + 1 //并联网元VIP一个失效，自动切换为MASTER
	MODE_PARALLEL_ALRAM                //并联网元VIP一个失效，触发告警，人工切换
)

const (
	STATION_OK             int = 0  //成功
	STATION_NEGOTITAE_S    int = 7  //协商成功
	STATION_NEGOTITAE_F    int = 8  //协商失败
	STATION_NEGOTITAE_N    int = 9  //协商不响应
	STATION_MASTER_REQ_F   int = 11 //收到请求为主失败
	STATION_MASTER_REQ_N   int = 12 //收到请求为主不响应
	STATION_BACKUP_REQ_N   int = 15 //收到请求为备不响应
	STATION_VIP_DOWN_CHECK int = 17 //站点串联网元down，CHECK
	STATION_NODESTA_BACKUP int = 19 //站点是节点备状态
)

// 给网管返回站点的状态数据
type Betweensiteinfo struct {
	Statvip     string `gorm:"column:stationvip" json:"stationvip" form:"stationvip"`    //本站点vip
	Statu       int    `gorm:"column:statu" json:"statu" form:"statu"`                   //当前节点状态
	When        string `gorm:"column:when" json:"when" form:"when"`                      //上次切换时间
	User        string `gorm:"column:user" json:"user" form:"user"`                      //上次切换用户
	Reason      string `gorm:"column:reason" json:"reason" form:"reason"`                //上次切换原因
	Preeempt    bool   `gorm:"column:preeempt" json:"preeempt" form:"preeempt"`          //站点是否抢占模式
	Priority    int    `gorm:"column:priority" json:"priority" form:"priority"`          //站点的优先级
	Handmaster  int64  `gorm:"column:handmaster" json:"handmaster" form:"handmaster"`    //不独立时上次为主的时间戳
	Abimaster   int    `gorm:"column:abimaster" json:"abimaster" form:"abimaster"`       //站点具备主的能力值，0具备，其他值
	Independent bool   `gorm:"column:independent" json:"independent" form:"independent"` //站点是否独立
}

// 发送自己站点的优先级以及是否抢占模式
type StationHearbeat struct {
	Priority   int   `gorm:"column:priority" json:"priority" form:"priority"` //优先级 1-255
	Preeempt   bool  `gorm:"column:preeempt" json:"preeempt" form:"preeempt"` //是否抢占模式
	Pstatuss   int   `gorm:"column:pstatuss" json:"pstatuss" form:"pstatuss"` //自己当前是主是备
	Handmaster int64 `gorm:"column:pstatuss" json:"pstatuss" form:"pstatuss"` //不独立时上次为主的时间戳
}

type Seriesnestavipslice struct {
	Ip      string `gorm:"column:ip" json:"ip" form:"ip"`                //ip
	Netment string `gorm:"column:netment" json:"netment" form:"netment"` //网元名字
	Status  int    `gorm:"column:status" json:"status" form:"status"`    //状态
}

var (
	stadone          chan bool                                 //当备站点发送http hearbeat 通道
	stabetweensite   Betweensiteinfo                           //本站点信息
	preeemptint      int                                       //本站点是否抢占模式
	seriesnestavip   []Seriesnestavipslice                     //创建串联网元VIP集合
	parallelnestapip map[string]int                            //创建并联联网元物理IP集合
	parallelnestavip map[string]int                            //创建并联联网元VIP集合
	NetAddrList      []haping.IpaddrDetectmult                 //创建IP检验集合
	client           = resty.New().SetTimeout(3 * time.Second) //http 客户端
	stationlock      sync.Mutex                                //站点信息同步锁
)

func InitRun() {
	log.Error("Ha station InitRun")
	varinit()
}

func Negotitae() {

	log.Error("Ha station Negotitae")
	// 设置自定义的 Transport 以启用长连接
	client.SetTransport(&http.Transport{
		DisableKeepAlives: false,
	})
	starun()
}

func StationPing() {
	log.Error("Ha station StationPing")
	go haping.Ping(NetAddrList, PingCallBack)
}

func GetSeriesneStavip() []Seriesnestavipslice {
	return seriesnestavip
}

func Getbetweensitesta() Betweensiteinfo {
	return stabetweensite
}

func Getstatus() int {
	return stabetweensite.Statu
}

func Getabimaster() int {
	return stabetweensite.Abimaster
}

func GetPriority() int {
	return stabetweensite.Priority
}

func GetHandmaster() int64 {
	return stabetweensite.Handmaster
}

func SendNmsStationAlarmCode(alert int) {

	log.Error("Ha SendNmsStationAlarmCode:", alert)
	alertcode := C.int(alert)
	C.StationAlarmCodeCallback(alertcode)

}

func SendNmsStationAlarmMsg(alert string) {

	log.Error("Ha SendNmsStationAlarmMsg:", alert)

	alertmsg := C.CString(alert)
	C.StationAlarmMsgCallback(alertmsg)
	C.free(unsafe.Pointer(alertmsg))
}

func StationMaster() {
	log.Error("Ha StationMaster")
	C.StationMasterCallback()
}

func StationBackup() {
	log.Error("Ha StationBackup")
	C.StationBackupCallback()
}

func SetAutoStatus(newsta int, User string, Reason string) {

	if stabetweensite.Independent {
		log.Error("Station SetAutoStatus but this station is Independent")
		return
	}

	stationlock.Lock()
	defer stationlock.Unlock()

	if stabetweensite.Statu == newsta {

		stabetweensite.When = time.Now().Format("2006-01-02 15:04:05")
		stabetweensite.User = User
		stabetweensite.Reason = Reason
		stabetweensite.Statu = newsta
		return
	}

	log.Error("Station SetAutoStatus:", newsta, " user:", User, " res:", Reason)

	stabetweensite.When = time.Now().Format("2006-01-02 15:04:05")
	stabetweensite.User = User
	stabetweensite.Reason = Reason
	stabetweensite.Statu = newsta

	if newsta == BACKUP {
		go StationBackup()
	} else if newsta == MASTER {
		go StationMaster()
	}
	return
}

func SethandStatus(newsta int, User string, Reason string) (int, string) {

	if stabetweensite.Independent {
		log.Error("Station SethandStatus but this station is Independent")
		return -1, "Station SethandStatus but this station is Independent"
	}

	stationlock.Lock()
	defer stationlock.Unlock()

	// if newsta == MASTER {
	// 	if Getabimaster() != 0 {
	// 		log.Error("Station SethandStatus:", newsta, " failed:", "this station no abimaster")
	// 		return -1, "this station no abimaster"
	// 	}
	// }

	if stabetweensite.Statu == newsta {

		stabetweensite.When = time.Now().Format("2006-01-02 15:04:05")
		stabetweensite.User = User
		stabetweensite.Reason = Reason
		stabetweensite.Statu = newsta

		return 0, "ok"
	}

	log.Error("Station SethandStatus:", newsta, " user:", User, " res:", Reason)

	if newsta == MASTER {
		stationapplybackupreq()
	} else if newsta == BACKUP {
		stationapplymasterreq()
		// if stationapplymasterreq() != STATION_OK {
		// 	log.Error("Station SethandStatus:", newsta, " failed:", "other station no resp or no abimaster")
		// 	return -1, "other station no resp or no abimaster"
		// }
	}

	stabetweensite.When = time.Now().Format("2006-01-02 15:04:05")
	stabetweensite.User = User
	stabetweensite.Reason = Reason
	stabetweensite.Statu = newsta

	if newsta == BACKUP {
		go StationBackup()
	} else if newsta == MASTER {
		go StationMaster()
	}
	return 0, "ok"
}

func SetAloneStatus(newsta int, User string, Reason string) string {

	log.Error("Station SetAloneStatus:", newsta, " user:", User, " res:", Reason)
	stabetweensite.Independent = true
	if stabetweensite.Statu == newsta {
		log.Info("站点独立一个新的状态和旧状态一致 return:", newsta)
		return "this station already set status"
	}
	stabetweensite.When = time.Now().Format("2006-01-02 15:04:05")
	stabetweensite.User = User
	stabetweensite.Reason = Reason
	stabetweensite.Statu = newsta
	if newsta == BACKUP {
		go StationBackup()
	} else if newsta == MASTER {
		go StationMaster()
	}
	return "ok"
}

func SetAloneRemove() {
	log.Error("Station SetAloneRemove")
	stabetweensite.Independent = false
}

func varinit() {

	stabetweensite.Preeempt = config.Instance.Station.Preeempt
	stabetweensite.Priority = config.Instance.Station.Priority
	stabetweensite.Statvip = config.Instance.Station.Self.Vip
	stabetweensite.Abimaster = -1
	stabetweensite.Handmaster = 0
	stabetweensite.Independent = false

	if config.Instance.Station.Preeempt {
		preeemptint = 1
	} else {
		preeemptint = 0
	}
	//串联vip集合

	for _, value := range config.Instance.Seriesne.Vip {
		seriesnestavip = append(seriesnestavip, Seriesnestavipslice{Ip: value.Ip, Netment: value.Netment, Status: 0})
		NetAddrList = append(NetAddrList, haping.IpaddrDetectmult{Ip: value.Ip, Detectmult: config.Instance.Seriesne.Detectmult})
	}

	parallelnestapip = make(map[string]int)

	for _, value := range config.Instance.Station.Paralle.Pip {
		parallelnestapip[value.Ip] = 0
		NetAddrList = append(NetAddrList, haping.IpaddrDetectmult{Ip: value.Ip, Detectmult: config.Instance.Station.Paralle.Detectmult})
	}

	parallelnestavip = make(map[string]int)
	parallelnestavip[config.Instance.Station.Paralle.Vip] = 0

	NetAddrList = append(NetAddrList, haping.IpaddrDetectmult{Ip: config.Instance.Station.Paralle.Vip, Detectmult: config.Instance.Station.Paralle.Detectmult * 2})
}

func starun() {

	stationcheckabimaster()
	if 0 != stabetweensite.Abimaster {
		SetAutoStatus(BACKUP, "Abimaster", "starun this station no abimaster, change backup and do not req make master")
	} else {
		if 1 == stationcheckvolte() {
			res, msg := stationhttphearbeatreq()
			if res == STATION_OK {
				//if MASTER != Getstatus() {
				SetAutoStatus(MASTER, "starun", msg)
				//}
			} else if res == STATION_NEGOTITAE_F {
				//if MASTER == Getstatus() {
				SetAutoStatus(BACKUP, "starun", msg)
				//}
			} else if res == STATION_NEGOTITAE_N {
				SetAutoStatus(MASTER, "starun", msg)
			}
		} else {
			log.Error("starun volte status not ok,  do not req make master")
			SetAutoStatus(stabetweensite.Statu, "old", "starun do not req make master, old status, volte link not ok")
		}
	}

	stationbackupsyncmaster()
	go BackupSendHearbeat()
}

func BackupSendHearbeat() {

	var noresptimes int

	ticker := time.NewTicker(1 * time.Second)
	for {
		time.Sleep(time.Second)
		select {
		case <-ticker.C:

			stationcheckabimaster()
			if 0 != stabetweensite.Abimaster {
				SetAutoStatus(BACKUP, "Abimaster", "ticker this station no abimaster, change backup and do not req make master")
				continue
			}
			if 1 == stationcheckvolte() {
				res, msg := stationhttphearbeatreq()
				if res == STATION_OK {
					noresptimes = 0
					//if MASTER != Getstatus() {
					SetAutoStatus(MASTER, "hearbeatreq", msg)
					//}
				} else if res == STATION_NEGOTITAE_F {
					noresptimes = 0
					//if MASTER == Getstatus() {
					SetAutoStatus(BACKUP, "hearbeatreq", msg)
					//}
				} else if res == STATION_NEGOTITAE_N {

					noresptimes = noresptimes + 1

					if noresptimes >= config.Instance.Station.Paralle.Detectmult {

						log.Error(fmt.Sprintf("ticker stationhttphearbeatreq negotiate no resp times: %d", noresptimes))
						SethandStatus(MASTER, "hearbeatreq", fmt.Sprintf("ticker stationhttphearbeatreq negotiate no resp times: %d", noresptimes))
						noresptimes = 0
					} else {
						log.Error(fmt.Sprintf("ticker stationhttphearbeatreq negotiate no resp times: %d, no change status", noresptimes))
					}
				}
			} else {
				log.Error("ticker volte status not ok,  do not req make master")
				SetAutoStatus(stabetweensite.Statu, "old", "ticker do not req make master, old status, volte link not ok")
			}
			stationbackupsyncmaster()
		case <-stadone:
			log.Error("BackupSendHearbeat ticker.Stop return")
			ticker.Stop()
			return
		}
	}
}

func stationcheckabimaster() {
	if true == Getseriesnevipallstate() {
		stabetweensite.Abimaster = 0
	} else {
		stabetweensite.Abimaster = -1
	}
}

func stationcheckvolte() int {
	content, err := ioutil.ReadFile(".voltesta")
	if err != nil { // 处理读取文件错误
		return -1
	}
	if len(content) > 0 {
		num, err := strconv.Atoi(string(content[0]))
		if err != nil {
			return -2
		}
		return num
	}
	return -1
}

func stationhttphearbeatreq() (int, string) {

	stationlock.Lock()
	defer stationlock.Unlock()

	if internal.MASTER == internal.Getstatus() {

		ReqBaseUrl := "http://" + config.Instance.Station.Paralle.Vip + ":" + config.Instance.Httpport + "/run/hearbeat/negotiate"
		resp, err := client.R().
			SetQueryParam("priority", strconv.Itoa(config.Instance.Station.Priority)).
			SetQueryParam("preeempt", strconv.Itoa(preeemptint)).
			SetQueryParam("pstatuss", strconv.Itoa(Getstatus())).
			SetQueryParam("handmaster", strconv.FormatInt(stabetweensite.Handmaster, 10)).
			Get(ReqBaseUrl)
		if err != nil {
			log.Error("send master station http hearbeat no resp")
			return STATION_NEGOTITAE_N, "other station http hearbeat no resp"
		}
		result := &calm.JsonResult{}
		json.Unmarshal(resp.Body(), result)
		if result.Success {
			return STATION_OK, "other station http hearbeat negotiate ok"
		} else {
			return STATION_NEGOTITAE_F, result.Message
		}
	} else {
		return STATION_NODESTA_BACKUP, ""
	}
}

func stationbackupsyncmaster() {

	if internal.MASTER != internal.Getstatus() {

		ReqBaseUrl := "http://" + config.Instance.Station.Self.Sip + ":" + config.Instance.Httpport + "/nms/betweensite/statinfo"
		resp, err := client.R().
			Get(ReqBaseUrl)
		if err != nil {
			log.Error("send master internal sync to self no resp:", err.Error())
			return
		}
		result := &calm.JsonResult{}
		json.Unmarshal(resp.Body(), result)

		jsonData, err := json.Marshal(result.Data)
		if err != nil {
			log.Error("stationbackupsyncmaster:", err.Error())
			return
		}
		var sipinfo Betweensiteinfo
		json.Unmarshal(jsonData, &sipinfo)

		stabetweensite.Handmaster = sipinfo.Handmaster
		stabetweensite.When = sipinfo.When
		stabetweensite.User = sipinfo.User
		stabetweensite.Reason = sipinfo.Reason
		stabetweensite.Abimaster = sipinfo.Abimaster
		stabetweensite.Independent = sipinfo.Independent

		if stabetweensite.Statu != sipinfo.Statu {
			log.Error("station sync internal master status: ", sipinfo.Statu)
			stabetweensite.Statu = sipinfo.Statu
			if stabetweensite.Statu == BACKUP {
				log.Error("station sync internal master status: ", BACKUP, " user:", "internal", " res:", "this station internal status isnot master")
				go StationBackup()
			} else if stabetweensite.Statu == MASTER {
				log.Error("station sync internal master status: ", MASTER, " user:", "internal", " res:", "this station internal status isnot master")
				go StationMaster()
			}
		}
	}
}

func stationapplybackupreq() int {

	log.Info("stationapplybackupreq send master station make backup")

	ReqBaseUrl := "http://" + config.Instance.Station.Paralle.Vip + ":" + config.Instance.Httpport + "/run/backupreq/negotiate"
	log.Info("stationapplybackupreq:", ReqBaseUrl)

	_, err := client.R().
		Get(ReqBaseUrl)
	if err != nil {
		log.Error("send master station make backup no resp")
		return STATION_BACKUP_REQ_N
	}
	return STATION_OK
}

func stationapplymasterreq() int {

	log.Info("stationapplymasterreq send backup station make master")

	ReqBaseUrl := "http://" + config.Instance.Station.Paralle.Vip + ":" + config.Instance.Httpport + "/run/masterreq/negotiate"
	log.Info("stationapplymasterreq:", ReqBaseUrl)
	resp, err := client.R().
		Get(ReqBaseUrl)
	if err != nil {
		log.Error("send backup station make master no resp")
		return STATION_MASTER_REQ_N
	}
	result := &calm.JsonResult{}
	json.Unmarshal(resp.Body(), result)
	if result.Success {
		return STATION_OK
	} else {
		log.Error("send backup station make master no abimaster")
		return STATION_MASTER_REQ_F
	}
}

// 获取串联网元所有状态是否ok
func Getseriesnevipallstate() bool {
	for _, value := range seriesnestavip {
		log.Info("Getseriesnevipallstate ip: ", value.Ip, " Statu: ", value.Status)
		if value.Status == 0 {

			log.Error("NO  Abimaster Getseriesnevipallstate ip: ", value.Ip, " Statu: ", value.Status)

			return false
		}
	}
	return true
}

// 获取并联网元物理IP正常状态个数

func CheckParallePipStatus() {

	parallelneupnum := 0
	for key, value := range parallelnestapip {
		log.Info("CheckParallePipStatus ip: ", key, " statu: ", value)
		if value > 0 {
			parallelneupnum = parallelneupnum + 1
		}
	}

	if parallelneupnum == len(config.Instance.Station.Paralle.Pip) {

		SendNmsStationAlarmMsg("other station paralle vip down and pip all up, will try change make master")
		if Getabimaster() != 0 {
			log.Error("other station paralle vip down and pip all up, bus this station no abimaster")
		} else {
			SethandStatus(MASTER, "paralle", "other station paralle vip down and pip all up, hand master")
		}

		// switch config.Instance.Station.Paralle.Allupmode {
		// case MODE_PARALLEL_AUTO:
		// 	{
		// 		if Getabimaster() != 0 {
		// 			log.Error("other station paralle vip down and pip all up, bus this station no abimaster")
		// 		} else {
		// 			SetAutoStatus(MASTER, "paralle", "other station paralle vip down and pip all up")
		// 		}
		// 		break
		// 	}
		// case MODE_PARALLEL_ALRAM:
		// 	{
		// 		SendNmsStationAlarmMsg("other station paralle vip down and pip all up alarm")
		// 		break
		// 	}
		// }

	} else if parallelneupnum > 0 {

		SendNmsStationAlarmMsg("other station paralle vip down and pip least one up, will try change make master")

		if Getabimaster() != 0 {
			log.Error("other station paralle vip down and pip least one up , bus this station no abimaster")
		} else {
			SethandStatus(MASTER, "paralle", "other station paralle vip down and pip least one up, hand master")
		}

		// switch config.Instance.Station.Paralle.Oneupmode {
		// case MODE_PARALLEL_AUTO:
		// 	{
		// 		if Getabimaster() != 0 {
		// 			log.Error("other station paralle vip down and pip least one up , bus this station no abimaster")
		// 		} else {
		// 			SetAutoStatus(MASTER, "paralle", "other station paralle vip down and pip least one up")
		// 		}
		// 		break
		// 	}
		// case MODE_PARALLEL_ALRAM:
		// 	{
		// 		SendNmsStationAlarmMsg("other station paralle vip down and pip least one up")
		// 		break
		// 	}
		// }
	} else {

		SendNmsStationAlarmMsg("other station paralle vip down and pip all down, 请查看现场站点, 如有异常, 可以尝试站点切换")

		// switch config.Instance.Station.Paralle.Zeroupmode {
		// case MODE_PARALLEL_AUTO:
		// 	{
		// 		if Getabimaster() != 0 {
		// 			log.Error("other station paralle vip down and pip all down, bus this station no abimaster")
		// 		} else {
		// 			SetAutoStatus(MASTER, "paralle", "other station paralle vip down and pip all down")
		// 		}
		// 		break
		// 	}
		// case MODE_PARALLEL_ALRAM:
		// 	{
		// 		SendNmsStationAlarmMsg("other station paralle vip down and pip all down")
		// 		break
		// 	}
		// }
	}

}

func PingCallBack(ipAddr string, pingrecv int) {

	for key, _ := range parallelnestapip {
		if key == ipAddr {
			log.Info("PingCallBack parallelnestapip ip: ", ipAddr, " recv: ", pingrecv)
			parallelnestapip[key] = pingrecv
		}
	}

	for index, value := range seriesnestavip {
		if value.Ip == ipAddr {
			log.Info("PingCallBack seriesnestavip ip: ", ipAddr, " recv: ", pingrecv)
			seriesnestavip[index].Status = pingrecv
		}
	}

	for key, _ := range parallelnestavip {
		if key == ipAddr {
			log.Info("PingCallBack parallelnestavip ip: ", ipAddr, " recv: ", pingrecv)
			parallelnestavip[key] = pingrecv
			// if Getstatus() == MASTER {
			// 	log.Info("PingCallBack parallelnestavip this station is MASTER return")
			// 	return
			// } else {
			// 	log.Info("PingCallBack parallelnestavip this station is BACKUP")
			// }

			//只要对端的vip失效
			if 0 == pingrecv {
				if internal.MASTER == internal.Getstatus() {
					log.Error("PingCallBack this station node status is master CheckParallePipStatus")
					CheckParallePipStatus()
				}
			}
		}
	}

	return
}
