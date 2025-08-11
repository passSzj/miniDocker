package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"miniDocker/constant"
	"os"
	"os/exec"
	"path"
	"syscall"
)

const (
	RUNNING       = "running"
	STOP          = "stopped"
	Exit          = "exited"
	InfoLoc       = "/var/lib/miniDocker/containers/"
	InfoLocFormat = InfoLoc + "%s/"
	ConfigName    = "config.json"
	IDLength      = 10

	LogFile = "%s-json.log" // 日志文件名格式
)

type Info struct {
	Pid         string `json:"pid"`        // 容器的init进程在宿主机上的 PID
	Id          string `json:"id"`         // 容器Id
	Name        string `json:"name"`       // 容器名
	Command     string `json:"command"`    // 容器内init运行命令
	CreatedTime string `json:"createTime"` // 创建时间
	Status      string `json:"status"`     // 容器的状态
}

// NewParentProcess 构建 command 用于启动一个新进程
/*
这里是父进程，也就是当前进程执行的内容。
1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool, volume string, containerId string) (*exec.Cmd, *os.File) {

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
	} else {
		//后台运行   将标准输入输出重定向到日志文件
		dirPath := fmt.Sprintf(InfoLocFormat, containerId)
		if err := os.MkdirAll(dirPath, constant.Perm0622); err != nil {
			log.Errorf("mkdir %s error: %v", dirPath, err)
			return nil, nil
		}
		stdLogFilePath := dirPath + GetLogfile(containerId)
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			log.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
		cmd.Stderr = stdLogFile
	}
	cmd.ExtraFiles = []*os.File{readPipe} // 将读取方转入子进程
	rootPath := "/root"
	NewWorkSpace(rootPath, volume) // 设置工作目录为 /root/busybox
	cmd.Dir = path.Join(rootPath, "merged")
	return cmd, writePipe
}

// NewWorkSpace Create an Overlay2 filesystem as container root workspace
func NewWorkSpace(rootPath string, volume string) {
	createLower(rootPath)
	createDirs(rootPath)
	mountOverlayFS(rootPath)

	// 如果有volume挂载，则需要将volume挂载到merged目录下
	if volume != "" {
		mntPath := path.Join(rootPath, "merged") // 容器的在宿主机的实际目录
		hostPath, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("volumeExtract error: %v", err)
			return
		}
		// 挂载volume到merged目录下
		mountVolume(mntPath, hostPath, containerPath)
	}
}

// createLower 将busybox作为overlayfs的lower层
func createLower(rootPath string) {
	// 把busybox作为overlayfs中的lower层
	busyboxPath := path.Join(rootPath, "busybox")
	busyboxTarPath := path.Join(rootPath, "busybox.tar")
	log.Infof("busybox:%s busybox.tar:%s", busyboxPath, busyboxTarPath)
	// 检查是否已经存在busybox文件夹
	exist, err := PathExists(busyboxPath)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxPath, err)
	}
	// 不存在则创建目录并将busybox.tar解压到busybox文件夹中
	if !exist {
		if err = os.Mkdir(busyboxPath, 0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxPath, err)
		}
		if _, err = exec.Command("tar", "-xvf", busyboxTarPath, "-C", busyboxPath).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", busyboxPath, err)
		}
	}
}

// createDirs 创建overlayfs需要的的merged、upper、worker目录
func createDirs(rootPath string) {
	dirs := []string{
		path.Join(rootPath, "merged"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"),
	}

	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0777); err != nil {
			log.Errorf("mkdir dir %s error. %v", dir, err)
		}
	}
}

// mountOverlayFS 挂载overlayfs
func mountOverlayFS(rootPath string) {
	// 拼接参数
	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work
	dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", path.Join(rootPath, "busybox"),
		path.Join(rootPath, "upper"), path.Join(rootPath, "work"))

	// 完整命令：mount -t overlay overlay -o lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work /root/merged
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, path.Join(rootPath, "merged"))
	log.Infof("mount overlayfs: [%s]", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

// DeleteWorkSpace Delete the AUFS filesystem while container exit
func DeleteWorkSpace(rootPath string, volume string) {
	mntPath := path.Join(rootPath, "merged") //容器的在宿主机的实际目录

	//umount volume
	//先umount volume   如果先把overlayfs卸载了，导致挂载在容器外的数据会被删除
	if volume != "" {
		_, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("volumeExtract error: %v", err)
			return
		}
		umountVolume(mntPath, containerPath)
	}

	//解除挂载
	umountOverlayFS(mntPath)
	deleteDirs(rootPath)
}

func umountOverlayFS(mntPath string) {
	cmd := exec.Command("umount", mntPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

func deleteDirs(rootPath string) {
	dirs := []string{
		path.Join(rootPath, "merged"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"),
	}

	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Errorf("Remove dir %s error %v", dir, err)
		}
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetLogfile(containerId string) string {
	return fmt.Sprintf(LogFile, containerId)
}
