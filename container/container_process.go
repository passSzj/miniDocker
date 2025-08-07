package container

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

// NewParentProcess 构建 command 用于启动一个新进程
/*
这里是父进程，也就是当前进程执行的内容。
1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		log.Errorf("Create pipe error: %v", err)
		return nil, nil
	}

	cmd := exec.Command("/proc/self/exe", "init") //  /proc/self/exe 是 Linux 系统中的一个符号链接，它指向当前进程的可执行文件。
	/*
		init中这里自己运行自己的目的：
		第一次是 控制器：启动容器
		第二次是 容器内的执行器：初始化并运行容器命令
		只有“新的进程”才能使用新的 namespace 和 cgroup，而我们希望容器内的进程和主机隔离，所以必须 创建一个新进程，并设置隔离环境，这就必须“再运行一次自己”。
	*/
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles = []*os.File{readPipe} // 将读取方转入子进程
	return cmd, writePipe
}
