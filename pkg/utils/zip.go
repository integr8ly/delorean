package utils

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"strings"
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

// folder - folder to be zipped
// zipFile - absolute path to the zip file to be generated
func ZipFolder(folder string, zipFile string) error {

	outFile, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	zipFilePath := strings.Split(zipFile, "/")
	err = addFiles(w, folder, "", zipFilePath[len(zipFilePath)-1])
	if err != nil {
		return err
	}

	err = w.Close()
	return err
}

func addFiles(w *zip.Writer, basePath, baseInZip string, zipFile string) error {

	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			// skipping zip file to be generated (allow creation of zip file inside zipped folder)
			if strings.EqualFold(zipFile, file.Name()) {
				continue
			}

			dat, err := ioutil.ReadFile(basePath + file.Name())
			if err != nil {
				return err
			}

			f, err := w.Create(baseInZip + file.Name())
			if err != nil {
				return err
			}

			_, err = f.Write(dat)
			if err != nil {
				return err
			}
		} else if file.IsDir() {

			// Recurse
			newBase := basePath + file.Name() + "/"
			addFiles(w, newBase, baseInZip+file.Name()+"/", zipFile)
		}
	}
	return nil
}
