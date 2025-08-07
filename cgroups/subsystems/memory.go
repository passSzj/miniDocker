package subsystems

import (
	"fmt"
	"miniDocker/constant"
	"os"
	"path"
	"strconv"
)

type MemorySubSystem struct{}

func (s *MemorySubSystem) Name() string {
	return "memory"
}

// Set 设置cgroupPath对应的cgroup内存限制
func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if res.MemoryLimit == "" {
		return nil
	}
	//获取memory的cgroup路径
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	//写入限制
	if err := os.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup memory fail %v", err)
	}
	return nil
}

// Apply 将pid加入到对应cgroupPath对应的cgroup中
func (s *MemorySubSystem) Apply(cgroupPath string, pid int, res *ResourceConfig) error {
	if res.MemoryLimit == "" {
		return nil
	}
	//获取memory的cgroup路径
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return fmt.Errorf("%v fail get cgroup: %s", err, cgroupPath)
	}
	if err := os.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}

	return nil
}

// Remove 删除cgroupPath对应的cgroup
func (s *MemorySubSystem) Remove(cgroupPath string) error {
	//获取memory的cgroup路径
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	//删除cgroup目录
	return os.RemoveAll(subsysCgroupPath)
}
