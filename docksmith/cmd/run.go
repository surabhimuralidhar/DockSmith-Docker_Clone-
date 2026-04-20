package cmd

import (
	"docksmith/images"
	"docksmith/runtime"
	"docksmith/util"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var envOverrides []string

// RunCmd represents the run command
var RunCmd = &cobra.Command{
	Use:   "run IMAGE[:TAG] [COMMAND]",
	Short: "Run a command in a new container",
	Long:  `Create and run a container from an image. Optionally override the default command.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRun,
}

func init() {
	// Add the -e / --env flag. StringArrayVarP allows it to be used multiple times (repeatable)
	RunCmd.Flags().StringArrayVarP(&envOverrides, "env", "e", []string{}, "Override or add an environment variable (repeatable)")
}

func runRun(cmd *cobra.Command, args []string) error {
	imageRef := args[0]
	var overrideCmd []string
	if len(args) > 1 {
		overrideCmd = args[1:]
	}
	
	if err := util.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to initialize directories: %w", err)
	}
	
	image, err := images.LoadImage(imageRef)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	
	// Apply environment overrides if the -e flag was used
	if len(envOverrides) > 0 {
		envMap := make(map[string]string)
		
		// First load existing image environment variables
		for _, e := range image.Config.Env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}
		
		// Then apply user overrides
		for _, e := range envOverrides {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}
		
		// Rebuild the final environment list
		var newEnv []string
		for k, v := range envMap {
			newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, v))
		}
		image.Config.Env = newEnv
	}
	
	container := runtime.NewContainer(image, overrideCmd)
	
	if err := container.Run(); err != nil {
		return fmt.Errorf("container execution failed: %w", err)
	}
	
	return nil
}
