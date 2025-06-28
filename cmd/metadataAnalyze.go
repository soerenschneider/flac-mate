package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/soerenschneider/flac-mate/internal"
	"github.com/soerenschneider/flac-mate/internal/tui"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [target]",
	Short: "Make sure that the supplied tags have only a single value across all files",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyze,
}

func init() {
	metadataCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringSliceVarP(&flagMetaUniformTags, "tags", "t", defaultUniformCmdTags, "Tags to check for uniformity")
	analyzeCmd.Flags().BoolVarP(&flagMetaJsonOutput, "json", "j", false, "Encode result to JSON instead of printing a human-friendly table")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	target := args[0]

	action, err := analyzeMetadata(target)
	if err != nil {
		return err
	}

	return action.Run()
}

func analyzeMetadata(target string) (*internal.GenericResult[analyzeResult], error) {
	collectedMetadata, err := collectMetadataForFile(target)
	if err != nil {
		return nil, err
	}

	result := analyzeResult{
		MissingTags:   make(map[string][]string),
		UndesiredTags: make(map[string]map[string]string),
	}

	wantedTags := map[string]bool{}
	for _, tag := range flagMetaUniformTags {
		wantedTags[tag] = true
	}

	missingTagsList := getMissingTags(collectedMetadata, wantedTags)
	if err != nil {
		return nil, err
	}

	for file, missingTags := range missingTagsList {
		_, found := result.MissingTags[file]
		if !found {
			result.MissingTags[file] = make([]string, 0)
		}
		result.MissingTags[file] = append(result.MissingTags[file], missingTags...)
	}

	missingCovers, err := getFilesWithMissingCovers(collectedMetadata)
	if err != nil {
		return nil, err
	}

	result.MissingCovers = missingCovers

	result.MultiValuedTags = getMultiValuedKeys(collectedMetadata, flagMetaUniformTags)

	//
	// Check for undesired tags
	for file, metadata := range collectedMetadata {
		for tag, value := range metadata {
			if !strings.HasPrefix(tag, "_") && !slices.Contains(defaultCleansedTags, tag) {
				_, found := result.UndesiredTags[file]
				if !found {
					result.UndesiredTags[file] = map[string]string{}
				}
				result.UndesiredTags[file][tag] = value
			}
		}
	}

	return &internal.GenericResult[analyzeResult]{
		Operation: "analyze",
		Data:      result,
		Execute:   analyzeAction,
	}, nil
}

func getMissingTags(albumMetadata map[string]map[string]string, tags map[string]bool) map[string][]string {
	missing := make(map[string][]string)
	for file, existentTags := range albumMetadata {
		for wantedTag := range tags {
			val := existentTags[wantedTag]
			if len(val) == 0 {
				_, found := missing[file]
				if !found {
					missing[file] = make([]string, 0)
				}
				missing[file] = append(missing[file], wantedTag)
			}
		}
	}
	return missing
}

func getFilesWithMissingCovers(albumMetadata map[string]map[string]string) ([]string, error) {
	missing := make([]string, 0)
	for file := range albumMetadata {
		images, err := internal.GetFlacImages(file)
		if err != nil {
			return nil, err
		}
		if len(images) == 0 {
			missing = append(missing, file)
		}
	}
	return missing, nil
}

// returns dir - { tag: [val1, val2] }
func getMultiValuedKeys(collectedMetadata map[string]map[string]string, tags []string) map[string]map[string][]string {
	multiValued := make(map[string]map[string][]string)

	// Group by directory instead of individual files
	dirValues := map[string]map[string][]string{}

	for file, metadata := range collectedMetadata {
		// Extract directory from file path
		dir := filepath.Dir(file)

		// Initialize directory entry if not exists
		if _, exists := dirValues[dir]; !exists {
			dirValues[dir] = make(map[string][]string)
		}

		// Process each tag for this file
		for _, tag := range tags {
			value, found := metadata[tag]
			if found {
				// Initialize tag slice if not exists for this directory
				if _, initialized := dirValues[dir][tag]; !initialized {
					dirValues[dir][tag] = make([]string, 0)
				}

				// Add value if not already present (avoid duplicates within directory)
				if !slices.Contains(dirValues[dir][tag], value) {
					dirValues[dir][tag] = append(dirValues[dir][tag], value)
				}
			}
		}
	}

	// Check for multi-valued tags per directory and include the values
	for dir, tagValues := range dirValues {
		multiValuedTagsWithValues := make(map[string][]string)
		for tag, values := range tagValues {
			if len(values) > 1 {
				multiValuedTagsWithValues[tag] = values
			}
		}
		if len(multiValuedTagsWithValues) > 0 {
			multiValued[dir] = multiValuedTagsWithValues
		}
	}

	return multiValued
}

