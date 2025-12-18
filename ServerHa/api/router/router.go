package router

import (
	"ha/api/nms"
	"ha/api/run"
	"ha/config"
	_ "ha/docs"
	"net/http"

	"github.com/iris-contrib/swagger"
	"github.com/iris-contrib/swagger/swaggerFiles"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func Run() {
	app := newApp()
	useswag(app)
	mvcHandle(app)
	appRun(app)
}

func newApp() *iris.Application {
	app := iris.New()
	//设置日志级别  开发阶段为debug
	app.Logger().SetLevel("error")
	return app
}

func useswag(app *iris.Application) {
	swaggerurl := "http://" + config.Instance.Station.Self.Pip + ":" + config.Instance.Httpport + "/still-her/swagger.json"
	swaggerUI := swagger.Handler(swaggerFiles.Handler,
		swagger.URL(swaggerurl),
		swagger.DeepLinking(true),
		swagger.Prefix("/still-her"),
	)
	// Register on http://127.0.0.1:9999/swagger
	app.Get("/still-her", swaggerUI)
	// And http://127.0.0.1:9999/swagger/index.html, *.js, *.css and e.t.c.
	app.Get("/still-her/{any:path}", swaggerUI)
}

/*
站点间自动：
启动根据优先级和抢占模式决定谁是主是备，自己配
当站点间的主备发生变化，需要将信息同步给站点的其他节点，请求物理http请求
*/
func mvcHandle(app *iris.Application) {

	mvc.Configure(app.Party("/nms"), func(m *mvc.Application) {
		m.Party("/withinsite").Handle(new(nms.NmsInsiteController))
		m.Party("/betweensite").Handle(new(nms.NmsBetweenController))
		m.Party("/touchreq").Handle(new(nms.NmsTouchController))
	})
	mvc.Configure(app.Party("/run"), func(m *mvc.Application) {
		m.Party("/hearbeat").Handle(new(run.RunHearbeatController))
		m.Party("/masterreq").Handle(new(run.RunMasterreqController))
		m.Party("/backupreq").Handle(new(run.RunBackupreqController))
	})
}

func appRun(app *iris.Application) {
	server := &http.Server{Addr: ":" + config.Instance.Httpport}
	//server.ListenAndServeTLS("server.crt", "server.key")
	go app.Run(iris.Server(server))
}
