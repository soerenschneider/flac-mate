package internal

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/soerenschneider/flac-mate/pkg"
	"go.uber.org/multierr"
)

const SyntheticFilePathTag = "_filepath"

const (
	TagAlbumArtist = "ALBUMARTIST"
	TagAlbum       = "ALBUM"
	TagArtist      = "ARTIST"
	TagBand        = "BAND"
	TagComment     = "COMMENT"
	TagComposer    = "COMPOSER"
	TagDate        = "DATE"
	TagDiscNumber  = "DISCNUMBER"
	TagDiscsTotal  = "DISCTOTAL"
	TagGenre       = "GENRE"
	TagTitle       = "TITLE"
	TagTrackNumber = "TRACKNUMBER"
	TagTracksTotal = "TRACKTOTAL"
)

var AllowedTags = map[string]bool{
	TagAlbumArtist: true,
	TagAlbum:       true,
	TagArtist:      true,
	TagBand:        true,
	TagComment:     true,
	TagComposer:    true,
	TagDate:        true,
	TagDiscNumber:  true,
	TagDiscsTotal:  true,
	TagGenre:       true,
	TagTitle:       true,
	TagTrackNumber: true,
	TagTracksTotal: true,
}

// FLAC metadata tags mapping
var MAPPINGS = map[string]string{
	"a":  TagArtist,
	"aa": TagAlbumArtist,
	"b":  TagAlbum,
	"ba": TagBand,
	"c":  TagComposer,
	"cm": TagComment,
	"d":  TagDate,
	"g":  TagGenre,
	"n":  TagTrackNumber,
	"nt": TagTracksTotal,
	"t":  TagTitle,
	"di": TagDiscNumber,
	"dt": TagDiscsTotal,
}

type FlacImage struct {
	Type        string
	MIMEType    string
	Description string
	Width       string
	Height      string
	Depth       string
	Colors      string
	Size        string
}

// FetchMetadata fetches metadata for a given file path.
// The metadata is returned as a map[string]string.
func FetchMetadata(filepath string, tags []string, includeFile bool) (map[string]string, error) {
	_, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	var args []string
	if len(tags) == 0 {
		args = append(args, "--export-tags-to=-")
	} else {
		for _, tag := range tags {
			args = append(args, fmt.Sprintf("--show-tag=%s", tag))
		}
	}
	args = append(args, filepath)

	// Execute metaflac command
	cmd := exec.Command("metaflac", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderrOutput := strings.TrimSpace(stderr.String())
			if stderrOutput != "" {
				return nil, fmt.Errorf("metaflac remove failed (exit code %d): %s", exitError.ExitCode(), stderrOutput)
			}
			return nil, fmt.Errorf("metaflac remove failed with exit code %d", exitError.ExitCode())
		}
		return nil, fmt.Errorf("metaflac remove failed to execute: %v", err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	metadata := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "=") {
			continue
		}

		index := strings.Index(line, "=")
		tag := line[:index]
		//tag := strings.ToLower(line[:index])
		value := line[index+1:]

		if value == "" {
			continue
		}

		// Special handling for tracknumber - zero-pad to 2 digits
		if tag == "TRACKNUMBER" {
			if num, err := strconv.Atoi(value); err == nil {
				metadata[tag] = fmt.Sprintf("%02d", num)
			} else {
				metadata[tag] = value
			}
		} else {
			metadata[tag] = value
		}
	}

	// Attach filepath to metadata if we have any metadata
	if includeFile && len(metadata) > 0 {
		metadata[SyntheticFilePathTag] = filepath
	}

	return metadata, nil
}

func RemoveMetadata(filepath string, data map[string]string) error {
	if len(data) == 0 {
		return errors.New("no data provided")
	}

	_, err := os.Stat(filepath)
	if err != nil {
		return err
	}

	var removeArgs []string
	for tag := range data {
		removeArgs = append(removeArgs, fmt.Sprintf("--remove-tag=%s", tag))
	}

	removeArgs = append(removeArgs, filepath)
	cmd := exec.Command("metaflac", removeArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderrOutput := strings.TrimSpace(stderr.String())
			if stderrOutput != "" {
				return fmt.Errorf("metaflac remove failed (exit code %d): %s",
					exitError.ExitCode(), stderrOutput)
			}
			return fmt.Errorf("metaflac remove failed with exit code %d",
				exitError.ExitCode())
		}
		return fmt.Errorf("metaflac remove failed to execute: %v", err)
	}

	return nil
}

