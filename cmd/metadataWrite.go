package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write [target]",
	Short: "Writes metadata for a single tag for all flac files in target",
	Args:  cobra.ExactArgs(1),
	RunE:  runWrite,
}

func init() {
	metadataCmd.AddCommand(writeCmd)
	writeCmd.Flags().BoolVarP(&flagMetaWriteForce, "force", "f", false, "Force writing unknown tags")
	writeCmd.Flags().StringToStringVarP(&flagMetaWriteData, "data", "d", nil, "Data to write (format: tag1=value1,tag2=value2)")
	_ = writeCmd.MarkFlagRequired("data")
}

func runWrite(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]
	action, err := writeMetadata(target, flagMetaWriteData)
	if err != nil {
		return err
	}

	return action.Run()
}

func writeMetadata(target string, metadata map[string]string) (*internal.GenericResult[map[string]map[string]string], error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	for tag := range metadata {
		_, found := internal.AllowedTags[tag]
		if !found && !flagMetaWriteForce {
			return nil, fmt.Errorf("refusing to write unknown tag %q", tag)
		}
	}

	action := &internal.GenericResult[map[string]map[string]string]{
		Operation: "write",
		Data:      make(map[string]map[string]string),
		Execute:   writeAction,
	}

	if !info.IsDir() {
		action.Data[target] = metadata
		return action, nil
	}

	for _, tag := range unsafeRecursiveTags {
		_, found := metadata[tag]
		if found {
			return nil, fmt.Errorf("refusing to recursively write tag %s", tag)
		}
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".flac") {
			return nil
		}

		action.Data[path] = metadata
		return nil
	})

	if err != nil {
		return nil, err
	}

	return action, nil
}

func writeAction(action *internal.GenericResult[map[string]map[string]string]) error {
	if len(action.Data) == 0 {
		return nil
	}

	var tableData [][]string
	for file, metadata := range action.Data {
		for tag, value := range metadata {
			tableData = append(tableData, []string{file, tag, value})
		}
	}

	tui.PrintTable("Affected Files", []string{"File", "Tag", "Value"}, tableData, tui.TableOpts{})

	proceed, err := tui.Confirm("Proceed with writing metadata?")
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	for file, metadata := range action.Data {
		if err := internal.SetMetadata(file, metadata, flagMetaWriteForce); err != nil {
			return err
		}
	}

	return nil
}
