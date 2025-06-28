package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/soerenschneider/flac-mate/pkg"

	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

var renameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename FLAC files and directories based on metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runRenamer,
}

const (
	defaultRenameFileScheme = "%n - %a - %t"
	defaultRenameDirScheme  = "%a - %d - %b"
)

var (
	// Unwanted characters for filenames
	unwantedCharsRegex = regexp.MustCompile(`[/\\!?%$*|"'<>]`)

	flagRenameScheme    string
	flagRenameDirScheme string
	flagRenameDryrun    bool
	flagRenameCoverName string
)

func init() {
	RootCmd.AddCommand(renameCmd)

	renameCmd.Flags().StringVarP(&flagRenameScheme, "file-scheme", "f", defaultRenameFileScheme, "File naming scheme")
	renameCmd.Flags().StringVarP(&flagRenameDirScheme, "directory-scheme", "d", defaultRenameDirScheme, "Directory naming scheme")
	renameCmd.Flags().BoolVarP(&flagRenameDryrun, "dry-run", "n", false, "Dry run mode")
	renameCmd.Flags().StringVarP(&flagRenameCoverName, "cover-name", "c", "cover", "Cover image name")
}

// runRenamer is the main command handler
func runRenamer(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]

	fileScheme, err := unwrapKeys(flagRenameScheme, false)
	if err != nil {
		return err
	}

	dirScheme, err := unwrapKeys(flagRenameDirScheme, true)
	if err != nil {
		return err
	}

	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		return err
	}

	target = strings.TrimSuffix(target, "/")

	// Walk directories in reverse order (bottom-up)
	var dirs []string
	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
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

		action := workDir(dirname, filenames, fileScheme, dirScheme, flagRenameCoverName)
		if action != nil && action.Actionable() {
			if err := action.CarryOut(flagRenameDryrun); err != nil {
				errs = multierr.Append(errs, err)
			}
		} else {
			successStyle := lipgloss.NewStyle().
				Bold(true)
			fmt.Println(successStyle.Render("✓ No renaming needed!"))
		}

		if action.EncounteredErrors() {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

// Action accumulates all the actions needed to rename target files
type renameAction struct {
	Dir            string
	FileActions    map[string]string
	DirAction      [2]string // [old, new]
	ImageAction    [2]string // [old, new]
	Errors         []error
	hasDirAction   bool
	hasImageAction bool
}

// NewAction creates a new Action instance
func NewAction(directory string) *renameAction {
	return &renameAction{
		Dir:         directory,
		FileActions: make(map[string]string),
		Errors:      make([]error, 0),
	}
}

// AddError adds an error to the action
func (a *renameAction) AddError(err error) {
	a.Errors = append(a.Errors, err)
}

// AddFileAction adds an action to rename a file
func (a *renameAction) AddFileAction(oldFilepath, newFilepath string) {
	a.FileActions[oldFilepath] = newFilepath
}

// SetDirAction adds an action to rename a directory
func (a *renameAction) SetDirAction(oldFilepath, newFilepath string) {
	a.DirAction = [2]string{oldFilepath, newFilepath}
	a.hasDirAction = true
}

// SetImageAction adds an action to rename an image file
func (a *renameAction) SetImageAction(oldFilepath, newFilepath string) {
	a.ImageAction = [2]string{oldFilepath, newFilepath}
	a.hasImageAction = true
}

func (a *renameAction) Actionable() bool {
	return len(a.FileActions) > 0 || a.hasDirAction
}

func (a *renameAction) EncounteredErrors() bool {
	return len(a.Errors) > 0
}

// CarryOut actually performs the operations
func (a *renameAction) CarryOut(dryrun bool) error {
	var data [][]string
	headers := []string{"Old", "New"}

	// Rename files
	for oldPath, newPath := range a.FileActions {
		data = append(data, []string{oldPath, newPath})
	}

	// Rename image
	if a.hasImageAction {
		data = append(data, []string{a.ImageAction[0], a.ImageAction[1]})
	}

	// Rename directory
	if a.hasDirAction {
		data = append(data, []string{a.DirAction[0], a.DirAction[1]})
	}

	tui.PrintTable("Move", headers, data, tui.TableOpts{})

	if dryrun {
		return nil
	}

	proceed, err := tui.Confirm("Proceed?")
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	for _, tuples := range data {
		if err := os.Rename(tuples[0], tuples[1]); err != nil {
			return err
		}
	}

	return nil
}

// unwrapKeys processes the scheming arguments
func unwrapKeys(scheme string, directory bool) (string, error) {
	if !directory {
		// Check if %t or %n is present for unique filename
		if !strings.Contains(scheme, "%t") && !strings.Contains(scheme, "%n") {
			return "", fmt.Errorf("error: %%t or %%n has to be present in scheme")
		}
	}

	// Replace short tags with long format
	for short, long := range internal.MAPPINGS {
		scheme = strings.ReplaceAll(scheme, "%"+short, "%("+long+")s")
	}

	return scheme, nil
}

// hasSufficientMetadata checks if metadata is sufficient for rename operation
func hasSufficientMetadata(metadata map[string]string, scheme string) bool {
	if len(metadata) == 0 || scheme == "" {
		return false
	}

	// Find all tags in scheme like %(tagname)s
	re := regexp.MustCompile(`\((\w+)\)`)
	matches := re.FindAllStringSubmatch(scheme, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1]
			if value, exists := metadata[tag]; !exists || value == "" {
				return false
			}
		}
	}

	return true
}

// renameFile renames a file based on scheme and metadata
func renameFile(scheme, dirname, filename string, metadata map[string]string) (string, string, bool) {
	path := filepath.Join(dirname, filename)

	// Apply metadata to scheme
	newFilename := applyMetadataToScheme(scheme, metadata) + ".flac"
	newFilename = unwantedCharsRegex.ReplaceAllString(newFilename, "")
	newFilepath := filepath.Join(dirname, newFilename)

	if path == newFilepath {
		return "", "", false
	}

	return path, newFilepath, true
}

