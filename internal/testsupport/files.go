package testsupport

import (
	"os"
	"testing"
)

// SetFiles can be used to temporarily create files on the local file system.
// It returns a function that will clean up all files it created.
func SetFiles(t *testing.T, files map[string]string) func() {
	for file, content := range files {
		filepath := os.TempDir() + "/" + file

		// check if file exists
		_, err := os.Stat(filepath)
		if err == nil {
			t.Fatalf("Trying to set file %s, but it already exists. Please choose another filepath for the test.", filepath)
		}

		// create file
		f, err := os.Create(filepath)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filepath, err)
		}

		// write content to file
		_, err = f.WriteString(content)
		if err != nil {
			t.Fatalf("Failed to write to file %s: %v", filepath, err)
		}

		// close file
		f.Close()
	}

	return func() {
		for file := range files {
			filepath := os.TempDir() + "/" + file
			err := os.Remove(filepath)
			if err != nil {
				t.Fatalf("Failed to remove file %s: %v", filepath, err)
			}
		}
	}
}
