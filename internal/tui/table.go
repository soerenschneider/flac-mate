package tui

import (
	"cmp"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/soerenschneider/flac-mate/internal"
	"golang.org/x/term"
)

type TableOpts struct {
	Wrap      bool
	FullWidth bool
	Style     *func(row, col int) lipgloss.Style
}

var defaultStyle = func(row, col int) lipgloss.Style {
	headerTextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true).
		Align(lipgloss.Center)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 2)

	//highlightStyle := lipgloss.NewStyle().
	//	Foreground(lipgloss.Color("231")). // bright white
	//	Background(lipgloss.Color("161")). // rich red
	//	Padding(0, 2)

	if row == table.HeaderRow {
		return headerTextStyle
	}

	return cellStyle
}

func PrintMetadataTable(metadataList []map[string]string) {
	if len(metadataList) == 0 {
		return
	}

	// Group metadata by directory path
	dirGroups := make(map[string][]map[string]string)

	for _, metadata := range metadataList {
		filepath := metadata["_filepath"]
		if filepath == "" {
			filepath = "Unknown Path"
		}

		// Extract directory from filepath
		lastSlashIndex := strings.LastIndex(filepath, "/")
		var dir string
		if lastSlashIndex == -1 {
			dir = "Root Directory"
		} else {
			dir = filepath[:lastSlashIndex]
			if dir == "" {
				dir = "Root Directory"
			}
		}

		dirGroups[dir] = append(dirGroups[dir], metadata)
	}

	// Sort directory names for consistent output
	var dirNames []string
	for dir := range dirGroups {
		dirNames = append(dirNames, dir)
	}
	sort.Strings(dirNames)

	// Print a table for each directory
	for _, dir := range dirNames {
		dirMetadata := dirGroups[dir]

		// Prepare headers - include filename and all unique tags for this directory
		headers := []string{"File"}
		tagSet := make(map[string]bool)

		// Collect all unique tags from this directory's metadata
		for _, metadata := range dirMetadata {
			for key := range metadata {
				if !strings.HasPrefix(key, "_") {
					tagSet[key] = true
				}
			}
		}

		// If specific tags were requested, use those; otherwise use all found tags
		for tag := range tagSet {
			headers = append(headers, tag)
		}
		sort.Strings(headers[1:]) // Sort tags but keep "File" first

		// Prepare data rows for this directory
		var data [][]string
		for _, metadata := range dirMetadata {
			row := make([]string, len(headers))

			// Set filename (just the basename, not full path)
			if filepath, exists := metadata["_filepath"]; exists {
				row[0] = filepath[strings.LastIndex(filepath, "/")+1:]
			}

			// Set tag values
			for i, header := range headers[1:] {
				if value, exists := metadata[header]; exists {
					row[i+1] = value
				} else {
					row[i+1] = ""
				}
			}
			data = append(data, row)
		}

		// Sort rows by track number if available
		sort.Slice(data, func(i, j int) bool {
			// Find tracknumber column
			trackColIndex := -1
			for idx, header := range headers {
				if header == internal.TagTrackNumber {
					trackColIndex = idx
					break
				}
			}

			if trackColIndex >= 0 && trackColIndex < len(data[i]) && trackColIndex < len(data[j]) {
				return data[i][trackColIndex] < data[j][trackColIndex]
			}

			// Fallback to filename sort
			return data[i][0] < data[j][0]
		})

		// Use directory path as table title
		PrintTable(dir, headers, data, TableOpts{})
	}
}

func PrintTable(tableHeader string, headers []string, data [][]string, opts TableOpts) {
	// Define styles

	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	tableTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).  // magenta-like
		Background(lipgloss.Color("236")). // dark gray
		Padding(0, 2).
		Align(lipgloss.Center).
		MarginBottom(1).
		MarginTop(1)

	// Get terminal width if needed
	var width int
	if opts.FullWidth {
		var err error
		width, _, err = term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			fmt.Println("Error getting terminal size:", err)
			return
		}
	}

	styleFunc := cmp.Or(opts.Style, &defaultStyle)
	// Build the table
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(data...).
		Width(width).
		StyleFunc(*styleFunc).
		Wrap(opts.Wrap)

	// Print header and table
	fmt.Println(tableTitleStyle.Render(" " + tableHeader + " "))
	fmt.Println(t.String())
}
