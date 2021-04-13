package utils

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// JUnitTestSuites is a collection of JUnit test suites.
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite is a single JUnit test suite which may contain many
// testcases.
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Time       string          `xml:"time,attr"`
	Name       string          `xml:"name,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase is a single test case with its result.
type JUnitTestCase struct {
	XMLName     xml.Name          `xml:"testcase"`
	Classname   string            `xml:"classname,attr"`
	Name        string            `xml:"name,attr"`
	Time        string            `xml:"time,attr"`
	SkipMessage *JUnitSkipMessage `xml:"skipped,omitempty"`
	Failure     *JUnitFailure     `xml:"failure,omitempty"`
}

// JUnitSkipMessage contains the reason why a testcase was skipped.
type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

// JUnitProperty represents a key/value pair used to define properties.
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitFailure contains data related to a failed test.
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

func (s *JUnitTestSuites) WriteXML(w io.Writer) error {
	bytes, err := xml.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(w)
	writer.WriteString(xml.Header)
	writer.Write(bytes)
	writer.WriteByte('\n')
	writer.Flush()
	return nil
}

type PipelineRunStageError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type PipelineRunStage struct {
	Name              string                `json:"name"`
	StartTimeInMillis int64                 `json:"startTimeMillis"`
	DurationInMills   int64                 `json:"durationMillis"`
	Status            string                `json:"status"`
	Error             PipelineRunStageError `json:"error,omitempty"`
}

type PipelineRun struct {
	Name              string             `json:"name"`
	Status            string             `json:"status"`
	StartTimeInMillis int64              `json:"startTimeMillis"`
	EndTimeInMillis   int64              `json:"endTimeMillis"`
	DurationInMillis  int64              `json:"durationMillis"`
	Stages            []PipelineRunStage `json:"stages"`
}

const (
	TestSuiteName             = "pipeline-status"
	PipelineRunStatusFailed   = "FAILED"
	PipelineRunStatusAborted  = "ABORTED"
	PipelineRunStatusUnstable = "UNSTABLE"
)

// ToJUnitSuites will convert the status of the pipeline run into JUnit test suites
func (p *PipelineRun) ToJUnitSuites(filter string) (*JUnitTestSuites, error) {
	suites := &JUnitTestSuites{
		Suites: []JUnitTestSuite{},
	}
	ts := JUnitTestSuite{
		Tests:     len(p.Stages),
		Time:      formatMillis(p.DurationInMillis),
		Name:      TestSuiteName,
		TestCases: []JUnitTestCase{},
		Failures:  0,
	}

	failure := false
	aborted := false
	for _, s := range p.Stages {
		tc := JUnitTestCase{
			Name:    s.Name,
			Time:    formatMillis(s.DurationInMills),
			Failure: nil,
		}

		switch s.Status {
		case PipelineRunStatusFailed:
			if !failure && s.Status == PipelineRunStatusFailed {
				ts.Failures++
				tc.Failure = &JUnitFailure{
					Message: s.Error.Message,
					Type:    s.Error.Type,
				}
				failure = true
			} else {
				tc.SkipMessage = &JUnitSkipMessage{
					Message: s.Error.Message,
				}
			}
		case PipelineRunStatusAborted:
			if !aborted {
				ts.Failures++
				tc.Failure = &JUnitFailure{
					Message: s.Error.Message,
					Type:    s.Error.Type,
				}
				aborted = true
			} else {
				tc.SkipMessage = &JUnitSkipMessage{
					Message: s.Error.Message,
				}
			}
		case PipelineRunStatusUnstable:
			ts.Failures++
			tc.Failure = &JUnitFailure{
				Message: s.Error.Message,
				Type:    s.Error.Type,
			}
		}

		// If creating a report only from stages with a specific prefix,
		// skip adding stage names to the report that do not match the prefix
		if filter != "" && !strings.Contains(s.Name, filter) {
			continue
		}

		ts.TestCases = append(ts.TestCases, tc)
	}

	suites.Suites = append(suites.Suites, ts)
	return suites, nil
}

func formatMillis(t int64) string {
	d := time.Duration(t) * time.Millisecond
	return fmt.Sprintf("%.3f", d.Seconds())
}
