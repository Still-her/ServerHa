package nms

import (
	"ha/internal"
	"ha/log"
	"ha/station"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/Still-her/calm"

	"github.com/kataras/iris/v12"
)

// 网管站点内控制块
type NmsInsiteController struct {
	Ctx iris.Context
}

// 网管站点间控制块
type NmsBetweenController struct {
	Ctx iris.Context
}

// 网管一键请求控制块
type NmsTouchController struct {
	Ctx iris.Context
}

// @Tags  		NMS
// @Summary     一键请求使此站点独立且成为主
// @Produce  	json
// @Router 		/nms/touchreq/alonemaster [post]
func (c *NmsTouchController) PostAlonemaster() *calm.JsonResult {

	log.Error(time.Now(), ":NMS 一键请求使此站点独立且成为主")

	msg := station.SetAloneStatus(station.MASTER, "touchreq", "/nms/touchreq/alonemaster")

	return calm.JsonErrorMsg(msg)
}

// @Tags  		NMS
// @Summary     一键请求使此站点独立且成为备
// @Produce  	json
// @Router 		/nms/touchreq/alonebackup [post]
func (c *NmsTouchController) PostAlonebackup() *calm.JsonResult {

	log.Error(time.Now(), ":NMS 一键请求使此站点独立且成为备")

	msg := station.SetAloneStatus(station.BACKUP, "touchreq", "/nms/touchreq/alonebackup")

	return calm.JsonErrorMsg(msg)
}

// @Tags  		NMS
// @Summary     一键请求使此站点独立取消
// @Produce  	json
// @Router 		/nms/touchreq/aloneremove [post]
func (c *NmsTouchController) PostAloneremove() *calm.JsonResult {
	station.SetAloneRemove()
	return calm.JsonSuccess()
}

// @Tags  		NMS
// @Summary     一键请求使此站点节点成为主
// @Produce  	json
// @Router 		/nms/touchreq/masterreq [post]
func (c *NmsTouchController) PostMasterreq() *calm.JsonResult {

	log.Error(time.Now(), ":NMS 一键请求使此站点节点成为主")
	if station.MASTER != station.Getstatus() {
		res, msg := station.SethandStatus(station.MASTER, "touchreq", "/nms/touchreq/masterreq")
		if res == 0 {
			log.Error(time.Now(), ":NMS 一键请求使此站点节点成为主 站点已经OK")

		} else {
			log.Error(time.Now(), ":NMS 一键请求使此站点节点成为主 站点已经FA")
			return calm.JsonErrorMsg(msg)
		}
	}

	if internal.MASTER != internal.Getstatus() {
		internal.SetHandStatus(internal.MASTER, "touchreq", "/nms/touchreq/masterreq")
		log.Error(time.Now(), ":NMS 一键请求使此站点节点成为主 节点已经OK")
	}
	return calm.JsonSuccess()
}

// @Tags  		NMS
// @Summary     一键请求使此站点节点信息
// @Produce  	json
// @Router 		/nms/touchreq/nodeinfo [get]
func (c *NmsTouchController) GetNodeinfo() *calm.JsonResult {

	log.Info(time.Now(), ":NMS 一键请求使此站点节点信息")
	return calm.NewEmptyRspBuilder().
		Put("seriesne", station.GetSeriesneStavip()).
		Put("station", station.Getbetweensitesta()).
		Put("internal", internal.Getstawithinsitesta()).
		JsonResult()
}

// @Tags  		NMS
// @Summary     一键请求使此站点节点VOLTE状态
// @Produce  	json
// @Router 		/nms/touchreq/voltesta [get]
func (c *NmsTouchController) GetVoltesta() *calm.JsonResult {

	log.Info(time.Now(), ":NMS 一键请求使此站点节点VOLTE状态")

	content, err := ioutil.ReadFile(".voltesta")
	if err != nil { // 处理读取文件错误
		return calm.JsonErrorCode(-1)
	}
	num, err := strconv.Atoi(string(content[0]))
	if err != nil {
		return calm.JsonErrorCode(-2)
	}
	return calm.JsonErrorCode(num)
}

