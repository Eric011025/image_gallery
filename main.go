package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"

	"github.com/Eric011025/image_gallery/filemeta"
)

var (
	dataDir    string
	serverPort string
)

type FileInfo struct {
	Type        string            `json:"type"`
	Path        string            `json:"path"`
	PreviewPath string            `json:"preview_path"`
	ModTime     time.Time         `json:"mod_time"`
	FileMeta    filemeta.FileMeta `json:"file_meta"`
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	dataDir = os.Getenv("DATA_DIR")
	serverPort = os.Getenv("SERVER_PORT")
}

func main() {
	// HTML 템플릿 엔진 초기화
	engine := html.New("./templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// 요청 URL의 경로에 따라 파일 데이터 또는 파일 목록을 반환하는 엔드포인트
	app.Get("/*", GetFileHandler)
	app.Delete("/*", DeleteFileHandler)
	app.Patch("/*", PatchBookmarkHandler)

	log.Fatal(app.Listen(serverPort))
}

func DeleteFileHandler(c *fiber.Ctx) error {
	var (
		fullPath string
		metaPath string
		fileStat fs.FileInfo
		meta     filemeta.FileMeta
		err      error
	)

	if fullPath, err = url.PathUnescape(c.Params("*", "")); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	if isUnmodifiableFile(fullPath) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid File Path")
	}

	if fileStat, err = os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).SendString("File Not Found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	if fileStat.IsDir() {
		// TODO : 나중에 디렉토리 삭제 기능 추가 필요
		return c.Status(fiber.StatusBadRequest).SendString("Invalid File Path")
	}

	// file meta file
	metaPath = fullPath + ".meta"
	if meta, err = filemeta.ReadMeta(metaPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}
	meta.FileHide()
	if _, err = meta.WriteMetaFile(metaPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	return c.Status(fiber.StatusOK).SendString("Delete File Success")
}

func PatchBookmarkHandler(c *fiber.Ctx) error {
	var (
		fullPath string
		metaPath string
		fileStat fs.FileInfo
		meta     filemeta.FileMeta
		err      error
	)

	if fullPath, err = url.PathUnescape(c.Params("*", "")); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	if isUnmodifiableFile(fullPath) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid File Path")
	}

	if fileStat, err = os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).SendString("File Not Found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	if fileStat.IsDir() {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid File Path")
	}

	// file meta file
	metaPath = fullPath + ".meta"
	if meta, err = filemeta.ReadMeta(metaPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}
	meta.FileBookmark()
	if _, err = meta.WriteMetaFile(metaPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	return c.Status(fiber.StatusOK).SendString("Bookmark Success")
}

func GetFileHandler(c *fiber.Ctx) error {
	var (
		format   string
		fullPath string
		file     fs.FileInfo
		files    []FileInfo
		err      error
	)
	format = c.Query("format", "render")

	if fullPath, err = url.PathUnescape(c.Params("*", "")); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid File Path")
	}

	fullPath = strings.Replace(fullPath, "\\", "/", -1)
	fmt.Println("full path : ", fullPath)

	if strings.Split(fullPath, "/")[0] != dataDir && strings.Split(fullPath, "/")[0] != "favicon.ico" {
		fullPath = dataDir
	}

	if isUnpublicFile(fullPath) {
		return c.Status(fiber.StatusNotFound).SendFile("File Not Found")
	}

	if file, err = os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).SendString("File Not Found")
		}
		return c.Status(500).SendString("Internal Server Error")
	}

	if file.IsDir() {
		if files, err = getFiles(fullPath); err != nil {
			return c.Status(500).SendString("Internal Server Error")
		}

		if format == "json" {
			return c.JSON(files)
		}

		return c.Render("index", fiber.Map{
			"Files": files,
		})
	} else {
		return c.SendFile(fullPath)
	}
}

func isUnpublicFile(path string) bool {
	return strings.Contains(path, ".meta")
}

func isUnmodifiableFile(path string) bool {
	if strings.Contains(path, ".meta") {
		return true
	} else if strings.Contains(path, ".preview") {
		return true
	}
	return false
}

// getFiles는 디렉토리에서 이미지 파일 목록을 반환합니다.
func getFiles(dir string) ([]FileInfo, error) {
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

		if isUnmodifiableFile(file.Name()) {
			continue
		}

		filePath := filepath.Join(dir, file.Name())

		if fileStat, err = os.Stat(filePath); err != nil {
			return nil, err
		}

		fileType := "directory"
		if !file.IsDir() {
			// 삭제한 파일인지 확인
			if meta, err = filemeta.ReadMeta(filePath + ".meta"); err != nil {
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

		filePath = strings.Replace(filePath, "\\", "/", -1)
		previewPath = strings.Replace(previewPath, "\\", "/", -1)

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

func createPreviewFile(sourcePath string, previewPath string) error {
	fmt.Println("preview image create : ", sourcePath, previewPath)
	if err := exec.Command("ffmpeg", "-i", sourcePath, "-vf", "scale=-2:360", "-f", "image2", "-y", previewPath).Run(); err != nil {
		return err
	}
	return nil
}
