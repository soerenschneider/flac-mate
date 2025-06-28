package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var addPictureCmd = &cobra.Command{
	Use: "picture-add [target]",
	Aliases: []string{
		"pic-add",
		"add-picture",
		"add-pic",
	},
	Short: "Adds a picture to a flac file",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddPicture,
}

func init() {
	metadataCmd.AddCommand(addPictureCmd)
	addPictureCmd.Flags().StringVarP(&flagMetaPictureFile, "picture", "p", "", "Picture file to add to the flac")
}

func runAddPicture(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]
	result, err := addPicture(target, flagMetaPictureFile)
	if err != nil {
		return err
	}

	return result.Run()
}

func addPicture(target string, picture string) (*internal.GenericResult[addImageOp], error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	result := &internal.GenericResult[addImageOp]{
		Operation: "add-picture",
		Data: addImageOp{
			ImageFile: picture,
		},
		Execute: picturesAddAction,
	}

	if !info.IsDir() {
		result.Data.Files = []string{target}
		return result, nil
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".flac") {
			return nil
		}

		result.Data.Files = append(result.Data.Files, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

type addImageOp struct {
	ImageFile string
	Files     []string
}

func picturesAddAction(action *internal.GenericResult[addImageOp]) error {
	if len(action.Data.Files) == 0 {
		return nil
	}

	var tableData [][]string
	for _, file := range action.Data.Files {
		tableData = append(tableData, []string{file})
	}

	tui.PrintTable("Affected Files", []string{"File"}, tableData, tui.TableOpts{})

	proceed, err := tui.Confirm("Proceed with adding pictures?")
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	for _, file := range action.Data.Files {
		if err := internal.SetPicture(file, action.Data.ImageFile); err != nil {
			return err
		}
	}

	return nil
}
