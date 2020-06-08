package utils

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// UnstructYaml see LoadUnstructYaml
type UnstructYaml struct {
	slice *yaml.MapSlice
}

// LoadUnstructYaml parse the passed yaml file and create an UnstructYaml object
// that can be used to change one or more value of the yaml file without changing
// the order of the fields or removing unknow fileds
func LoadUnstructYaml(file string) (*UnstructYaml, error) {

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	slice := &yaml.MapSlice{}
	err = yaml.Unmarshal(bytes, slice)
	if err != nil {
		return nil, err
	}

	return &UnstructYaml{slice}, nil
}

// Set will modify a single filed of the UnstructYaml (see LoadUnstructYaml)
func (y *UnstructYaml) Set(path string, value interface{}) error {

	switch value.(type) {
	case int, bool, string:
		return unstructYamlSet(*y.slice, strings.Split(path, "."), value)

	default:
		return fmt.Errorf("unsupported value of type %T", value)

	}
}

// Write the UnstructYaml to the passed file (see LoadUnstructYaml)
func (y *UnstructYaml) Write(file string) error {

	bytes, err := yaml.Marshal(y.slice)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(file, bytes, 0644)
}

func unstructYamlSet(obj interface{}, path []string, value interface{}) error {

	key, path := path[0], path[1:]
	switch typ := obj.(type) {
	case yaml.MapSlice:
		for i, item := range typ {
			if item.Key == key {
				if len(path) > 0 {
					return unstructYamlSet(typ[i].Value, path, value)
				}

				typ[i].Value = value
				return nil
			}
		}
		return fmt.Errorf("failed to find key %s in object %+v", key, typ)

	case []interface{}:
		index, err := strconv.ParseInt(key, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to convert key %s in int in object %+v", key, typ)
		}

		if len(typ) < int(index) {
			return fmt.Errorf("index %d is out of range in object %+v", index, typ)
		}

		if len(path) > 0 {
			return unstructYamlSet(typ[index], path, value)
		}

		typ[index] = value
		return nil

	case *interface{}:
		return fmt.Errorf("unsported type %T in object %+v", typ, *typ)

	default:
		return fmt.Errorf("unknow type %T in object %+v", typ, typ)
	}
}