// SetMetadata writes/overwrites a metadata value for a given file.
func SetMetadata(filepath string, data map[string]string, force bool) error {
	fmt.Println(force)
	if len(data) == 0 {
		return errors.New("no data provided")
	}

	_, err := os.Stat(filepath)
	if err != nil {
		return err
	}

	var removeArgs []string
	for tag := range data {
		if strings.HasPrefix(tag, "%") {
			var err error
			tag, err = ExpandTag(tag)
			if err != nil {
				return err
			}
		}
		tag = strings.ToUpper(tag)

		removeArgs = append(removeArgs, fmt.Sprintf("--remove-tag=%s", tag))
	}

	removeArgs = append(removeArgs, filepath)
	cmd := exec.Command("metaflac", removeArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderrOutput := strings.TrimSpace(stderr.String())
			if stderrOutput != "" {
				return fmt.Errorf("metaflac remove failed (exit code %d): %s",
					exitError.ExitCode(), stderrOutput)
			}
			return fmt.Errorf("metaflac remove failed with exit code %d",
				exitError.ExitCode())
		}
		return fmt.Errorf("metaflac remove failed to execute: %v", err)
	}

	var setArgs []string
	for tag, value := range data {
		if strings.HasPrefix(tag, "%") {
			var err error
			tag, err = ExpandTag(tag)
			if err != nil {
				return err
			}
		}

		// only write values that are non-empty
		if strings.TrimSpace(value) != "" {
			_, found := AllowedTags[tag]
			if !found && !force {
				return fmt.Errorf("refusing to write unknown tag %q", tag)
			}

			setArgs = append(setArgs, fmt.Sprintf("--set-tag=%s=%s", tag, value))
		}
	}
	setArgs = append(setArgs, filepath)

	cmd = exec.Command("metaflac", setArgs...)
	stderr = bytes.Buffer{}
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderrOutput := strings.TrimSpace(stderr.String())
			if stderrOutput != "" {
				return fmt.Errorf("metaflac set failed (exit code %d): %s",
					exitError.ExitCode(), stderrOutput)
			}
			return fmt.Errorf("metaflac set failed with exit code %d",
				exitError.ExitCode())
		}
		return fmt.Errorf("metaflac set failed to execute: %v", err)
	}

	return nil
}

// SetPicture deletes all pictures and then writes the specified picture for a given file.
func SetPicture(flacFilePath string, pictureFilePath string) error {
	isValid, _, _, err := pkg.IsValidImage(pictureFilePath)
	if !isValid {
		return err
	}

	if err := DeletePictures(flacFilePath); err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--import-picture-from=%s", pictureFilePath),
		flacFilePath,
	}

	cmd := exec.Command("metaflac", args...)
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderrOutput := strings.TrimSpace(stderr.String())
			if stderrOutput != "" {
				return fmt.Errorf("metaflac set failed (exit code %d): %s", exitError.ExitCode(), stderrOutput)
			}
			return fmt.Errorf("metaflac set failed with exit code %d", exitError.ExitCode())
		}
		return fmt.Errorf("metaflac set failed to execute: %v", err)
	}

	return nil
}

// DeletePictures deletes all pictures from a given flac
func DeletePictures(filepath string) error {
	_, err := os.Stat(filepath)
	if err != nil {
		return err
	}

	args := []string{
		"--remove",
		"--block-type=PICTURE",
		filepath,
	}

	cmd := exec.Command("metaflac", args...)
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderrOutput := strings.TrimSpace(stderr.String())
			if stderrOutput != "" {
				return fmt.Errorf("metaflac set failed (exit code %d): %s", exitError.ExitCode(), stderrOutput)
			}
			return fmt.Errorf("metaflac set failed with exit code %d", exitError.ExitCode())
		}
		return fmt.Errorf("metaflac set failed to execute: %v", err)
	}

	return nil
}

// ExpandTag expands the given tag from short notation to long notation.
// Returns the string representation of the tag.
func ExpandTag(tag string) (string, error) {
	if tag == "" {
		return "", nil
	}

	if expanded, exists := MAPPINGS[strings.ToLower(tag)]; exists {
		return expanded, nil
	}

	return "", fmt.Errorf("short notation %q not found", tag)
}

func GetFlacImages(filepath string) ([]FlacImage, error) {
	cmd := exec.Command("metaflac", "--list", "--block-type=PICTURE", filepath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running metaflac: %v\nOutput: %s", err, out.String())
	}

	lines := strings.Split(out.String(), "\n")
	var images []FlacImage
	var current FlacImage

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "METADATA block #") && current.MIMEType != "" {
			images = append(images, current)
			current = FlacImage{}
		}
		switch {
		case strings.HasPrefix(line, "type:"):
			current.Type = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
		case strings.HasPrefix(line, "MIME type:"):
			current.MIMEType = strings.TrimSpace(strings.TrimPrefix(line, "MIME type:"))
		case strings.HasPrefix(line, "description:"):
			current.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		case strings.HasPrefix(line, "width:"):
			current.Width = strings.TrimSpace(strings.TrimPrefix(line, "width:"))
		case strings.HasPrefix(line, "height:"):
			current.Height = strings.TrimSpace(strings.TrimPrefix(line, "height:"))
		case strings.HasPrefix(line, "depth:"):
			current.Depth = strings.TrimSpace(strings.TrimPrefix(line, "depth:"))
		case strings.HasPrefix(line, "colors:"):
			current.Colors = strings.TrimSpace(strings.TrimPrefix(line, "colors:"))
		case strings.HasPrefix(line, "data length:"):
			current.Size = strings.TrimSpace(strings.TrimPrefix(line, "data length:"))
		}
	}
	if current.MIMEType != "" {
		images = append(images, current)
	}
	return images, nil
}

// String returns a human-readable string representation of the image metadata
func (img FlacImage) String() string {
	return fmt.Sprintf(
		`Image:
  Type        : %s
  MIME Type   : %s
  Description : %s
  Dimensions  : %sx%s px
  Depth       : %s bits
  Colors      : %s
  Size        : %s bytes`,
		img.Type,
		img.MIMEType,
		img.Description,
		img.Width,
		img.Height,
		img.Depth,
		img.Colors,
		img.Size,
	)
}

// ExpandTags expands the given tags from their short notation to long notation.
// Returns a slice of strings for the tags.
func ExpandTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return []string{}, nil
	}

	var result []string
	var errs error
	for _, tag := range tags {
		if strings.HasPrefix(tag, "%") {
			var err error
			tag, err = ExpandTag(tag)
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result, errs
}
