package runtime

import (
	"docksmith/images"
	"docksmith/layers"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Container struct {
	image     *images.ImageManifest
	rootfsDir string
	command   []string
}

func NewContainer(image *images.ImageManifest, overrideCmd []string) *Container {
	cmd := image.Config.Cmd
	if len(overrideCmd) > 0 { cmd = overrideCmd }
	return &Container{ image: image, command: cmd }
}

func (c *Container) Run() error {
	tmpDir, err := os.MkdirTemp("", "docksmith-run-*")
	if err != nil { return fmt.Errorf("failed to create temp dir: %w", err) }
	c.rootfsDir = tmpDir
	defer c.Cleanup()
	
	fmt.Printf("Extracting %d layers...\n", len(c.image.Layers))
	if err := layers.ExtractLayers(c.image.GetLayerDigests(), c.rootfsDir); err != nil {
		return fmt.Errorf("failed to extract layers: %w", err)
	}
	
	fmt.Printf("Running: %s\n", strings.Join(c.command, " "))
	return c.execute()
}

func (c *Container) execute() error {
	if len(c.command) == 0 { return fmt.Errorf("no command specified") }
	
	workDir := c.image.Config.WorkingDir
	if workDir == "" { workDir = "/" }
	
	workDirPath := filepath.Join(c.rootfsDir, workDir)
	if err := os.MkdirAll(workDirPath, 0755); err != nil { return fmt.Errorf("failed to create workdir: %w", err) }
	
	env := c.image.Config.Env
	if len(env) == 0 {
		env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	}
	return c.runWithIsolation(workDir, env)
}

func (c *Container) runWithIsolation(workDir string, env []string) error {
	// FIX: Use $@ to perfectly preserve JSON array arguments
	wrapperScript := fmt.Sprintf("#!/bin/sh\ncd %s || exit 1\nexec \"$@\"\n", workDir)
	
	wrapperPath := filepath.Join(c.rootfsDir, "tmp", "docksmith-exec.sh")
	os.MkdirAll(filepath.Dir(wrapperPath), 0755)
	os.WriteFile(wrapperPath, []byte(wrapperScript), 0755)
	
	// Pass the command array directly to exec.Command
	args := []string{"/tmp/docksmith-exec.sh"}
	args = append(args, c.command...)
	
	cmd := exec.Command("/bin/sh", args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
		Chroot:     c.rootfsDir,
	}
	
	return cmd.Run()
}

func (c *Container) Cleanup() {
	if c.rootfsDir != "" { os.RemoveAll(c.rootfsDir) }
}
