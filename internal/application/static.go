package application

import (
	"io"
	"os"
	"path/filepath"
)

func CopyDir(sourceDir, destinationDir string) error {
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create the destination path by replacing the source directory with the destination directory
		destinationPath := filepath.Join(destinationDir, path[len(sourceDir):])

		if info.IsDir() {
			// Create the directory in the destination path
			err = os.MkdirAll(destinationPath, info.Mode())
			if err != nil {
				return err
			}
		} else {
			// Copy the file
			err = CopyFile(path, destinationPath)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func CopyFile(sourcePath, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	err = destinationFile.Sync()
	if err != nil {
		return err
	}

	sourceFileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	err = os.Chmod(destinationPath, sourceFileInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}
