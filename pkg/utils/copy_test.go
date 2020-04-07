package utils

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func writeFile(t *testing.T, file string, content string) {
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, file string) string {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func TestCopyFile(t *testing.T) {

	content := "Some test content\nfor our file"

	tmp, err := ioutil.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	// Prepare the file to copy
	src := path.Join(tmp, "src.txt")
	writeFile(t, src, content)

	// Copy the file
	dest := path.Join(tmp, "dest.txt")

	err = CopyFile(src, dest)
	if err != nil {
		t.Fatalf("failed to copy the file %s to %s: %s", src, dest, err)
	}

	// Assert that the content of the dest file is equal to the src file
	if result := readFile(t, dest); result != content {
		t.Fatalf("expected '%s' but found '%s' in the dest file", content, result)
	}

	// Assert that the src file didn't change
	if result := readFile(t, src); result != content {
		t.Fatalf("expected '%s' but found '%s' in the src file", content, result)
	}
}

func TestCopyDirectory(t *testing.T) {

	fileA := "a.txt"
	fileB := "b.txt"

	contentA := "Some test content\nfor our file"
	contentB := "Some other content"

	// prepare the src diretory
	src, err := ioutil.TempDir(os.TempDir(), "test-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(src)

	writeFile(t, path.Join(src, fileA), contentA)
	writeFile(t, path.Join(src, fileB), contentB)

	// prepare the dest directory
	dest, err := ioutil.TempDir(os.TempDir(), "test-dest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dest)

	// Copy the dir
	err = CopyDirectory(src, dest)
	if err != nil {
		t.Fatalf("failed to copy the directory %s to %s: %s", src, dest, err)
	}

	// Assert that the content of the dest file is equal to the src file
	if result := readFile(t, path.Join(dest, fileA)); result != contentA {
		t.Fatalf("expected '%s' but found '%s' in the dest file", contentA, result)
	}

	if result := readFile(t, path.Join(dest, fileB)); result != contentB {
		t.Fatalf("expected '%s' but found '%s' in the dest file", contentB, result)
	}
}
