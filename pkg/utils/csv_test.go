package utils

import (
	"reflect"
	"testing"

	"github.com/blang/semver"
	"github.com/operator-framework/operator-registry/pkg/registry"
)

func TestVerifyManifestDirs(t *testing.T) {
	type args struct {
		dirs []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid manifest dir",
			args:    args{[]string{"./testdata/validManifests/3scale"}},
			wantErr: false,
		},
		{
			name:    "multiple valid manifest dirs",
			args:    args{[]string{"./testdata/validManifests/3scale", "./testdata/validManifests/3scale2"}},
			wantErr: false,
		},
		{
			name:    "invalid manifest dir no package.yaml",
			args:    args{[]string{"./testdata"}},
			wantErr: true,
		},
		{
			name:    "invalid manifest dir missing dir",
			args:    args{[]string{"./testdataaaaaaa"}},
			wantErr: true,
		},
		{
			name:    "multiple  invalid",
			args:    args{[]string{"./testdata", "./testdata/validManifests/3scale"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyManifestDirs(tt.args.dirs...); (err != nil) != tt.wantErr {
				t.Errorf("VerifyManifestDirs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPackageManifest(t *testing.T) {
	type args struct {
		packageDir string
	}
	tests := []struct {
		name    string
		args    args
		want    *registry.PackageManifest
		want1   string
		wantErr bool
	}{
		{
			name:    "valid package dir",
			args:    args{"./testdata/validManifests/3scale"},
			wantErr: false,
			want: &registry.PackageManifest{
				PackageName: "rhmi-3scale",
				Channels: []registry.PackageChannel{
					{
						Name:           "rhmi",
						CurrentCSVName: "3scale-operator.v0.4.0",
					},
				},
			},
			want1: "testdata/validManifests/3scale/3scale.package.yaml",
		},
		{
			name:    "valid package dir 2",
			args:    args{"./testdata/validManifests/3scale2"},
			wantErr: false,
			want: &registry.PackageManifest{
				PackageName: "rhmi-3scale",
				Channels: []registry.PackageChannel{
					{
						Name:           "rhmi",
						CurrentCSVName: "3scale-operator.v0.5.0",
					},
				},
			},
			want1: "testdata/validManifests/3scale2/3scale.package.yaml",
		},
		{
			name:    "invalid package dir",
			args:    args{"./testdata"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetPackageManifest(tt.args.packageDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPackageManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPackageManifest() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetPackageManifest() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGetSortedCSVNames(t *testing.T) {
	sortedcsvs := []csvName{
		{
			Name: "3scale-operator.v0.4.0",
			Version: semver.Version{
				Major: 0,
				Minor: 4,
				Patch: 0,
			},
		},
		{
			Name: "3scale-operator.v0.5.0",
			Version: semver.Version{
				Major: 0,
				Minor: 5,
				Patch: 0,
			},
		},
	}
	type args struct {
		packageDir string
	}
	tests := []struct {
		name    string
		args    args
		want    csvNames
		wantErr bool
	}{
		{
			name:    "valid get sorted dir",
			args:    args{"./testdata/validManifests/3scale2"},
			want:    sortedcsvs,
			wantErr: false,
		},
		{
			name:    "invalid package dir",
			args:    args{"./testdata/validManifests/somebaddir"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSortedCSVNames(tt.args.packageDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got != nil {
					if got[0].Name != tt.want[0].Name {
						t.Errorf("GetCurrentCSV() got1 = %v, want %v", got, tt.want)
					}
				} else {
					t.Errorf("GetCurrentCSV() got = %v", got)
				}
			}
		})
	}

}

func TestGetCurrentCSV(t *testing.T) {
	type args struct {
		packageDir string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name:    "valid package dir",
			args:    args{"./testdata/validManifests/3scale"},
			wantErr: false,
			want:    "3scale-operator.v0.4.0",
			want1:   "testdata/validManifests/3scale/0.4.0/3scale-operator.v0.4.0.clusterserviceversion.yaml",
		},
		{
			name:    "valid package dir 2",
			args:    args{"./testdata/validManifests/3scale2"},
			wantErr: false,
			want:    "3scale-operator.v0.5.0",
			want1:   "testdata/validManifests/3scale2/0.5.0/3scale-operator.v0.5.0.clusterserviceversion.yaml",
		},
		{
			name:    "invalid package dir",
			args:    args{"./testdata"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetCurrentCSV(tt.args.packageDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got != nil {
					if got.Name != tt.want {
						t.Errorf("GetCurrentCSV() got1 = %v, want %v", got.Name, tt.want)
					}
				} else {
					t.Errorf("GetCurrentCSV() got = %v", got)
				}
			}

			if got1 != tt.want1 {
				t.Errorf("GetCurrentCSV() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestReadCSVFromBundleDirectory(t *testing.T) {
	type args struct {
		bundleDir string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name:    "valid bundle dir",
			args:    args{"./testdata/validManifests/3scale/0.4.0"},
			wantErr: false,
			want:    "3scale-operator.v0.4.0",
			want1:   "testdata/validManifests/3scale/0.4.0/3scale-operator.v0.4.0.clusterserviceversion.yaml",
		},
		{
			name:    "invalid bundle dir",
			args:    args{"./testdata/validManifests/3scale"},
			wantErr: true,
		},
		{
			name:    "invalid dir",
			args:    args{"./testdataaaaaaaaaa"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ReadCSVFromBundleDirectory(tt.args.bundleDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadCSVFromBundleDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got != nil {
					if got.Name != tt.want {
						t.Errorf("ReadCSVFromBundleDirectory() got1 = %v, want %v", got.Name, tt.want)
					}
				} else {
					t.Errorf("ReadCSVFromBundleDirectory() got = %v", got)
				}
			}
			if got1 != tt.want1 {
				t.Errorf("ReadCSVFromBundleDirectory() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
