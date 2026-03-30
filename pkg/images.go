package pkg

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// IsValidImage checks if the file is a valid JPEG or PNG image,
// and returns its validity, format (jpeg/png), MIME type, and error.
func IsValidImage(filePath string) (bool, string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	_, format, err := image.DecodeConfig(file)
	if err != nil {
		return false, "", "", nil // Not a valid image
	}

	var mimeType string
	switch format {
	case "jpeg":
		mimeType = "image/jpeg"
	case "png":
		mimeType = "image/png"
	default:
		return false, format, "", nil
	}

	return true, format, mimeType, nil
}

var ErrNoImages = errors.New("no images found")

// GetMainCover tries to guess which image is the correct cover
func GetMainCover(dirname string, images []string) (string, error) {
	if len(images) == 0 {
		return "", ErrNoImages
	}

	if len(images) == 1 {
		return images[0], nil
	}

	// Look for cover-indicating names
	var coverNames = []string{
		"front",
		"folder",
		"cover",
		"devant",
		"albumart",
		"album art",
		"artwork",
	}

	// Look for cover-indicating names (basename, ignoring extension)
	for _, image := range images {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(image), filepath.Ext(image)))
		for _, name := range coverNames {
			if base == name || strings.HasPrefix(base, name+"-") || strings.HasPrefix(base, name+"_") {
				return image, nil
			}
		}
	}

	// Look for a square (or near-square) image, typical for album art
	for _, image := range images {
		path := filepath.Join(dirname, image)
		if IsNearlySquare(path, 0.1) {
			return image, nil
		}
	}

	// Fall back to the largest file
	maxSize := int64(0)
	maxImage := images[0]
	for _, image := range images {
		path := filepath.Join(dirname, image)
		if info, err := os.Stat(path); err == nil {
			if info.Size() > maxSize {
				maxSize = info.Size()
				maxImage = image
			}
		}
	}

	return maxImage, nil
}

// IsNearlySquare returns true if the image at path has an aspect ratio
// within tolerance of 1:1 (e.g. tolerance=0.1 allows up to 10% deviation).
func IsNearlySquare(path string, tolerance float64) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() {
		_ = f.Close()
	}()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return false
	}

	if cfg.Height == 0 {
		return false
	}

	ratio := float64(cfg.Width) / float64(cfg.Height)
	return math.Abs(ratio-1.0) <= tolerance
}
