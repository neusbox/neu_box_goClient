package cli

import (
	"os/exec"
	"syscall"
)

// ExecFn 用 argv 替换当前进程映像。
//
// 生产实现是 syscall.Exec —— 成功后不返回。只有真的替换进程，docker 才能拿到
// 控制终端、信号和退出码（`-it` 要的就是这个），所以这里不能改成起子进程。
// 注入是为了让测试不必真的去调 docker。
//
// 替换后 stdin/stdout/stderr 沿用的是当前进程的 0/1/2 号 fd；生产入口
// Run(os.Stdout, os.Stderr) 给的就是它们。
type ExecFn func(path string, argv []string, env []string) error

// LookPathFn 解析可执行文件路径；注入是为了让测试不依赖 PATH 上装没装 docker。
type LookPathFn func(file string) (string, error)

func defaultExec(path string, argv []string, env []string) error {
	return syscall.Exec(path, argv, env)
}

func defaultLookPath(file string) (string, error) {
	return exec.LookPath(file)
}
