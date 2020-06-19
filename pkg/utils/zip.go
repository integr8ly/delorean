package utils

import (
	"archive/zip"
	"io/ioutil"
)

func ReadFileFromZip(zipfile string, fileName string) ([]byte, error) {
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == fileName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return ioutil.ReadAll(rc)
		}
	}
	return nil, nil
}
