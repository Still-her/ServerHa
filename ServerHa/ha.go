package main

/*
#cgo CFLAGS:  -I./hacallback
#cgo LDFLAGS: -L./hacallback
#include "hacallback.h"
#include <stdlib.h>
#include <stdio.h>
*/
import "C"
import (
	"fmt"
	"ha/api/router"
	"ha/internal"
	"ha/log"
	"ha/notify"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	//"ha/log"
	"ha/station"
)

func init() {
	data := "Hello, still-her, this ha is v1.0\n"
	os.WriteFile("ha.version", []byte(data), 0644)
}

//export Handlesignal
func Handlesignal() {
	go func() {
		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			fmt.Println("handlesignal recv signal: ", sig)
			done <- true
			//internal.PidDwonNetlindel()
		}()
		<-done
		os.Exit(0)
	}()
}

//export HaRuntime
func HaRuntime() {
	C.test()
	runtime.GOMAXPROCS(runtime.NumCPU())
}

//export HaLoginit
func HaLoginit() {
	log.Init()
}

//export HaInternal
func HaInternal() {
	log.Error(time.Now(), ": HaInternal")
	internal.Init()
}

//export HaStaInitRun
func HaStaInitRun() {
	log.Error(time.Now(), ": HaStaInitRun")
	station.InitRun()
}

//export HaStaNegotitae
func HaStaNegotitae() {
	log.Error(time.Now(), ": HaStaNegotitae")
	station.Negotitae()
}

//export HaStaStationPing
func HaStaStationPing() {
	log.Error(time.Now(), ": HaStaStationPing")
	station.StationPing()
}

//export HaRouter
func HaRouter() {
	log.Error(time.Now(), ": HaRouter")
	router.Run()
}

//export ChangeStaStatus
func ChangeStaStatus(Newsta int, User string, Reason int) {
	log.Error(time.Now(), ": ChangeStaStatus", " User: ", User, " Reason: ", Reason)
	station.SetAutoStatus(Newsta, User, "ChangeStaStatus")
}

//export ChangeIntStatus
func ChangeIntStatus(Newsta int, User string, Reason int) {
	log.Error(time.Now(), ": ChangeIntStatus", " User: ", User, " Reason: ", Reason)
	internal.SetHandStatus(Newsta, User, "ChangeIntStatus")
}

//export GetVip
func GetVip() string {
	log.Error(time.Now(), ": ha GetVip")
	return internal.GetVip()
}

//export Notify
func Notify() {
	log.Error(time.Now(), ": ha Notify")
	go notify.Init()
}

//ip addr del 192.168.92.155/24 dev ens33
//ip addr add 192.168.92.155/24 dev ens33

//swag init -g ha.go
//go build -buildmode=c-shared -o libha.so ha.go
//go build -buildmode=c-archive -o libha.a ha.go

//LD_LIBRARY_PATH=./hacallback
//export LD_LIBRARY_PATH

// @version 1.0
// @title ha example api
// @contact.name LiXingbo
// @contact.email wy18565760326@163.com
// @description 花会沿路盛开,你以后的路也是
// @host 10.9.14.32:9999
// @BasePath /
func main() {

	//go handleSignal()

	// HaRuntime()
	// HaLoginit()
	// HaInternal()
	// HaStaInitRun()
	// HaStaNegotitae()
	// HaStaStationPing()
	// HaRouter()

	// for {
	// 	//time.Sleep(time.Duration1 * time.Microsecond)
	// 	time.Sleep(time.Duration(1) * time.Second)
	// }

}
