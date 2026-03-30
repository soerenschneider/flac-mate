package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/flac-mate/pkg"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

var metaPictureExtractCmd = &cobra.Command{
	Use: "picture-extract [target]",
	Aliases: []string{
		"pic-extract",
		"extract-picture",
		"extract-pic",
	},
	Short: "Extract a picture from a flac file to",
	Args:  cobra.ExactArgs(1),
	RunE:  runMetaPictureExtract,
}

func init() {
	metadataCmd.AddCommand(metaPictureExtractCmd)
	metaPictureExtractCmd.Flags().StringVarP(&flagMetaPictureFile, "picture", "p", "", "Picture file to add to the flac")
}

func runMetaPictureExtract(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := pkg.GetExpandedFile(args[0])

	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		return err
	}

	target = strings.TrimSuffix(target, "/")

	// Walk directories in reverse order (bottom-up)
	var dirs []string
	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Process directories from deepest to shallowest
	var errs error
	for i := len(dirs) - 1; i >= 0; i-- {
		dirname := dirs[i]

		entries, err := os.ReadDir(dirname)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}

		var filenames []string
		for _, entry := range entries {
			if !entry.IsDir() {
				filenames = append(filenames, entry.Name())
			}
		}

		if err := extractCoverForDir(dirname, filenames); err != nil {
			fmt.Println(err)
		}
	}

	return errs
}

func extractCoverForDir(basedir string, filenames []string) error {
	var collectedImages []string
	var flacFiles []string

	// Sort filenames for consistent processing
	sort.Strings(filenames)

	for _, filename := range filenames {
		if strings.HasSuffix(strings.ToLower(filename), ".flac") {
			flacFiles = append(flacFiles, filename)
		} else if isImage(filepath.Join(basedir, filename)) {
			collectedImages = append(collectedImages, filename)
		}
	}

	if len(flacFiles) == 0 {
		return nil // not a music dir
	}

	// Handle cover image renaming
	cover, err := pkg.GetMainCover(basedir, collectedImages)
	if cover != "" && err == nil {
		return err
	}

	var errs error
	for _, file := range flacFiles {
		path := filepath.Join(basedir, file)
		_, err := tryExtractCover(basedir, path)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		} else {
			return errs
		}
	}

	return errs
}

var ErrNoCoverFound = errors.New("no cover image found in metadata")

func tryExtractCover(basedir, path string) (string, error) {
	tmp, err := os.CreateTemp(basedir, "cover-*")
	if err != nil {
		return "", err
	}
	_ = tmp.Close()
	defer func() {
		_ = os.Remove(tmp.Name())
	}()

	if err := exec.Command("metaflac", "--export-picture-to="+tmp.Name(), path).Run(); err != nil {
		return "", err
	}

	ext, err := imageExt(tmp.Name())
	if err != nil {
		return "", err
	}

	outPath := filepath.Join(basedir, "cover"+ext)
	log.Info().Str("path", outPath).Msg("extracting cover")
	if err := os.Rename(tmp.Name(), outPath); err != nil {
		return "", fmt.Errorf("saving cover: %w", err)
	}

	return outPath, nil
}

// imageExt sniffs the first few bytes of a file to determine the image format.
func imageExt(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	buf := make([]byte, 12)
	if _, err := f.Read(buf); err != nil {
		return "", err
	}

	switch {
	case buf[0] == 0xff && buf[1] == 0xd8:
		return ".jpg", nil
	case string(buf[:8]) == "\x89PNG\r\n\x1a\n":
		return ".png", nil
	case string(buf[:4]) == "GIF8":
		return ".gif", nil
	case string(buf[:4]) == "RIFF" && string(buf[8:12]) == "WEBP":
		return ".webp", nil
	default:
		return "", fmt.Errorf("unrecognised image format")
	}
}
