package cmd

import (
	"fmt"

	"github.com/soerenschneider/flac-mate/internal"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints version and exists",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(internal.BuildVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
