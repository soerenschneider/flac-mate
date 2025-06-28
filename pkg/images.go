package pkg

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
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

// GetMainCover tries to guess which image is the correct cover
func GetMainCover(dirname string, images []string) string {
	if len(images) == 0 {
		return ""
	}

	if len(images) == 1 {
		return images[0]
	}

	// Look for cover-indicating names
	for _, image := range images {
		lower := strings.ToLower(image)
		if strings.Contains(lower, "front") ||
			strings.Contains(lower, "folder") ||
			strings.Contains(lower, "cover") ||
			strings.Contains(lower, "devant") {
			return image
		}
	}

	// Return the largest file
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

	return maxImage
}
