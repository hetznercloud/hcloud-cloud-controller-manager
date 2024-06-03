package testsupport

import (
	"os"
	"testing"
)

func Setfiles(t *testing.T, files map[string]string) func() {
	for file, content := range files {
		// check if file exists
		_, err := os.Stat(file)
		if err == nil {
			t.Fatalf("Trying to set file %s, but it already exists. Please choose another filepath for the test.", file)
		}

		// create file
		f, err := os.Create(file)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}

		// write content to file
		_, err = f.WriteString(content)
		if err != nil {
			t.Fatalf("Failed to write to file %s: %v", file, err)
		}

		// close file
		f.Close()
	}

	return func() {
		for file := range files {
			err := os.Remove(file)
			if err != nil {
				t.Fatalf("Failed to remove file %s: %v", file, err)
			}
		}
	}
}
