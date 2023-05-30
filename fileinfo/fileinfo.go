package fileinfo

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Eric011025/image_gallery/filemeta"
)

type FileInfo struct {
	Type        string            `json:"type"`
	Path        string            `json:"path"`
	PreviewPath string            `json:"preview_path"`
	ModTime     time.Time         `json:"mod_time"`
	FileMeta    filemeta.FileMeta `json:"file_meta"`
}

func GetFile(path string) (FileInfo, error) {
	var (
		file     FileInfo
		fileStat fs.FileInfo
		meta     filemeta.FileMeta
		err      error
	)

	if fileStat, err = os.Stat(path); err != nil {
		return file, err
	}
	file.Path = path

	if fileStat.IsDir() {
		file.Type = "directory"
	} else {
		if meta, err = filemeta.ReadMeta(path); err != nil {
			return file, err
		}
		file.Type = "file"
		file.PreviewPath = path + ".preview"

		if _, err = os.Stat(file.PreviewPath); err != nil {
			if os.IsNotExist(err) {
				if err = createPreviewFile(file.Path, file.PreviewPath); err != nil {
					file.PreviewPath = file.Path
				}
			}
		}
	}

	file.FileMeta = meta
	file.ModTime = fileStat.ModTime()
	return file, err
}

// GetFiles는 디렉토리에서 이미지 파일 목록을 반환합니다.
func GetFiles(dir string) ([]FileInfo, error) {
	var (
		fileList []fs.DirEntry
		files    []FileInfo
		meta     filemeta.FileMeta
		err      error
	)

	if fileList, err = os.ReadDir(dir); err != nil {
		return nil, err
	}

	for _, file := range fileList {
		var (
			previewPath string
			fileStat    fs.FileInfo
		)

		if IsUnmodifiableFile(file.Name()) {
			continue
		}

		filePath := filepath.Join(dir, file.Name())

		if fileStat, err = os.Stat(filePath); err != nil {
			return nil, err
		}

		fileType := "directory"
		if !file.IsDir() {
			// 삭제한 파일인지 확인
			if meta, err = filemeta.ReadMeta(filePath); err != nil {
				return nil, err
			}
			if meta.Hide {
				continue
			}

			fileType = "file"
			previewPath = filepath.Join(dir, file.Name()+".preview")
			if _, err := os.Stat(previewPath); err != nil {
				if os.IsNotExist(err) {
					// create preivew file
					if err = createPreviewFile(filePath, previewPath); err != nil {
						fmt.Println("create preview image failed : ", err.Error())
						previewPath = filePath
					}
				}
			}
		}

		filePath = ReplaceWindowsPath(filePath)
		previewPath = ReplaceWindowsPath(previewPath)

		files = append(files, FileInfo{
			Type:        fileType,
			Path:        filePath,
			PreviewPath: previewPath,
			ModTime:     fileStat.ModTime(),
			FileMeta:    meta,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Unix() > files[j].ModTime.Unix()
	})

	return files, nil
}

func ReplaceWindowsPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func IsUnpublicFile(path string) bool {
	return strings.Contains(path, ".meta")
}

func IsUnmodifiableFile(path string) bool {
	if strings.Contains(path, ".meta") {
		return true
	} else if strings.Contains(path, ".preview") {
		return true
	}
	return false
}

func createPreviewFile(sourcePath string, previewPath string) error {
	fmt.Println("preview image create : ", sourcePath, previewPath)
	if err := exec.Command("ffmpeg", "-i", sourcePath, "-vf", "scale=-2:360", "-f", "image2", "-y", previewPath).Run(); err != nil {
		return err
	}
	return nil
}
