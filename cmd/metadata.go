package cmd

import (
	"github.com/soerenschneider/flac-mate/internal"
	"github.com/spf13/cobra"
)

var (
	defaultUniformCmdTags = []string{internal.TagArtist, internal.TagAlbum, internal.TagDate, internal.TagGenre}
	defaultCleansedTags   = []string{internal.TagArtist, internal.TagAlbum, internal.TagDate, internal.TagGenre, internal.TagTrackNumber, internal.TagTitle, internal.TagDiscNumber}

	// these tags are not safe for being set recursively and indicate misusage
	unsafeRecursiveTags = []string{
		internal.TagTitle,
		internal.TagTrackNumber,
	}

	flagMetaReadTags    []string
	flagMetaWriteData   map[string]string
	flagMetaUniformTags []string
	flagMetaPictureFile string
	flagMetaWriteForce  bool
	flagMetaJsonOutput  bool
)

// CLI command structure
var metadataCmd = &cobra.Command{
	Use: "metadata",
	Aliases: []string{
		"meta",
	},
	Short: "A tool for managing FLAC metadata",
}

func init() {
	RootCmd.AddCommand(metadataCmd)
}
