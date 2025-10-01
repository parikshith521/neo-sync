package internal

import (
	"fmt"
	"os/exec"
)

func ExecuteCommands(commands []string) error {
	for _, cmdStr := range commands {
		fmt.Printf("▶️ Executing: %s\n", cmdStr)
		cmd := exec.Command("sh", "-c", cmdStr)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", cmdStr, err)
		}
	}
	return nil
}
