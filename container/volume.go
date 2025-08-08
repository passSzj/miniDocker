package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"miniDocker/constant"
	"os"
	"os/exec"
	"path"
	"strings"
)

// mountVolume 挂载Volume目录     三个参数地址分别是   容器在宿主机的地址 mmtPath，宿主机的地址hostPath，容器内的地址containerPath
func mountVolume(mmtPath, hostPath, containerPath string) {
	//创建宿主机目录  作为挂载source
	if err := os.MkdirAll(hostPath, constant.Perm0777); err != nil {
		log.Infof("mkdir hostPath %s error: %v", hostPath, err)
	}

	//拼接容器内的挂载路径 在容器的 rootfs（mntPath）下创建挂载目标目录（即容器内路径）  这个路径是基于宿主机的地址拼接的  实际上也是在宿主机上创建文件夹  但是是在内容内部使用
	containerPathInHost := path.Join(mmtPath, containerPath)
	if err := os.Mkdir(containerPathInHost, constant.Perm0777); err != nil {
		log.Infof("mkdir containerPath %s error: %v", containerPathInHost, err)
	}

	//将容器外地址挂载到容器内目录上  实现数据持久化
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount volume error: %v", err)
	}
}

// umountVolume 卸载Volume目录
func umountVolume(mntPath, containerPath string) {
	// 拼接容器内的挂载路径
	containerPathInHost := path.Join(mntPath, containerPath)

	// 卸载容器内的挂载目录
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount volume error: %v", err)
	}
}

// volumeExtract 提取Volume的源路径和目标路径
func volumeExtract(volume string) (sourcePath, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume [%s], must split by `:`", volume)
	}

	sourcePath, destinationPath = parts[0], parts[1]
	if sourcePath == "" || destinationPath == "" {
		return "", "", fmt.Errorf("invalid volume [%s], path can't be empty", volume)
	}

	return sourcePath, destinationPath, nil
}
