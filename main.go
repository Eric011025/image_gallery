package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

type FileInfo struct {
	Type    string
	Path    string
	ModTime time.Time
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
	app.Get("/file/*", handleFileRequest)
	app.Get("/*", handleRequest)

	log.Fatal(app.Listen(":3000"))
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

func handleFileRequest(c *fiber.Ctx) error {
	fmt.Println("file request")
	path, err := url.PathUnescape(c.Params("*", ""))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file path")
	}

	fullPath := path
	fmt.Println("full path : ", fullPath)
	return c.SendFile(fullPath)
}

// getFiles는 디렉토리에서 이미지 파일 목록을 반환합니다.
func getFiles(dir string) ([]FileInfo, error) {
	localFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := []FileInfo{}
	for _, file := range localFiles {
		fmt.Println("file list : ", file.Name())
		fileType := "directory"
		if !file.IsDir() {
			fileType = "file"
		}
		files = append(files, FileInfo{
			Type:    fileType,
			Path:    filepath.Join(dir, file.Name()),
			ModTime: file.ModTime(),
		})
	}

	return files, nil
}
