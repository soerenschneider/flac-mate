package internal

import (
	"reflect"
	"testing"
)

func TestFetchMetadata(t *testing.T) {
	type args struct {
		filepath string
		tags     []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "",
			args: args{
				filepath: "../test/flacs/tests_populated.flac",
				tags:     nil,
			},
			want: map[string]string{
				"_filepath":   "../test/flacs/tests_populated.flac",
				"ALBUM":       "Album",
				"ARTIST":      "Artist",
				"COMPOSER":    "Composer",
				"DATE":        "2000",
				"GENRE":       "Genre",
				"TITLE":       "Title",
				"TRACKNUMBER": "01",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchMetadata(tt.args.filepath, tt.args.tags, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FetchMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}