// @Tags  		NMS
// @Summary     获取站点内主节点信息
// @Produce  	json
// @Success     200      {object}    internal.Withinsiteinfo "当前站点内主节点信息"
// @Router 		/nms/withinsite/nodeinfo [get]
func (c *NmsInsiteController) GetNodeinfo() *calm.JsonResult {
	log.Info(time.Now(), ":NMS 获取站点内主节点信息")
	return calm.JsonData(internal.Getstawithinsitesta())
}

// @Tags  		NMS
// @Summary     获取当前站点信息
// @Produce  	json
// @Success     200      {object}    station.Betweensiteinfo "当前站点信息"
// @Router 		/nms/betweensite/statinfo [get]
func (c *NmsBetweenController) GetStatinfo() *calm.JsonResult {
	log.Info(time.Now(), ":NMS 获取当前站点信息")
	return calm.JsonData(station.Getbetweensitesta())
}

// @Tags  		NMS
// @Summary     获取当前站点同级网元状态
// @Produce  	json
// @Success     200      {object}    station.Seriesnestavipslice "当前站点同级网元状态"
// @Router 		/nms/betweensite/seriesne [get]
func (c *NmsBetweenController) GetSeriesne() *calm.JsonResult {
	log.Info(time.Now(), ":NMS 获取当前站点同级网元状态")
	return calm.JsonData(station.GetSeriesneStavip())
}

// @Tags  		NMS
// @Summary     请求使站点内某一节点成为主
// @Produce  	json
// @Router 		/nms/withinsite/masterreq [post]
func (c *NmsInsiteController) PostMasterreq() *calm.JsonResult {
	log.Error(time.Now(), ":NMS 请求使站点内某一节点成为主")

	if internal.MASTER != internal.Getstatus() {
		internal.SetHandStatus(internal.MASTER, "nms", "/nms/withinsite/masterreq")
		return calm.JsonSuccess()
	} else {
		return calm.JsonErrorMsg("this internal already master")
	}
}

// @Tags  		NMS
// @Summary     请求使站点内某一节点成为备
// @Produce  	json
// @Router 		/nms/withinsite/backupreq [post]
func (c *NmsInsiteController) PostBackupreq() *calm.JsonResult {
	log.Error(time.Now(), ":NMS 请求使站点内某一节点成为备")
	internal.SetHandStatus(internal.BACKUP, "nms", "/nms/withinsite/backupreq")
	return calm.JsonSuccess()
}

// @Tags  		NMS
// @Summary     请求使此站点成为主
// @Produce  	json
// @Router 		/nms/betweensite/masterreq [post]
func (c *NmsBetweenController) PostMasterreq() *calm.JsonResult {
	log.Error(time.Now(), ":NMS 请求使此站点成为主")
	if station.MASTER != station.Getstatus() {
		res, msg := station.SethandStatus(station.MASTER, "nms", "/nms/betweensite/masterreq")
		if res == 0 {
			return calm.JsonErrorCode(res)
		}
		return calm.JsonErrorMsg(msg)
	} else {
		return calm.JsonErrorMsg("this station already master")
	}
}

// @Tags  		NMS
// @Summary     请求使此站点成为备
// @Produce  	json
// @Router 		/nms/betweensite/backupreq [post]
func (c *NmsBetweenController) PostBackupreq() *calm.JsonResult {
	log.Error(time.Now(), ":NMS 请求使此站点成为备")
	if station.BACKUP != station.Getstatus() {
		res, msg := station.SethandStatus(station.BACKUP, "nms", "/nms/betweensite/backupreq")
		if res == 0 {
			return calm.JsonErrorCode(res)
		}
		return calm.JsonErrorMsg(msg)
	} else {
		return calm.JsonErrorMsg("this station already backup")
	}
}