func analyzeAction(action *internal.GenericResult[analyzeResult]) error {
	if flagMetaJsonOutput {
		encoded, err := json.Marshal(action.Data)
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}

	if len(action.Data.MissingCovers) > 0 {
		var data [][]string
		for _, file := range action.Data.MissingCovers {
			data = append(data, []string{file})
		}
		tui.PrintTable(
			"Missing Covers",
			[]string{"File"},
			data,
			tui.TableOpts{},
		)
	}

	// Print Missing Tags table
	if len(action.Data.MissingTags) > 0 {
		var data [][]string
		for file, tags := range action.Data.MissingTags {
			data = append(data, []string{file, strings.Join(tags, ", ")})
		}
		tui.PrintTable(
			"Missing Tags",
			[]string{"File", "Missing Tags"},
			data,
			tui.TableOpts{},
		)
	}

	// Print Multi-Valued Tags table
	if len(action.Data.MultiValuedTags) > 0 {
		var data [][]string
		for file, tagMap := range action.Data.MultiValuedTags {
			for tag, values := range tagMap {
				data = append(data, []string{file, tag, strings.Join(values, ", ")})
			}
		}
		tui.PrintTable(
			"Multi-Valued Tags",
			[]string{"File", "Tag", "Values"},
			data,
			tui.TableOpts{},
		)
	}

	// Print Undesired Tags table
	if len(action.Data.UndesiredTags) > 0 {
		var data [][]string
		for file, tagMap := range action.Data.UndesiredTags {
			for tag, value := range tagMap {
				data = append(data, []string{file, tag, value})
			}
		}
		tui.PrintTable(
			"Undesired Tags",
			[]string{"File", "Tag", "Value"},
			data,
			tui.TableOpts{},
		)
	}

	action.Data.PrintSummary()

	return nil
}

type analyzeResult struct {
	MissingCovers   []string
	MissingTags     map[string][]string
	MultiValuedTags map[string]map[string][]string
	UndesiredTags   map[string]map[string]string
}

func (ar *analyzeResult) PrintSummary() {
	var summaryLines []string

	// Style definitions
	numberStyle := lipgloss.NewStyle().
		Bold(true)

	categoryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	detailStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Underline(true)

	// Missing Covers
	if len(ar.MissingCovers) > 0 {
		count := numberStyle.Render(fmt.Sprintf("%d", len(ar.MissingCovers)))
		category := categoryStyle.Render("missing covers")
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s", count, category))
	}

	// Missing Tags
	if len(ar.MissingTags) > 0 {
		totalMissing := 0
		var examples []string
		for _, tags := range ar.MissingTags {
			totalMissing += len(tags)
			for _, tag := range tags {
				if len(examples) < 3 {
					examples = append(examples, tag)
				}
			}
		}

		count := numberStyle.Render(fmt.Sprintf("%d", totalMissing))
		category := categoryStyle.Render("missing tags")
		detail := detailStyle.Render(fmt.Sprintf("(%s)", strings.Join(examples, ", ")))
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s %s", count, category, detail))
	}

	// Multi-Valued Tags
	if len(ar.MultiValuedTags) > 0 {
		totalMultiValued := 0
		var examples []string
		for _, tagMap := range ar.MultiValuedTags {
			for tag, values := range tagMap {
				totalMultiValued++
				if len(examples) < 2 {
					examples = append(examples, fmt.Sprintf("%s: %s", tag, strings.Join(values, "/")))
				}
			}
		}

		count := numberStyle.Render(fmt.Sprintf("%d", totalMultiValued))
		category := categoryStyle.Render("multi-valued tags")
		detail := detailStyle.Render(fmt.Sprintf("(%s)", strings.Join(examples, ", ")))
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s %s", count, category, detail))
	}

	// Undesired Tags
	if len(ar.UndesiredTags) > 0 {
		totalUndesired := 0
		var examples []string
		for _, tagMap := range ar.UndesiredTags {
			for tag, value := range tagMap {
				totalUndesired++
				if len(examples) < 2 {
					examples = append(examples, fmt.Sprintf("%s: %s", tag, value))
				}
			}
		}

		count := numberStyle.Render(fmt.Sprintf("%d", totalUndesired))
		category := categoryStyle.Render("undesired tags")
		detail := detailStyle.Render(fmt.Sprintf("(%s)", strings.Join(examples, ", ")))
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s %s", count, category, detail))
	}

	if len(summaryLines) > 0 {
		title := titleStyle.Render("Analysis Summary")
		fmt.Println(title)
		for _, line := range summaryLines {
			fmt.Printf("  • %s\n", line)
		}
		fmt.Println()
	} else {
		successStyle := lipgloss.NewStyle().
			Bold(true)
		fmt.Println(successStyle.Render("✓ No issues found!"))
		fmt.Println()
	}
}
