package utils

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestOC_Run(t *testing.T) {
	oc := &OC{executable: "ls"}
	err := oc.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOC_RunWithOutputFile(t *testing.T) {
	oc := &OC{executable: "ls"}
	dir, err := ioutil.TempDir("/tmp", "oc-test")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	out := path.Join(dir, "out.txt")
	err = oc.RunWithOutputFile(out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output, err := ioutil.ReadFile(out)
	if err != nil {
		t.Fatalf("can not read file: %v", err)
	}
	if len(string(output)) == 0 {
		t.Fatalf("output is empty")
	}
}
