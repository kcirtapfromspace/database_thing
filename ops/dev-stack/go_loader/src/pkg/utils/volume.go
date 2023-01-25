package volume

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

func WriteToVolume(file *multipart.FileHeader) error {
    // Open the file
    src, err := file.Open()
    if err != nil {
        return fmt.Errorf("Failed to open file: %v", err)
    }
    defer src.Close()

    // Create a new file on the persistent volume
    dst, err := os.Create("/mnt/data/" + file.Filename)
    if err != nil {
        return fmt.Errorf("Failed to create file on volume: %v", err)
    }
    defer dst.Close()

    // Copy the contents of the uploaded file to the new file on the volume
    _, err = io.Copy(dst, src)
    if err != nil {
        return fmt.Errorf("Failed to write file to volume: %v", err)
    }

    return nil
}

