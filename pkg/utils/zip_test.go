package utils

import "testing"

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
