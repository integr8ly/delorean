package utils

import (
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
)

// PopulateObjectFromYAML will read the content from the given yaml file and use it to unmarshal the given object
func PopulateObjectFromYAML(yamlFile string, obj interface{}) error {
	read, err := os.Open(yamlFile)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadAll(read)

	err = read.Close()
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, obj)
	if err != nil {
		return err
	}
	return nil
}

// WriteObjectToYAML will marshal the given object and write to the given yaml file
func WriteObjectToYAML(obj interface{}, yamlFile string) error {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	// truncate the existing file
	write, err := os.Create(yamlFile)
	if err != nil {
		return err
	}

	_, err = write.Write(bytes)
	if err != nil {
		return err
	}

	err = write.Close()
	if err != nil {
		return err
	}
	return nil
}
