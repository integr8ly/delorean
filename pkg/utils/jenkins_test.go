package utils

import (
	"bytes"
	"testing"
)

func TestPipelineRun_ToJUnitSuites(t *testing.T) {
	cases := []struct {
		description        string
		pipelineStatusFile string
		expectedFailures   int
	}{
		{
			description:        "success",
			pipelineStatusFile: "./testdata/jenkins/pipeline-status.json",
			expectedFailures:   8,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			p := &PipelineRun{}
			err := PopulateObjectFromJSON(c.pipelineStatusFile, p)
			if err != nil {
				t.Fatalf("failed to parse pipeline status file: %v", err)
			}
			suites, _ := p.ToJUnitSuites()
			ts := suites.Suites[0]
			if ts.Name != TestSuiteName {
				t.Fatalf("name doesn't match. expected: %s got: %s", TestSuiteName, ts.Name)
			}
			if len(ts.TestCases) != len(p.Stages) {
				t.Fatalf("number of test cases does't match. expected: %d got: %d", len(p.Stages), len(ts.TestCases))
			}
			if ts.Failures != c.expectedFailures {
				t.Fatalf("number of failures doesn't match. expected: %d got: %d", c.expectedFailures, ts.Failures)
			}

			w := bytes.NewBufferString("")
			err = suites.WriteXML(w)
			if err != nil {
				t.Fatalf("failed to write suites due to error: %v", err)
			}
		})
	}
}
