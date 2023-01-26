package file

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/config"
)

// WriteToVolume writes a file to the volume
func WriteToVolume(file *multipart.FileHeader) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %s", err)
    }
    // Open the uploaded file
    src, err := file.Open()
    if err != nil {
        return fmt.Errorf("failed to open file: %s", err)
    }
    defer src.Close()

    // Create a new file
    dst, err := os.Create(filepath.Join(cfg.VolumePath, file.Filename))
    if err != nil {
        return fmt.Errorf("failed to create file: %s", err)
    }
    defer dst.Close()

    // Copy the uploaded file to the new file
    if _, err := io.Copy(dst, src); err != nil {
        return fmt.Errorf("failed to copy file: %s", err)
    }
    return nil
}

// RemoveTempFile removes the temporary file from the volume
func RemoveTempFile(filepath string) error {
    if err := os.Remove(filepath); err != nil {
        return fmt.Errorf("failed to remove temporary file: %s", err)
    }
    return nil
}
