package run

import (
	"ha/log"
	"ha/station"
	"strconv"
	"time"

	"github.com/Still-her/calm"

	"github.com/kataras/iris/v12"
)

// 网管站点内控制块
type RunHearbeatController struct {
	Ctx iris.Context
}

// 网管站点内控制块
type RunMasterreqController struct {
	Ctx iris.Context
}

// 网管站点内控制块
type RunBackupreqController struct {
	Ctx iris.Context
}

// @Tags  		RUN
// @Summary     站点间心跳加协商主备关系
// @Accept 		json
// @Param    	priority   	query   string   true     "优先级"
// @Param    	preeempt   	query   string   true     "是否抢占"
// @Param    	pstatuss   	query   string   true     "当前状态"
// @Produce  	json
// @Router 		/run/hearbeat/negotiate [get]
func (c *RunHearbeatController) GetNegotiate() *calm.JsonResult {

	var hearbeat station.StationHearbeat
	hearbeat.Priority, _ = strconv.Atoi(c.Ctx.FormValue("priority"))
	preeempt, _ := strconv.Atoi(c.Ctx.FormValue("preeempt"))
	hearbeat.Pstatuss, _ = strconv.Atoi(c.Ctx.FormValue("pstatuss"))
	hearbeat.Handmaster, _ = strconv.ParseInt(c.Ctx.FormValue("handmaster"), 10, 64)

	if preeempt == 1 {
		hearbeat.Preeempt = true
	} else {
		hearbeat.Preeempt = false
	}

	// if station.Getabimaster() != 0 {
	// 	//if station.BACKUP != station.Getstatus() {
	// 	station.SetAutoStatus(station.BACKUP, "hearbeatrsp", "negotiate this station no abimaster")
	// 	//}
	// 	return calm.JsonSuccess()
	// }

	if station.BACKUP == station.Getstatus() {
		return calm.JsonSuccess()
	}

	if hearbeat.Priority > station.GetPriority() && hearbeat.Pstatuss == station.MASTER {
		station.SetAutoStatus(station.BACKUP, "hearbeatrsp", "negotiate other station is priority")
		return calm.JsonSuccess()
	} else if hearbeat.Priority > station.GetPriority() && hearbeat.Preeempt {
		station.SetAutoStatus(station.BACKUP, "hearbeatrsp", "negotiate other station is preeempt")
		return calm.JsonSuccess()
	} else {
		return calm.JsonErrorMsg("hearbeatrsp negotiate station priority height, make backup")
	}

	//station.SetAutoStatus(station.MASTER, "hearbeatrsp", "negotiate other station is backup")
	//return calm.JsonErrorMsg("hearbeatrsp negotiate station disagree, make backup")
}

// @Tags  		RUN
// @Summary     申请本站点为主
// @Accept 		multipart/form-data
// @Produce  	json
// @Router 		/run/masterreq/negotiate [get]
func (c *RunMasterreqController) GetNegotiate() *calm.JsonResult {

	log.Error(time.Now(), ":RUN 申请本站点为主")

	// if station.Getabimaster() != 0 {
	// 	return calm.JsonErrorMsg("negotiate other station no abimaster")
	// }

	station.SetAutoStatus(station.MASTER, "negotiate", "negotiate from other station is req this station make master")
	return calm.JsonSuccess()
}

// @Tags  		RUN
// @Summary     申请本站点为备
// @Accept 		multipart/form-data
// @Produce  	json
// @Router 		/run/backupreq/negotiate [get]
func (c *RunBackupreqController) GetNegotiate() *calm.JsonResult {

	log.Error(time.Now(), ":RUN 申请本站点为备")
	station.SetAutoStatus(station.BACKUP, "negotiate", "negotiate from other station is req this station make backup")
	return calm.JsonSuccess()
}
