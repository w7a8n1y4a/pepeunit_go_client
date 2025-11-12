package pepeunit

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type FileManager struct {
	pathLocks sync.Map
}

// NewFileManager creates a new file manager
func NewFileManager() *FileManager {
	return &FileManager{}
}

// FileExists checks if a file exists
func (fm *FileManager) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// ReadJSON reads and parses a JSON file
func (fm *FileManager) ReadJSON(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err == nil {
		return result, nil
	}

	var jsonString string
	if err := json.Unmarshal(data, &jsonString); err == nil {
		if err := json.Unmarshal([]byte(jsonString), &result); err == nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("invalid JSON format in %s", filePath)
}

// WriteJSON writes data to a JSON file
func (fm *FileManager) WriteJSON(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	perm := os.FileMode(0644)
	if info, statErr := os.Stat(filePath); statErr == nil {
		perm = info.Mode().Perm()
	}

	return writeFileAtomic(filePath, jsonData, perm)
}

// CopyFile copies a file from source to destination
func (fm *FileManager) CopyFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// CopyDirectoryContents copies all contents from source directory to destination directory
func (fm *FileManager) CopyDirectoryContents(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return fm.CopyFile(path, destPath)
	})
}

// ExtractTarGz extracts a tar.gz archive to a destination directory
func (fm *FileManager) ExtractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bufReader := bufio.NewReader(file)

	// Detect header: gzip (1f 8b) vs zlib (RFC1950)
	header, err := bufReader.Peek(2)
	if err != nil {
		return err
	}

	var decompressed io.ReadCloser
	switch {
	case len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b:
		gz, err := gzip.NewReader(bufReader)
		if err != nil {
			return err
		}
		decompressed = gz
	default:
		zr, err := zlib.NewReader(bufReader)
		if err != nil {
			return err
		}
		decompressed = zr
	}
	defer decompressed.Close()

	tarReader := tar.NewReader(decompressed)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip if the path contains ".." for security
		if strings.Contains(header.Name, "..") {
			continue
		}

		destPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(destPath, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(destPath), 0755)
			if err != nil {
				return err
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type: %v", header.Typeflag)
		}
	}

	return nil
}

// AppendToJSONList appends an item to a JSON array file
func (fm *FileManager) AppendToJSONList(filePath string, item interface{}) error {
	var arrayData []interface{}
	var objectData map[string]interface{}
	writeAsObject := false

	if fm.FileExists(filePath) {
		fileData, err := os.ReadFile(filePath)
		if err == nil {
			if err := json.Unmarshal(fileData, &arrayData); err != nil {
				if err := json.Unmarshal(fileData, &objectData); err == nil {
					if entries, ok := objectData["entries"].([]interface{}); ok {
						arrayData = entries
						writeAsObject = true
					}
				}
			}
		}
	}

	if arrayData == nil {
		arrayData = make([]interface{}, 0)
	}

	arrayData = append(arrayData, item)

	if writeAsObject {
		objectData["entries"] = arrayData
		return fm.WriteJSON(filePath, objectData)
	}

	return fm.WriteJSON(filePath, arrayData)
}

func (fm *FileManager) AppendNDJSONWithLimit(filePath string, item map[string]interface{}, maxLines int) error {
	muIface, _ := fm.pathLocks.LoadOrStore(filePath, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	if fm.FileExists(filePath) {
		f, err := os.Open(filePath)
		if err == nil {
			reader := bufio.NewReader(f)
			first, _ := reader.Peek(1)
			_ = f.Close()
			if len(first) == 1 && first[0] == '[' {
				data, err := os.ReadFile(filePath)
				if err == nil {
					var arr []interface{}
					if json.Unmarshal(data, &arr) == nil {
						tmp := filePath + ".tmp"
						tf, terr := os.Create(tmp)
						if terr == nil {
							enc := json.NewEncoder(tf)
							for _, it := range arr {
								_ = enc.Encode(it)
							}
							_ = tf.Close()
							_ = os.Rename(tmp, filePath)
						}
					}
				}
			}
		}
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	_ = f.Sync()
	if maxLines > 0 {
		return fm.TrimNDJSON(filePath, maxLines)
	}
	return nil
}

func (fm *FileManager) IterNDJSON(filePath string) ([]map[string]interface{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return []map[string]interface{}{}, nil
	}
	defer f.Close()
	result := make([]map[string]interface{}, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if json.Unmarshal([]byte(line), &obj) == nil {
			result = append(result, obj)
		}
	}
	return result, nil
}

func (fm *FileManager) TrimNDJSON(filePath string, maxLines int) error {
	if maxLines <= 0 {
		return nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()
	total := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		total++
	}
	if total <= maxLines {
		return nil
	}
	_, _ = f.Seek(0, 0)
	toSkip := total - maxLines
	tmpPath := filePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return nil
	}
	defer out.Close()
	scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		if toSkip > 0 {
			toSkip--
			continue
		}
		_, _ = out.WriteString(scanner.Text())
		_, _ = out.WriteString("\n")
	}
	_ = out.Sync()
	_ = os.Rename(tmpPath, filePath)
	return nil
}

// CreateTarGz creates a tar.gz archive from a directory
func (fm *FileManager) CreateTarGz(sourceDir, archivePath string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func writeFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)

	f, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := f.Name()

	var writeErr error
	_, writeErr = f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if closeErr := f.Close(); writeErr == nil && closeErr != nil {
		writeErr = closeErr
	}
	if writeErr == nil {
		if chmodErr := os.Chmod(tmpPath, perm); chmodErr != nil {
			writeErr = chmodErr
		}
	}
	if writeErr == nil {
		writeErr = os.Rename(tmpPath, filename)
	}
	if writeErr == nil {
		if df, derr := os.Open(dir); derr == nil {
			_ = df.Sync()
			_ = df.Close()
		}
	}
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}
	return nil
}
