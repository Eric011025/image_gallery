package main

import (
	"fmt"
	"io/ioutil"
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

type FileInfo struct {
	Type        string
	Path        string
	PreviewPath string
	ModTime     time.Time
	Resolution  string
	Exif        string
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	// HTML 템플릿 엔진 초기화
	engine := html.New("./templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// 요청 URL의 경로에 따라 파일 데이터 또는 파일 목록을 반환하는 엔드포인트
	app.Get("/*", handleRequest)

	log.Fatal(app.Listen(os.Getenv("SERVER_PORT")))
}

func handleRequest(c *fiber.Ctx) error {
	fullPath, err := url.PathUnescape(c.Params("*", ""))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file path")
	}

	if fullPath == "" {
		fullPath = os.Getenv("DATA_DIRECTORY")
	}
	fmt.Println("full path : ", fullPath)
	file, err := os.Stat(fullPath)
	if err != nil {
		return c.Status(500).SendString("Internal Server Error")
	}

	if file.IsDir() {
		files, err := getFiles(fullPath)
		if err != nil {
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
	localFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := []FileInfo{}
	for _, file := range localFiles {
		if strings.Contains(file.Name(), ".meta") {
			continue
		}
		if strings.Contains(file.Name(), ".preview") {
			continue
		}
		fmt.Println("file list : ", file.Name())

		filePath := filepath.Join(dir, file.Name())
		var previewPath string

		fileType := "directory"
		if file.IsDir() == false {
			fileType = "file"
			previewPath = filepath.Join(dir, file.Name()+".preview")
			_, err := os.Stat(previewPath)
			if err != nil {
				if os.IsNotExist(err) {
					// create preivew file
					err = createPreviewFile(filePath, previewPath)
					if err != nil {
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
			ModTime:     file.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Unix() > files[j].ModTime.Unix()
	})

	return files, nil
}

func createPreviewFile(sourcePath string, previewPath string) error {
	fmt.Println("preview image create : ", sourcePath, previewPath)
	err := exec.Command("ffmpeg", "-i", sourcePath, "-vf", "scale=-2:360", "-f", "image2", "-y", previewPath).Run()
	if err != nil {
		return err
	}
	return nil
}
