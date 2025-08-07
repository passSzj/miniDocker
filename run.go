package main

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"miniDocker/container"
)

// Run 执行具体 command
/*
这里的Start方法是真正开始执行由NewParentProcess构建好的command的调用，它首先会clone出来一个namespace隔离的
进程，然后在子进程中，调用/proc/self/exe,也就是调用自己，发送init参数，调用我们写的init方法，
去初始化容器的一些资源。
*/
func Run(tty bool, comArray []string) {
	parent, wirtePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Error("NewParentProcess error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	sendInitCommand(comArray, wirtePipe) //创建子进程后传递参数
	_ = parent.Wait()

}

// sendInitCommand 通过writePipe将命令发到子进程
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close() // 关闭管道，通知子进程
}
