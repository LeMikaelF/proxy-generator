package flags

import (
	"errors"
	"flag"
	"os"
	"testing"
)

func TestCsvToMap(t *testing.T) {
	tests := []struct {
		name string
		csv  string
		want map[string]bool
	}{
		{
			name: "Empty string",
			csv:  "",
			want: map[string]bool{},
		},
		{
			name: "Single element",
			csv:  "method1",
			want: map[string]bool{"method1": true},
		},
		{
			name: "Multiple elements",
			csv:  "method1,method2,method3",
			want: map[string]bool{"method1": true, "method2": true, "method3": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := csvToMap(tt.csv)
			if !compareMaps(got, tt.want) {
				t.Errorf("CsvToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	// Save original args and reset to original state after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tests := []struct {
		name    string
		args    []string
		want    *ParsedFlags
		wantErr error
	}{
		{
			name:    "No flags provided",
			args:    []string{"cmd"},
			want:    nil,
			wantErr: errors.New("usage: go run github.com/LeMikaelF/proxy-generator --type <type> [--passthrough-methods <method1,method2>]"),
		},
		{
			name: "Only type provided",
			args: []string{"cmd", "--type", "MyType"},
			want: &ParsedFlags{
				TypeName:           "MyType",
				PassthroughMethods: map[string]bool{},
				PackageName:        os.Getenv("GOPACKAGE"),
			},
			wantErr: nil,
		},
		{
			name: "Type and passthrough methods provided",
			args: []string{"cmd", "--type", "MyType", "--passthrough-methods", "method1,method2"},
			want: &ParsedFlags{
				TypeName: "MyType",
				PassthroughMethods: map[string]bool{
					"method1": true,
					"method2": true,
				},
				PackageName: os.Getenv("GOPACKAGE"),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = tt.args

			got, err := Parse()

			if (tt.wantErr == nil && err != nil) ||
				(tt.wantErr != nil && err == nil) ||
				(err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !compareParsedFlags(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareMaps(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func compareParsedFlags(a, b *ParsedFlags) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.TypeName != b.TypeName || a.PackageName != b.PackageName || !compareMaps(a.PassthroughMethods, b.PassthroughMethods) {
		return false
	}

	return true

}
