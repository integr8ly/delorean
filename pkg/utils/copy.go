package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// CopyDirectory will copy all files from the src directory into the dest directory
// and it will create the dest directory if it doesn't exists
//
// - It doesn't copy subdirectories
// - It doesn't copy symlinks
// - It doesn't preserve owners and permissions while copying
// - It doesn't remove filese from the dest directory that are not in the src directory
func CopyDirectory(src, dest string) error {
	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read the directory %s: %s", src, err)
	}

	// create the directory if it doesn't exists
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		err = os.Mkdir(dest, 0755)
		if err != nil {
			return fmt.Errorf("failed to create the directory %s: %s", dest, err)
		}
	}

	for _, entry := range entries {
		srcFile := filepath.Join(src, entry.Name())
		destFile := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(srcFile)
		if err != nil {
			return fmt.Errorf("failed to retrieve the stats of the file %s: %s", srcFile, err)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			err = CopyDirectory(srcFile, destFile)
			if err != nil {
				return fmt.Errorf("failed to copy the directory form %s to %s: %s", srcFile, destFile, err)
			}
		case os.ModeSymlink:
			return fmt.Errorf("unxepcted symlink %s to coyp", srcFile)
		default:
			err = CopyFile(srcFile, destFile)
			if err != nil {
				return fmt.Errorf("failed to copy file form %s to %s: %s", srcFile, destFile, err)
			}
		}
	}

	return nil
}

// CopyFile will copy the src file to the dest file, if the dest file already exists
// it will overwrite it, otherwise it will create a new one
//
// - It doesn't preserve owners and permissions while copying
func CopyFile(src, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create the file %s: %s", dest, err)
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to read the file %s: %s", dest, err)
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("failed to copy the file content: %s", err)
	}

	return nil
}
