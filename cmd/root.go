package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "flac-metadata",
	Short: "A tool for managing FLAC metadata",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}
