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
		expectedSkips      int
	}{
		{
			description:        "success",
			pipelineStatusFile: "./testdata/jenkins/pipeline-status.json",
			expectedFailures:   1,
			expectedSkips:      7,
		},
		{
			description:        "Aborted",
			pipelineStatusFile: "./testdata/jenkins/aborted-pipeline-status.json",
			expectedFailures:   1,
			expectedSkips:      6,
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

			skipCount := countSkipped(ts.TestCases)
			if skipCount != c.expectedSkips {
				t.Fatalf("number of failures doesn't match. expected: %d got: %d", c.expectedSkips, skipCount)
			}

			w := bytes.NewBufferString("")
			err = suites.WriteXML(w)
			if err != nil {
				t.Fatalf("failed to write suites due to error: %v", err)
			}
		})
	}
}

func countSkipped(cases []JUnitTestCase) int {
	skipped := 0

	for _, ts := range cases {
		if ts.SkipMessage != nil {
			skipped++
		}
	}

	return skipped
}
