package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	MappingFile = "image_mirror_mapping"
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
	return ioutil.WriteFile(yamlFile, bytes, 0644)
}

func WriteK8sObjectToYAML(obj interface{}, yamlFile string) error {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	deleteKeys := []string{"status", "creationTimestamp"}
	for _, dk := range deleteKeys {
		deleteKeyFromUnstructured(u, dk)
	}

	bytes, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(yamlFile, bytes, 0644)
}

// WriteObjectToJSON will marshal the given object and write to the given json file
func WriteObjectToJSON(obj interface{}, jsonFile string) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// truncate the existing file
	write, err := os.Create(jsonFile)
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

//https://github.com/operator-framework/operator-sdk/blob/master/internal/util/k8sutil/object.go
func deleteKeyFromUnstructured(u map[string]interface{}, key string) {
	if _, ok := u[key]; ok {
		delete(u, key)
		return
	}

	for _, v := range u {
		switch t := v.(type) {
		case map[string]interface{}:
			deleteKeyFromUnstructured(t, key)
		case []interface{}:
			for _, ti := range t {
				if m, ok := ti.(map[string]interface{}); ok {
					deleteKeyFromUnstructured(m, key)
				}
			}
		}
	}
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func WriteToFile(writePath string, content []string) error {
	return ioutil.WriteFile(writePath, []byte(strings.Join(content, "\n")), 0644)
}
