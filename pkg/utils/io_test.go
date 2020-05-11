package utils

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

type sampleYAML struct {
	Name   string         `yaml:"name,omitempty"`
	Value  int            `yaml:"value,omitempty"`
	Array  []int          `yaml:"array,omitempty"`
	Object map[string]int `yaml:"object,omitempty"`
}

type sampleJSON struct {
	Name   string         `json:"name,omitempty"`
	Value  int            `json:"value,omitempty"`
	Array  []int          `json:"array,omitempty"`
	Object map[string]int `json:"object,omitempty"`
}

func TestPopulateObjectFromYAML(t *testing.T) {
	content := `
name: test
value: 1
array:
- 1
- 2
object:
  key: 1
`
	tmp, err := ioutil.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	// Prepare the file to copy
	src := path.Join(tmp, "src.yaml")
	writeFile(t, src, content)
	obj := &sampleYAML{}
	err = PopulateObjectFromYAML(src, obj)
	if err != nil {
		t.Errorf("failed to parse yaml file: %v", err)
	}
}

func TestWriteObjectToYAML(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	obj := &sampleYAML{
		Name:   "test",
		Value:  1,
		Array:  []int{1, 2},
		Object: map[string]int{"key": 1},
	}
	dest := path.Join(tmp, "out.yaml")
	err = WriteObjectToYAML(obj, dest)
	if err != nil {
		t.Errorf("failed to write yaml file: %v", err)
	}
	content := readFile(t, dest)
	expected := `Array:
- 1
- 2
Name: test
Object:
  key: 1
Value: 1
`
	if content != expected {
		t.Errorf("expected output is not valid. Expected:\n%s\n Actual:\n%s\n", expected, content)
	}
}

func TestWriteObjectToJSON(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	obj := &sampleJSON{
		Name:   "test",
		Value:  1,
		Array:  []int{1, 2},
		Object: map[string]int{"key": 1},
	}
	dest := path.Join(tmp, "out.json")
	err = WriteObjectToJSON(obj, dest)
	if err != nil {
		t.Errorf("failed to write json file: %v", err)
	}
	content := readFile(t, dest)
	expected := `{"name":"test","value":1,"array":[1,2],"object":{"key":1}}`
	if content != expected {
		t.Errorf("expected output is not valid. Expected:\n%s\n Actual:\n%s\n", expected, content)
	}
}
