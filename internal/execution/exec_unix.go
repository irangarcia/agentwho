//go:build darwin || linux

package execution

import "syscall"

func Replace(executable string, args, env []string) error {
	return syscall.Exec(executable, append([]string{executable}, args...), env)
}
