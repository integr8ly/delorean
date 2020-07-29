package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestPipelineJUnitReport_Run(t *testing.T) {
	dir, err := ioutil.TempDir("", "pipeline-junit-report-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	out := path.Join(dir, "pipeline.xml")
	cmd := &pipelineJUnitReportCmd{
		input:  "../pkg/utils/testdata/jenkins/pipeline-status.json",
		output: out,
	}
	if err := cmd.run(context.TODO()); err != nil {
		t.Fatalf("failed to generate junit report: %v", err)
	}
	content, err := ioutil.ReadFile(out)
	if err != nil {
		t.Fatalf("can not read generated file: %v", err)
	}
	if string(content) == "" {
		t.Fatalf("generated junit file is empty: %s", string(content))
	}
}
