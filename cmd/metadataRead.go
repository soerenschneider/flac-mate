package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use: "read [target]",
	Aliases: []string{
		"cat",
	},
	Short: "Reads metadata for all flac files under target",
	Args:  cobra.ExactArgs(1),
	RunE:  runRead,
}

func init() {
	metadataCmd.AddCommand(readCmd)
	readCmd.Flags().StringSliceVarP(&flagMetaReadTags, "tag", "t", nil, "Tags to read")
	readCmd.Flags().BoolVarP(&flagMetaJsonOutput, "json", "j", false, "Encode result to JSON instead of printing a human-friendly table")
}

func runRead(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]

	action, err := readMetadata(target, flagMetaReadTags)
	if err != nil {
		return err
	}

	return action.Run()
}

func readMetadata(target string, tags []string) (*internal.GenericResult[[]map[string]string], error) {
	expandedTags, err := internal.ExpandTags(tags)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	result := &internal.GenericResult[[]map[string]string]{
		Operation: "read",
		Data:      make([]map[string]string, 0),
		Execute:   readAction,
	}

	if !info.IsDir() {
		metadata, err := internal.FetchMetadata(target, expandedTags, true)
		if err != nil {
			return nil, err
		}
		result.Data = append(result.Data, metadata)
		return result, nil
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".flac") {
			return nil
		}

		metadata, err := internal.FetchMetadata(path, expandedTags, true)
		if err != nil {
			return err
		}
		result.Data = append(result.Data, metadata)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func readAction(action *internal.GenericResult[[]map[string]string]) error {
	if flagMetaJsonOutput {
		encoded, err := json.Marshal(action.Data)
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}

	tui.PrintMetadataTable(action.Data)
	return nil
}
