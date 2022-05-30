package cmd

import (
	"io"
	"os"
	"testing"
)

func TestDoUpdate3scaleBundle(t *testing.T) {
	var (
		validBundleName  = "3scale-operator.v0.9.0"
		validBundleImage = "quay.io/integreatly/3scale-bundle:v0.9.0"
		validBundlesPath = "./testdata/update3scaleBundleTest/bundles.yaml"
		invalidPath      = "./not/a/real/path"
		tmpDir           = "./testdata/update3scaleBundleTest"
	)
	type args struct {
		bundleName  string
		bundleImage string
		bundlesPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid parameters",
			args: args{
				bundleName:  validBundleName,
				bundleImage: validBundleImage,
				bundlesPath: validBundlesPath,
			},
			wantErr: false,
		},
		{
			name: "missing bundle name",
			args: args{
				bundleName:  "",
				bundleImage: validBundleImage,
				bundlesPath: validBundlesPath,
			},
			wantErr: true,
		},
		{
			name: "missing bundle image",
			args: args{
				bundleName:  validBundleName,
				bundleImage: "",
				bundlesPath: validBundlesPath,
			},
			wantErr: true,
		},
		{
			name: "invalid products path",
			args: args{
				bundleName:  validBundleName,
				bundleImage: validBundleImage,
				bundlesPath: invalidPath,
			},
			wantErr: true,
		},
	}
	tmp, err := os.MkdirTemp(tmpDir, "tmp")
	if err != nil {
		t.Errorf("TestDoUpdate3scaleBundle error = %v, failed to initialise", err)
	}
	for _, tt := range tests {
		bundlePath := ""
		if tt.args.bundlesPath == validBundlesPath {
			bundlePath = tmp + "/bundles.yaml"
			copyFileContents(validBundlesPath, bundlePath)
		} else {
			bundlePath = tt.args.bundlesPath
		}

		command := &Update3scaleBundleCommand{
			BundleName:     tt.args.bundleName,
			BundleImage:    tt.args.bundleImage,
			BundleFilePath: bundlePath,
		}

		if err := command.Run(); (err != nil) != tt.wantErr {
			t.Errorf("Update3scaleBundle error = %v, wantErr %v", err, tt.wantErr)
		}

	}
	err = os.RemoveAll(tmp)
	if err != nil {
		t.Logf("Update3scaleBundle error = %v, failed to remove tmp dir %s", err, tmp)
	}
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
