//go:build !windows

package ojadapter

import "os/exec"

func configure(cmd *exec.Cmd) {}
