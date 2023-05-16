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
)

var (
	dataDir    string
	serverPort string
)

type FileInfo struct {
	Type        string
	Path        string
	PreviewPath string
	ModTime     time.Time
	Resolution  string
	Exif        string
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
	app.Get("/*", handleRequest)

	log.Fatal(app.Listen(serverPort))
}

func handleRequest(c *fiber.Ctx) error {
	var (
		fullPath string
		file     fs.FileInfo
		files    []FileInfo
		err      error
	)

	if fullPath, err = url.PathUnescape(c.Params("*", "")); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file path")
	}

	if fullPath == "" {
		fullPath = dataDir
	}
	fmt.Println("full path : ", fullPath)

	if file, err = os.Stat(fullPath); err != nil {
		return c.Status(500).SendString("Internal Server Error")
	}

	if file.IsDir() {
		if files, err = getFiles(fullPath); err != nil {
			return c.Status(500).SendString("Internal Server Error")
		}

		return c.Render("index", fiber.Map{
			"Files": files,
		})
	} else {
		return c.SendFile(fullPath)
	}
}

// getFiles는 디렉토리에서 이미지 파일 목록을 반환합니다.
func getFiles(dir string) ([]FileInfo, error) {
	var (
		fileList []fs.DirEntry
		files    []FileInfo
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

		if strings.Contains(file.Name(), ".meta") {
			continue
		}
		if strings.Contains(file.Name(), ".preview") {
			continue
		}

		filePath := filepath.Join(dir, file.Name())

		if fileStat, err = os.Stat(filePath); err != nil {
			return nil, err
		}

		fileType := "directory"
		if !file.IsDir() {
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

		files = append(files, FileInfo{
			Type:        fileType,
			Path:        filePath,
			PreviewPath: previewPath,
			ModTime:     fileStat.ModTime(),
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
