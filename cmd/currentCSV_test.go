package cmd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDoCurrentCSV(t *testing.T) {
	type args struct {
		ctx     context.Context
		cmdOpts *currentCSVFlags
	}
	tests := []struct {
		name    string
		args    args
		verify  func(t *testing.T, output string) error
		wantErr bool
	}{
		{
			name: "test current csv CRW",
			args: args{context.TODO(), &currentCSVFlags{
				directory: "../pkg/utils/testdata/validManifests/crw",
			}},
			wantErr: false,
			verify: func(t *testing.T, output string) error {
				return verifyCSVJson(t, output, "2.1.1")
			},
		},
		{
			name: "test current csv 3scale",
			args: args{context.TODO(), &currentCSVFlags{
				directory: "../pkg/utils/testdata/validManifests/3scale",
			}},
			wantErr: false,
			verify: func(t *testing.T, output string) error {
				return verifyCSVJson(t, output, "0.4.0")
			},
		},
		{
			name: "test current csv 3scale2",
			args: args{context.TODO(), &currentCSVFlags{
				directory: "../pkg/utils/testdata/validManifests/3scale2",
			}},
			wantErr: false,
			verify: func(t *testing.T, output string) error {
				return verifyCSVJson(t, output, "0.5.0")
			},
		},
		{
			name: "test current csv invalid directory)",
			args: args{context.TODO(), &currentCSVFlags{
				directory: "../pkg/utils/testdata",
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			testDir, err := ioutil.TempDir(os.TempDir(), "test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(testDir)
			tt.args.cmdOpts.output = filepath.Join(testDir, "testcsv.json")

			if err := DoCurrentCSV(tt.args.ctx, tt.args.cmdOpts); (err != nil) != tt.wantErr {
				t.Errorf("DoCurrentCSV() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if tt.verify != nil {
					if err := tt.verify(t, tt.args.cmdOpts.output); err != nil {
						t.Fatalf("verification failed due to error: %v", err)
					}
				}
			}
		})
	}
}

func verifyCSVJson(t *testing.T, output, expectedVersion string) error {
	_, err := os.Stat(output)
	if err != nil {
		return err
	}
	if filepath.Base(output) != "testcsv.json" {
		return err
	}

	jsonFile, err := os.Open(output)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	bytes, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(bytes), &result)
	if err != nil {
		return err
	}

	csvSpec := result["spec"].(map[string]interface{})
	foundVersion := csvSpec["version"]
	if foundVersion != expectedVersion {
		t.Errorf("error reading json, wanted = %s, got %v", expectedVersion, foundVersion)
	}
	return nil
}
