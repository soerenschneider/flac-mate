package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var listPicturesCmd = &cobra.Command{
	Use: "pictures-list",
	Aliases: []string{
		"pics-ls",
		"pic-ls",
		"pictures-list",
		"pictures-list",
		"picture-list",
		"picture-ls",
		"ls-pics",
		"ls-pictures",
		"ls-pic",
		"ls-picture",
	},
	Short: "Lists all pictures of a flac file",
	Args:  cobra.ExactArgs(1),
	RunE:  runListPicture,
}

func init() {
	metadataCmd.AddCommand(listPicturesCmd)
}

func runListPicture(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]

	result, err := fetchImages(target)
	if err != nil {
		return err
	}

	return result.Run()
}

func fetchImages(target string) (*internal.GenericResult[map[string][]internal.FlacImage], error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	tableName := "pictureslist"
	result := &internal.GenericResult[map[string][]internal.FlacImage]{
		Operation: tableName,
		Execute:   picturesListAction,
		Data:      make(map[string][]internal.FlacImage),
	}

	if !info.IsDir() {
		images, err := internal.GetFlacImages(target)
		if err != nil {
			return nil, err
		}

		result.Data[target] = images
		return result, nil
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".flac") {
			return nil
		}

		images, err := internal.GetFlacImages(path)
		if err != nil {
			return err
		}

		result.Data[path] = images

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func picturesListAction(action *internal.GenericResult[map[string][]internal.FlacImage]) error {
	var flacImageHeaders = []string{
		"File",
		"Type",
		"MIME Type",
		"Description",
		"Width",
		"Height",
		"Depth",
		"Colors",
		"Size (bytes)",
	}

	var tableData [][]string
	for file, images := range action.Data {
		for _, img := range images {
			row := []string{
				file,
				img.Type,
				img.MIMEType,
				img.Description,
				img.Width,
				img.Height,
				img.Depth,
				img.Colors,
				img.Size,
			}
			tableData = append(tableData, row)
		}
	}

	tui.PrintTable("Pictures", flacImageHeaders, tableData, tui.TableOpts{})
	return nil
}
