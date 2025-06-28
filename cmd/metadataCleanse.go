package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var cleanseCmd = &cobra.Command{
	Use:   "cleanse [target]",
	Short: "Cleanse all files",
	Args:  cobra.ExactArgs(1),
	RunE:  runCleanse,
}

func init() {
	metadataCmd.AddCommand(cleanseCmd)
}

func runCleanse(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]

	action, err := cleanse(target)
	if err != nil {
		return err
	}

	return action.Run()
}

type CleanseData struct {
	Metadata map[string]string
	Files    []string
}

// filename -> {tag: value}
func cleanse(target string) (*internal.GenericResult[map[string]map[string]string], error) {
	collectedMetadata, err := collectMetadataForFile(target)
	if err != nil {
		return nil, err
	}

	data := make(map[string]map[string]string)

	for file, metadata := range collectedMetadata {
		for tag := range metadata {
			if !slices.Contains(defaultCleansedTags, tag) {
				_, found := data[file]
				if !found {
					data[file] = map[string]string{}
				}
				data[file][tag] = ""
			}
		}
	}

	return &internal.GenericResult[map[string]map[string]string]{
		Operation: "cleanse",
		Data:      data,
		Execute:   cleanseAction,
	}, nil
}

func collectMetadataForFile(target string) (map[string]map[string]string, error) {
	collectedMetadata := make(map[string]map[string]string)

	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		fileMetadata, err := internal.FetchMetadata(target, nil, false)
		if err != nil {
			return nil, err
		}
		collectedMetadata[target] = make(map[string]string)
		collectedMetadata[target] = fileMetadata
		return collectedMetadata, nil
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".flac") {
			return nil
		}

		fileMetadata, err := internal.FetchMetadata(path, nil, false)
		if err != nil {
			return err
		}
		collectedMetadata[path] = make(map[string]string)
		collectedMetadata[path] = fileMetadata
		return nil
	})

	return collectedMetadata, err
}

func cleanseAction(action *internal.GenericResult[map[string]map[string]string]) error {
	if len(action.Data) == 0 {
		successStyle := lipgloss.NewStyle().
			Bold(true)
		fmt.Println(successStyle.Render("✓ No cleansing needed!"))
		return nil
	}

	var tableData [][]string
	for file, metadata := range action.Data {
		for tag, value := range metadata {
			tableData = append(tableData, []string{file, tag, value})
		}
	}

	tui.PrintTable("Cleanse", []string{"File", "Tag", "Value"}, tableData, tui.TableOpts{})

	proceed, err := tui.Confirm("Proceed with cleansing files?")
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	for file, data := range action.Data {
		if err := internal.RemoveMetadata(file, data); err != nil {
			return err
		}
	}

	return nil
}
