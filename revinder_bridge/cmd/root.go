/*
Copyright © 2026 sottey
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "revinder_bridge",
	Short: "revinder_bridge task capture service",
	Long:  "revinder_bridge captures tasks and stores them for later sync to Apple Reminders.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
