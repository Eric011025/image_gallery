package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"

	"github.com/Eric011025/image_gallery/bookmark"
	"github.com/Eric011025/image_gallery/fileinfo"
	"github.com/Eric011025/image_gallery/filemeta"
)

var (
	dataDir    string
	serverPort string
)

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

	app.Get("/bookmark", bookmark.GetBookmarkList)

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

	if fileinfo.IsUnmodifiableFile(fullPath) {
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
	if err = meta.WriteMetaFile(metaPath); err != nil {
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

	if fileinfo.IsUnmodifiableFile(fullPath) {
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
	if meta.Bookmark {
		bookmark.AppendBookmark(fullPath)
	} else {
		bookmark.RemoveBookmark(fullPath)
	}

	if err = meta.WriteMetaFile(metaPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	return c.Status(fiber.StatusOK).SendString("Bookmark Success")
}

func GetFileHandler(c *fiber.Ctx) error {
	var (
		format   string
		fullPath string
		file     fs.FileInfo
		files    []fileinfo.FileInfo
		err      error
	)
	format = c.Query("format", "render")

	if fullPath, err = url.PathUnescape(c.Params("*", "")); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid File Path")
	}

	fullPath = fileinfo.ReplaceWindowsPath(fullPath)
	fmt.Println("full path : ", fullPath)

	if strings.Split(fullPath, "/")[0] != dataDir && strings.Split(fullPath, "/")[0] != "favicon.ico" {
		fullPath = dataDir
	}

	if fileinfo.IsUnpublicFile(fullPath) {
		return c.Status(fiber.StatusNotFound).SendFile("File Not Found")
	}

	if file, err = os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).SendString("File Not Found")
		}
		return c.Status(500).SendString("Internal Server Error")
	}

	if file.IsDir() {
		if files, err = fileinfo.GetFiles(fullPath); err != nil {
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
