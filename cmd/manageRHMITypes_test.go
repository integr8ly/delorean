package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/integr8ly/delorean/pkg/utils"
)

func TestManageRHMITypes(t *testing.T) {
	type args struct {
		ctx     context.Context
		cmdOpts *manageTypesCmdOptions
	}
	tests := []struct {
		name    string
		args    args
		verify  func(t *testing.T, directory string, product string, version string, majmin string) error
		wantErr bool
		majmin  string
	}{
		{
			name: "test setting 3scale major version",
			args: args{context.TODO(), &manageTypesCmdOptions{
				filepath: "",
				product:  "3scale",
				version:  "9.9.9",
			}},
			wantErr: false,
			verify: func(t *testing.T, directory string, product string, version string, majmin string) error {
				return verifyTypesFile(t, directory, product, version, "")
			},
			majmin: "major",
		},
		{
			name: "test setting amq-online major version",
			args: args{context.TODO(), &manageTypesCmdOptions{
				filepath: "",
				product:  "amq-online",
				version:  "9.9.9",
			}},
			wantErr: false,
			verify: func(t *testing.T, directory string, product string, version string, majmin string) error {
				return verifyTypesFile(t, directory, product, version, "")
			},
			majmin: "major",
		},
		{
			name: "test setting 3scale minor version",
			args: args{context.TODO(), &manageTypesCmdOptions{
				filepath: "",
				product:  "3scale",
				version:  "9.10.0",
			}},
			wantErr: false,
			verify: func(t *testing.T, directory string, product string, version string, majmin string) error {
				return verifyTypesFile(t, directory, product, version, "")
			},
			majmin: "minor",
		},
		{
			name: "test setting amq-online minor version",
			args: args{context.TODO(), &manageTypesCmdOptions{
				filepath: "",
				product:  "amq-online",
				version:  "9.10.0",
			}},
			wantErr: false,
			verify: func(t *testing.T, directory string, product string, version string, majmin string) error {
				return verifyTypesFile(t, directory, product, version, "")
			},
			majmin: "minor",
		},
	}
	for _, tt := range tests {
		testDir, err := os.MkdirTemp(os.TempDir(), "test-")
		if err != nil {
			t.Fatal(err)
		}
		err = utils.CopyDirectory("./testdata/manageRHMITypes", testDir)
		if err != nil {
			t.Fatal(err)
		}
		tt.args.cmdOpts.filepath = path.Join(testDir, "rhmi_types")
		t.Run(tt.name, func(t *testing.T) {
			if err := SetVersion(tt.args.cmdOpts.filepath, tt.args.cmdOpts.product, tt.args.cmdOpts.version); (err != nil) != tt.wantErr {
				t.Errorf("SetVersion() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if tt.verify != nil {
					if err := tt.verify(t, tt.args.cmdOpts.filepath, tt.args.cmdOpts.product, tt.args.cmdOpts.version, tt.majmin); err != nil {
						fmt.Println("d: ", tt.args.cmdOpts.filepath)
						t.Fatalf("verification failed due to error: %v", err)
					}
				}
			}

			err = os.RemoveAll(testDir)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func verifyTypesFile(t *testing.T, filepath, product string, version string, majmin string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, _ := io.ReadAll(f)
	product = PrepareProductName(product)

	var ReOperatorVersion = regexp.MustCompile(`OperatorVersion` + product + `.*`)

	operatorVersion := ReOperatorVersion.FindString(string(bytes))

	// Remove the "'s so it can validate against the version
	ovs := strings.Split(operatorVersion, "=")[1]
	ovs = strings.ReplaceAll(ovs, "\"", "")
	ovs = strings.TrimSpace(ovs)

	if majmin == "major" {
		if ovs != version {
			t.Errorf("error found incorrect version string, wanted = %s, got %s", ovs, version)
		}
	}

	if majmin == "minor" {
		if ovs != version {
			t.Errorf("error found incorrect version string, wanted = %s, got %s", ovs, version)
		}
	}

	return nil
}
