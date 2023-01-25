package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteToVolume writes a file to the volume
func WriteToVolume(file *multer.File) error {
    // Create a new file
    out, err := os.Create(filepath.Join("/mnt/data/", file.Filename))
    if err != nil {
        return fmt.Errorf("Failed to create file: %s", err)
    }
    defer out.Close()

    // Copy the uploaded file to the new file
    if _, err := io.Copy(out, file); err != nil {
        return fmt.Errorf("Failed to copy file: %s", err)
    }
    return nil
}

// RemoveTempFile removes the temporary file from the volume
func RemoveTempFile(filepath string) error {
    if err := os.Remove(filepath); err != nil {
        return fmt.Errorf("Failed to remove temporary file: %s", err)
    }
    return nil
}