// renameDir renames directory based on scheme and metadata
func renameDir(dirScheme, dirname string, fileMetadata map[string]string) (string, string, bool) {
	path := filepath.Dir(dirname)
	newDirname := applyMetadataToScheme(dirScheme, fileMetadata)
	newDirname = unwantedCharsRegex.ReplaceAllString(newDirname, "")
	newFilepath := filepath.Join(path, newDirname)

	if dirname == newFilepath {
		return "", "", false
	}

	return dirname, newFilepath, true
}

// applyMetadataToScheme applies metadata values to a naming scheme
func applyMetadataToScheme(scheme string, metadata map[string]string) string {
	result := scheme

	// Replace %(tag)s patterns with actual values
	re := regexp.MustCompile(`%\((\w+)\)s`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		tagMatch := re.FindStringSubmatch(match)
		if len(tagMatch) > 1 {
			tag := tagMatch[1]
			if value, exists := metadata[tag]; exists {
				return value
			}
		}
		return match
	})

	return result
}

// appendMetadata appends file metadata to album metadata collection
func appendMetadata(albumMetadata map[string]map[string]bool, fileMetadata map[string]string) {
	for key, value := range fileMetadata {
		if albumMetadata[key] == nil {
			albumMetadata[key] = make(map[string]bool)
		}
		albumMetadata[key][value] = true
	}
}

// canRenameDirectory decides if directory can be renamed based on metadata
func canRenameDirectory(metadata map[string]map[string]bool, scheme string) (bool, error) {
	if len(metadata) == 0 {
		return false, errors.New("empty metadata")
	}

	// find tags with multiple values
	var multiValuedTags []string
	for _, tag := range flagMetaUniformTags {
		if metadata[tag] != nil && len(metadata[tag]) > 1 {
			multiValuedTags = append(multiValuedTags, tag)
		}
	}
	if len(multiValuedTags) > 0 {
		return false, fmt.Errorf("found multi-valued tags %v", multiValuedTags)
	}

	re := regexp.MustCompile(`\((\w+)\)`)
	matches := re.FindAllStringSubmatch(scheme, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1]
			tagData, exists := metadata[tag]
			if !exists || len(tagData) == 0 || len(tagData) != 1 {
				return false, fmt.Errorf("missing tag %q", tag)
			}

			// Check if the single value is not empty
			for value := range tagData {
				if value == "" {
					return false, errors.New("tag is empty")
				}
			}
		}
	}

	return true, nil
}

func renameCover(dirname string, images []string, coverName string) (string, string, bool) {
	selectedImage := pkg.GetMainCover(dirname, images)
	if selectedImage == "" {
		return "", "", false
	}

	oldFilepath := filepath.Join(dirname, selectedImage)
	newFilename := coverName + filepath.Ext(selectedImage)
	newFilepath := filepath.Join(dirname, newFilename)

	if oldFilepath == newFilepath {
		return "", "", false
	}

	return oldFilepath, newFilepath, true
}

func isImage(filename string) bool {
	imageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".tiff": true,
		".webp": true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if imageExtensions[ext] {
		isValid, _, _, _ := pkg.IsValidImage(filename)
		return isValid
	}

	return false
}

func workDir(dirname string, filenames []string, fileScheme, dirScheme, coverName string) *renameAction {
	albumMetadata := make(map[string]map[string]bool)
	var collectedImages []string
	var fileMetadata map[string]string
	dirContainsMusic := false

	action := NewAction(dirname)

	// Sort filenames for consistent processing
	sort.Strings(filenames)

	for _, filename := range filenames {
		if strings.HasSuffix(strings.ToLower(filename), ".flac") {
			dirContainsMusic = true
			filepath := filepath.Join(dirname, filename)

			var err error
			fileMetadata, err = internal.FetchMetadata(filepath, nil, false)
			if err != nil {
				action.AddError(fmt.Errorf("error fetching metadata for %s: %v", filename, err))
				continue
			}

			if !hasSufficientMetadata(fileMetadata, fileScheme) {
				action.AddError(fmt.Errorf("no sufficient metadata for %s", filename))
			} else {
				if oldPath, newPath, shouldRename := renameFile(fileScheme, dirname, filename, fileMetadata); shouldRename {
					action.AddFileAction(oldPath, newPath)
				}
				appendMetadata(albumMetadata, fileMetadata)
			}
		} else if isImage(filename) {
			collectedImages = append(collectedImages, filename)
		}
	}

	if dirContainsMusic {
		// Handle cover image renaming
		if len(collectedImages) > 0 {
			if oldPath, newPath, shouldRename := renameCover(dirname, collectedImages, coverName); shouldRename {
				action.SetImageAction(oldPath, newPath)
			}
		}

		// Handle directory renaming
		canRenameDir, err := canRenameDirectory(albumMetadata, dirScheme)
		if err != nil {
			action.AddError(err)
		} else if canRenameDir && fileMetadata != nil {
			// Convert albumMetadata to single-value metadata for directory naming
			singleMetadata := make(map[string]string)
			for key, valueSet := range albumMetadata {
				if len(valueSet) == 1 {
					for value := range valueSet {
						singleMetadata[key] = value
						break
					}
				}
			}

			if oldPath, newPath, shouldRename := renameDir(dirScheme, dirname, singleMetadata); shouldRename {
				action.SetDirAction(oldPath, newPath)
			}
		} else {
			action.AddError(fmt.Errorf("cannot rename %s due to missing metadata", dirname))
		}
	}

	return action
}
