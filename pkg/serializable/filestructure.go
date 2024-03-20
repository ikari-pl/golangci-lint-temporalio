package serializable

import (
	"fmt"
	"os"
	"path/filepath"
)

// findMainDirectory finds the root directory of the project by looking for the go.mod file.
// It starts from the current working directory and traverses upwards until it finds the root directory.
func findMainDirectory() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if isRootDirectory(currentDir) {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		// Check if the parent directory is the same as the current directory, indicating we've reached the root
		if parentDir == currentDir {
			return "", fmt.Errorf("could not find the root directory")
		}

		currentDir = parentDir
	}
}

// isRootDirectory checks if the given directory is the root directory of the project.
// It does so by checking if the directory contains a go.mod file.
func isRootDirectory(dir string) bool {
	// Check if the directory contains a go.mod file
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	if err == nil {
		return true // Directory contains go.mod file, considered as project root
	} else if os.IsNotExist(err) {
		return false // go.mod file does not exist in the directory
	} else {
		// Handle error if unable to determine the existence of go.mod file
		return false
	}
}
