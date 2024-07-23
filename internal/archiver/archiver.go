package archiver

import (
    "archive/zip"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/bodgit/sevenzip"
)

func IsArchive(file string) bool {
    ext := strings.ToLower(filepath.Ext(file))
    supportedExts := []string{".7z", ".zip", ".rar", ".gz", ".tar", ".bz2", ".xz"}
    for _, e := range supportedExts {
        if ext == e {
            return true
        }
    }
    return false
}

func ExtractArchive(src, dest string) error {
    ext := strings.ToLower(filepath.Ext(src))
    switch ext {
    case ".zip":
        return extractZip(src, dest)
    default:
        return extractSevenZip(src, dest)
    }
}

func extractZip(src, dest string) error {
    r, err := zip.OpenReader(src)
    if err != nil {
        return fmt.Errorf("failed to open zip: %v", err)
    }
    defer r.Close()

    for _, f := range r.File {
        err := extractZipFile(f, dest)
        if err != nil {
            return fmt.Errorf("failed to extract file %s: %v", f.Name, err)
        }
    }

    return nil
}

func extractZipFile(f *zip.File, dest string) error {
    rc, err := f.Open()
    if err != nil {
        return err
    }
    defer rc.Close()

    path := filepath.Join(dest, f.Name)

    if f.FileInfo().IsDir() {
        os.MkdirAll(path, f.Mode())
    } else {
        err := os.MkdirAll(filepath.Dir(path), 0755)
        if err != nil {
            return err
        }
        f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil {
            return err
        }
        defer f.Close()

        _, err = io.Copy(f, rc)
        if err != nil {
            return err
        }
    }
    return nil
}

func extractSevenZip(src, dest string) error {
    r, err := os.Open(src)
    if err != nil {
        return fmt.Errorf("failed to open archive: %v", err)
    }
    defer r.Close()

    archive, err := sevenzip.NewReader(r, int64(^uint(0)>>1))
    if err != nil {
        return fmt.Errorf("failed to create archive reader: %v", err)
    }

    for _, f := range archive.File {
        err := extractSevenZipFile(f, dest)
        if err != nil {
            return fmt.Errorf("failed to extract file %s: %v", f.Name, err)
        }
    }

    return nil
}

func extractSevenZipFile(f *sevenzip.File, dest string) error {
    path := filepath.Join(dest, f.Name)
    if f.FileInfo().IsDir() {
        return os.MkdirAll(path, f.Mode())
    }

    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }

    outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
    if err != nil {
        return err
    }
    defer outFile.Close()

    rc, err := f.Open()
    if err != nil {
        return err
    }
    defer rc.Close()

    _, err = io.Copy(outFile, rc)
    return err
}

func DetermineArchiveType(file string) (string, error) {
    ext := strings.ToLower(filepath.Ext(file))
    switch ext {
    case ".7z", ".zip", ".rar", ".gz", ".tar", ".bz2", ".xz":
        return ext[1:], nil
    default:
        return "", fmt.Errorf("unsupported archive type: %s", ext)
    }
}