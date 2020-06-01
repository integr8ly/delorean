package utils

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"
)

const yamlTestData = "testdata/yaml"

func TestUnstructYamlSet(t *testing.T) {
	cases := []struct {
		description string
		file        string
		params      struct {
			path  string
			value interface{}
		}
		expectError bool
		expect      string
	}{
		{
			description: "set .test in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "test",
				value: "else",
			},
			expectError: false,
			expect: strings.TrimLeft(`
test: else
array:
- data
- two
first:
  second: stuff
bigger:
- name: step
- name: stuff
  other: 23
`, "\n"),
		},
		{
			description: "set .array.0 in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "array.0",
				value: "one",
			},
			expectError: false,
			expect: strings.TrimLeft(`
test: data
array:
- one
- two
first:
  second: stuff
bigger:
- name: step
- name: stuff
  other: 23
`, "\n"),
		},
		{
			description: "set .bigger.1.other in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "bigger.1.other",
				value: "twentythree",
			},
			expectError: false,
			expect: strings.TrimLeft(`
test: data
array:
- data
- two
first:
  second: stuff
bigger:
- name: step
- name: stuff
  other: twentythree
`, "\n"),
		},
		{
			description: "set .test to bool in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "test",
				value: false,
			},
			expectError: false,
			expect: strings.TrimLeft(`
test: false
array:
- data
- two
first:
  second: stuff
bigger:
- name: step
- name: stuff
  other: 23
`, "\n"),
		},
		{
			description: "set .first.second to int in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "first.second",
				value: 44,
			},
			expectError: false,
			expect: strings.TrimLeft(`
test: data
array:
- data
- two
first:
  second: 44
bigger:
- name: step
- name: stuff
  other: 23
`, "\n"),
		},
		{
			description: "fail to set .bigger.4 in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "bigger.4",
				value: 0,
			},
			expectError: true,
			expect:      "",
		},
		{
			description: "fail to set .test to struct in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "test",
				value: struct{ foo string }{foo: "bar"},
			},
			expectError: true,
			expect:      "",
		},
		{
			description: "fail to set .random in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "random",
				value: 0,
			},
			expectError: true,
			expect:      "",
		},
		{
			description: "fail to set . in sample-one.yaml",
			file:        path.Join(yamlTestData, "sample-one.yaml"),
			params: struct {
				path  string
				value interface{}
			}{
				path:  "",
				value: 0,
			},
			expectError: true,
			expect:      "",
		},
	}

	for _, c := range cases {

		t.Run(c.description, func(t *testing.T) {
			y, err := LoadUnstructYaml(c.file)
			if err != nil {
				t.Fatalf("failed to LoadUnstructYaml with error: %s", err)
			}

			err = y.Set(c.params.path, c.params.value)
			if err != nil && c.expectError {
				return
			}
			if err != nil {
				t.Fatalf("failed to Set value with error: %s", err)
			}

			tmpdir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("failed to create tmpdir with error: %s", err)
			}
			outFile := path.Join(tmpdir, "result.yaml")

			err = y.Write(outFile)
			if err != nil {
				t.Fatalf("failed to write UnstructYaml with error: %s", err)
			}

			got, err := ReadFile(outFile)
			if err != nil {
				t.Fatalf("failed to ReadFile with error: %s", err)
			}

			if string(got) != c.expect {
				t.Fatalf("failed to verify test data\nexpected:\n%s\ngot:\n%s", c.expect, string(got))
			}
		})

	}
}
