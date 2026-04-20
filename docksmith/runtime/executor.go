package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Executor struct {
	rootfsDir string
	outputDir string
	workDir   string
	env       map[string]string
}

func NewExecutor(rootfsDir, outputDir string) *Executor {
	return &Executor{
		rootfsDir: rootfsDir,
		outputDir: outputDir,
		workDir:   "/",
		env:       make(map[string]string),
	}
}

func (e *Executor) SetWorkDir(dir string) { e.workDir = dir }
func (e *Executor) SetEnv(env map[string]string) { e.env = env }

func (e *Executor) snapshotRootfs() (map[string]os.FileInfo, error) {
	state := make(map[string]os.FileInfo)
	err := filepath.Walk(e.rootfsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		relPath, _ := filepath.Rel(e.rootfsDir, path)
		state[relPath] = info
		return nil
	})
	return state, err
}

func (e *Executor) Run(command []string) error {
	if len(command) == 0 { return fmt.Errorf("no command specified") }
	workDirPath := filepath.Join(e.rootfsDir, e.workDir)
	os.MkdirAll(workDirPath, 0755)
	return e.runWithNamespaces(command)
}

func (e *Executor) runWithNamespaces(command []string) error {
	preState, err := e.snapshotRootfs()
	if err != nil { return err }

	// FIX: Use $@ to preserve array arguments
	wrapperScript := fmt.Sprintf("#!/bin/sh\nset -e\ncd %s || exit 1\nexec \"$@\"\n", e.workDir)
	wrapperPath := filepath.Join(e.rootfsDir, "tmp", "docksmith-run.sh")
	os.MkdirAll(filepath.Dir(wrapperPath), 0755)
	os.WriteFile(wrapperPath, []byte(wrapperScript), 0755)

	envVars := []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	for k, v := range e.env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	args := []string{"/tmp/docksmith-run.sh"}
	args = append(args, command...)

	cmd := exec.Command("/bin/sh", args...)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
		Chroot:     e.rootfsDir,
	}

	if err := cmd.Run(); err != nil { return err }
	return e.captureChanges(preState)
}

func (e *Executor) captureChanges(preState map[string]os.FileInfo) error {
	return filepath.Walk(e.rootfsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		relPath, _ := filepath.Rel(e.rootfsDir, path)
		if relPath == "." || relPath == "tmp" || relPath == "tmp/docksmith-run.sh" { return nil }

		preInfo, existedBefore := preState[relPath]
		isModified := !existedBefore || preInfo.Size() != info.Size() || preInfo.ModTime() != info.ModTime()

		if isModified {
			destPath := filepath.Join(e.outputDir, relPath)
			if info.IsDir() { return os.MkdirAll(destPath, info.Mode()) }
			os.MkdirAll(filepath.Dir(destPath), 0755)
			data, _ := os.ReadFile(path)
			return os.WriteFile(destPath, data, info.Mode())
		}
		return nil
	})
}
