package utils

import (
	"io/ioutil"
	"testing"
)

func TestReadFileFromZip(t *testing.T) {
	cases := []struct {
		description     string
		zipFile         string
		fileToRead      string
		expectedContent string
	}{
		{
			description: "should read file",
			zipFile:     "../../cmd/testdata/rhmi-install-addon-flow.zip",
			fileToRead:  "metadata.json",
			expectedContent: `{
  "name": "rhmi-install-addon-flow",
  "rhmiVersion": "2.4.0-rc1",
  "jobURL": "https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Integreatly/job/rhmi-install-addon-flow/143/"
}
`,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			read, err := ReadFileFromZip(c.zipFile, c.fileToRead)
			if err != nil {
				t.Fatalf("failed to read zip file: %v", err)
			}
			if string(read) != c.expectedContent {
				t.Fatalf("want: %s, got: %s", c.expectedContent, string(read))
			}
		})
	}
}

func TestZipFolder(t *testing.T) {
	cases := []struct {
		description         string
		zipFileAbsolutePath string
		folderToZip         string
	}{
		{
			description:         "should create zip file",
			zipFileAbsolutePath: "/tmp/results.zip",
			folderToZip:         "./testdata/results/",
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			err := ZipFolder(c.folderToZip, c.zipFileAbsolutePath)
			if err != nil {
				t.Fatalf("failed to create zip file: %v", err)
			}
			_, err = ioutil.ReadFile(c.zipFileAbsolutePath)
			if err != nil {
				t.Fatalf("zip file not found: %v", err)
			}
			_, err = ReadFileFromZip(c.zipFileAbsolutePath, "metadata.json")
			if err != nil {
				t.Fatalf("zip file missing expected file: %v", err)
			}

		})
	}
}
