package cmd

import (
	"testing"
)

func TestDoProcessBundle(t *testing.T) {
	var (
		invalidPath          = "./not/a/real/path"
		validBundleImage     = "this-is-a-test-bundle-image"
		validBundleDirectory = "./testdata/processBundleTest/3scale-bundle-extracted"
		validProductsPath    = "./testdata/processBundleTest/products/installation-cpaas.yaml"
		validProductName     = "3scale"
	)
	type args struct {
		bundleImage     string
		bundleDirectory string
		productsPath    string
		productName     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid parameters",
			args: args{
				bundleImage:     validBundleImage,
				bundleDirectory: validBundleDirectory,
				productsPath:    validProductsPath,
				productName:     validProductName,
			},
			wantErr: false,
		},
		{
			name: "missing bundle image",
			args: args{
				bundleImage:     "",
				bundleDirectory: validBundleDirectory,
				productsPath:    validProductsPath,
				productName:     validProductName,
			},
			wantErr: true,
		},
		{
			name: "missing product name",
			args: args{
				bundleImage:     validBundleImage,
				bundleDirectory: validBundleDirectory,
				productsPath:    validProductsPath,
				productName:     "",
			},
			wantErr: true,
		},
		{
			name: "invalid bundle directory",
			args: args{
				bundleImage:     validBundleImage,
				bundleDirectory: invalidPath,
				productsPath:    validProductsPath,
				productName:     validProductName,
			},
			wantErr: true,
		},
		{
			name: "invalid products path",
			args: args{
				bundleImage:     validBundleImage,
				bundleDirectory: validBundleDirectory,
				productsPath:    invalidPath,
				productName:     validProductName,
			},
			wantErr: true,
		},
		{
			name: "invalid product name",
			args: args{
				bundleImage:     validBundleImage,
				bundleDirectory: validBundleDirectory,
				productsPath:    validProductsPath,
				productName:     "foobar",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		updater := &ProductInstallationCompositeUpdater{
			Updaters: []ProductInstallationUpdater{
				&ProductInstallationUpdaterFromValues{
					Bundle:      &tt.args.bundleImage,
					InstallFrom: &bundleInstallFrom,
				},
			},
		}
		if tt.args.bundleDirectory != "" {
			updater.Updaters = append(updater.Updaters, &ProductInstallationUpdaterFromBundle{
				BundleDir: tt.args.bundleDirectory,
			})
		}
		command := &ProcessBundleCommand{
			ProductsInstallationPath: tt.args.productsPath,
			ProductKey:               tt.args.productName,
			Updater:                  updater,
		}

		if err := command.Run(); (err != nil) != tt.wantErr {
			t.Errorf("ProcessBundle error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}
