package utils

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type UnstructYaml struct {
	slice *yaml.MapSlice
}

func LoadUnstructYaml(file string) (*UnstructYaml, error) {

	bytes, err := ReadFile(file)
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

func (y *UnstructYaml) Set(path string, value interface{}) error {

	switch value.(type) {
	case int, bool, string:
		return unstructYamlSet(*y.slice, strings.Split(path, "."), value)

	default:
		return fmt.Errorf("unsupported value of type %T", value)

	}
}

func (y *UnstructYaml) Write(file string) error {

	bytes, err := yaml.Marshal(y.slice)
	if err != nil {
		return err
	}

	return WriteFile(bytes, file)
}
