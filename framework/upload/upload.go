package upload

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/logger"
)

// Upload handles file uploads
type Upload struct {
    config     *config.UploadConfig
    logger     logger.Logger
    uploadPath string
}

// File represents an uploaded file
type File struct {
    Name      string
    Original  string
    Size      int64
    MimeType  string
    Extension string
    Path      string
    URL       string
}

// UploadResult contains the result of a file upload
type UploadResult struct {
    Success bool
    Files   []*File
    Errors  []error
}

// New creates a new Upload instance
func New(cfg *config.UploadConfig, log logger.Logger, uploadPath string) *Upload {
    // Create upload directory if it doesn't exist
    os.MkdirAll(uploadPath, 0755)

    return &Upload{
        config:     cfg,
        logger:     log,
        uploadPath: uploadPath,
    }
}

// ProcessFile processes a single file upload
func (u *Upload) ProcessFile(file multipart.File, handler *multipart.FileHeader) (*File, error) {
    // Validate file size
    if handler.Size > u.config.MaxSize {
        return nil, fmt.Errorf("file exceeds maximum size of %d bytes", u.config.MaxSize)
    }

    // Read file content to detect MIME type
    content, err := io.ReadAll(file)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    // Detect MIME type
    mimeType := http.DetectContentType(content)
    if !u.isAllowedType(mimeType) {
        return nil, fmt.Errorf("file type %s is not allowed", mimeType)
    }

    // Reset file reader
    file.Seek(0, 0)

    // Generate unique filename
    ext := filepath.Ext(handler.Filename)
    originalName := strings.TrimSuffix(handler.Filename, ext)
    uniqueName := u.generateUniqueFilename(originalName, ext)

    // Create subdirectory by date
    datePath := time.Now().Format("2006/01/02")
    fullPath := filepath.Join(u.uploadPath, datePath)
    os.MkdirAll(fullPath, 0755)

    // Save file
    filePath := filepath.Join(fullPath, uniqueName)
    dest, err := os.Create(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to create file: %w", err)
    }
    defer dest.Close()

    _, err = io.Copy(dest, file)
    if err != nil {
        return nil, fmt.Errorf("failed to save file: %w", err)
    }

    // Build URL
    url := fmt.Sprintf("/uploads/%s/%s", datePath, uniqueName)

    return &File{
        Name:      uniqueName,
        Original:  handler.Filename,
        Size:      handler.Size,
        MimeType:  mimeType,
        Extension: ext,
        Path:      filePath,
        URL:       url,
    }, nil
}

// ProcessFiles processes multiple file uploads
func (u *Upload) ProcessFiles(files []*multipart.FileHeader) *UploadResult {
    result := &UploadResult{
        Success: true,
        Files:   make([]*File, 0),
        Errors:  make([]error, 0),
    }

    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            result.Errors = append(result.Errors, err)
            result.Success = false
            continue
        }
        defer file.Close()

        uploadedFile, err := u.ProcessFile(file, fileHeader)
        if err != nil {
            result.Errors = append(result.Errors, err)
            result.Success = false
            continue
        }

        result.Files = append(result.Files, uploadedFile)
    }

    return result
}

// DeleteFile deletes a file from the uploads directory
func (u *Upload) DeleteFile(filePath string) error {
    fullPath := filepath.Join(u.uploadPath, filePath)
    return os.Remove(fullPath)
}

// GetFile retrieves a file from the uploads directory
func (u *Upload) GetFile(filePath string) ([]byte, error) {
    fullPath := filepath.Join(u.uploadPath, filePath)
    return os.ReadFile(fullPath)
}

// isAllowedType checks if the MIME type is allowed
func (u *Upload) isAllowedType(mimeType string) bool {
    for _, allowed := range u.config.AllowedTypes {
        if mimeType == allowed {
            return true
        }
        // Check for wildcard
        if strings.HasSuffix(allowed, "/*") {
            prefix := strings.TrimSuffix(allowed, "/*")
            if strings.HasPrefix(mimeType, prefix) {
                return true
            }
        }
    }
    return false
}

// generateUniqueFilename creates a unique filename
func (u *Upload) generateUniqueFilename(original, ext string) string {
    // Generate random string
    bytes := make([]byte, 16)
    rand.Read(bytes)
    random := hex.EncodeToString(bytes)

    // Clean filename
    original = strings.ReplaceAll(original, " ", "_")
    original = strings.ToLower(original)

    return fmt.Sprintf("%s_%s_%d%s", original, random, time.Now().UnixNano(), ext)
}

// GetFileInfo returns file information
func (u *Upload) GetFileInfo(filePath string) (os.FileInfo, error) {
    fullPath := filepath.Join(u.uploadPath, filePath)
    return os.Stat(fullPath)
}

// ListFiles lists all files in a directory
func (u *Upload) ListFiles(path string) ([]string, error) {
    fullPath := filepath.Join(u.uploadPath, path)
    files, err := os.ReadDir(fullPath)
    if err != nil {
        return nil, err
    }

    names := make([]string, len(files))
    for i, file := range files {
        names[i] = file.Name()
    }
    return names, nil
}
