package notify

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

func Init() {
	// 1、NewWatcher 初始化一个 watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
	}
	defer watcher.Close()

	//3、创建新的 goroutine，等待管道中的事件或错误
	done := make(chan bool)
	go func() {
		for {
			select {
			case e, ok := <-watcher.Events:
				if !ok {
					return
				}
				fmt.Printf("监听到文件 %s 变化| ", e.Name)
				switch e.Op {
				case fsnotify.Create:
					fmt.Println("创建事件", e.Op)
				case fsnotify.Write:
					fmt.Println("写入事件", e.Op)
				case fsnotify.Remove:
					fmt.Println("删除事件", e.Op)
				case fsnotify.Rename:
					fmt.Println("重命名事件", e.Op)
				case fsnotify.Chmod:
					fmt.Println("属性修改事件", e.Op)
				default:
					fmt.Println("some thing else")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("error:", err)
			}
		}
	}()
	// 2、使用 watcher 的 Add 方法增加需要监听的文件或目录到监听队列中
	err = watcher.Add("./")
	if err != nil {
		fmt.Println(err)
	}
	<-done
}
