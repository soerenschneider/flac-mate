package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var delPictureCmd = &cobra.Command{
	Use: "picture-delete [target]",
	Aliases: []string{
		"picture-del",
		"pic-del",
		"pics-del",
		"pic-delete",
		"pics-delete",
		"del-picture",
		"del-pic",
		"del-pics",
		"delete-pic",
		"delete-pics",
	},
	Short: "Deletes a picture from a flac file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelPicture,
}

func init() {
	metadataCmd.AddCommand(delPictureCmd)
}

func runDelPicture(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]
	action, err := delete(target)
	if err != nil {
		return err
	}

	return action.Run()
}

func delete(target string) (*internal.GenericResult[[]string], error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	action := &internal.GenericResult[[]string]{
		Operation: "pic-delete",
		Data:      make([]string, 0),
		Execute:   picturesDeleteAction,
	}

	if !info.IsDir() {
		action.Data = []string{target}
		return action, nil
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".flac") {
			return nil
		}

		action.Data = append(action.Data, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return action, nil
}

func picturesDeleteAction(action *internal.GenericResult[[]string]) error {
	if len(action.Data) == 0 {
		return nil
	}

	var tableData [][]string
	for _, file := range action.Data {
		tableData = append(tableData, []string{file})
	}

	tui.PrintTable("Affected Files", []string{"File"}, tableData, tui.TableOpts{})

	proceed, err := tui.Confirm("Proceed with deleting pictures?")
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	for _, flac := range action.Data {
		if err := internal.DeletePictures(flac); err != nil {
			return err
		}
	}

	return nil
}
