package bookmark

import (
	"encoding/json"
	"os"

	"github.com/Eric011025/image_gallery/fileinfo"
	"github.com/gofiber/fiber/v2"
)

const (
	bookmarkListFile = ".bookmark"
)

type BookmarkList struct {
	Files []string
}

func GetBookmarkList(c *fiber.Ctx) error {
	var (
		files        []fileinfo.FileInfo
		file         fileinfo.FileInfo
		bookmakrList BookmarkList
		err          error
	)

	if bookmakrList, err = ReadBookmarkList(); err != nil {
		return c.Status(500).SendString("Internal Server Error")
	}

	for _, bookmarkFile := range bookmakrList.Files {
		if file, err = fileinfo.GetFile(bookmarkFile); err != nil {
			continue
		}
		if file.FileMeta.Hide {
			continue
		}
		if file.FileMeta.Bookmark {
			files = append(files, file)
		}
	}

	return c.Render("index", fiber.Map{
		"Files": files,
	})
}

func ReadBookmarkList() (BookmarkList, error) {
	var (
		bookmarkList BookmarkList
		fileByte     []byte
		err          error
	)

	if fileByte, err = os.ReadFile(bookmarkListFile); err != nil {
		if os.IsNotExist(err) {
			err = bookmarkList.WriteBookmarkList()
			return bookmarkList, err
		}
		return bookmarkList, err
	}

	if err = json.Unmarshal(fileByte, &bookmarkList); err != nil {
		return bookmarkList, err
	}

	return bookmarkList, err
}

func AppendBookmark(path string) error {
	var (
		bookmarkList BookmarkList
		err          error
	)

	if bookmarkList, err = ReadBookmarkList(); err != nil {
		return err
	}

	for _, file := range bookmarkList.Files {
		if file == "path" {
			return nil
		}
	}

	bookmarkList.Files = append(bookmarkList.Files, path)
	if err = bookmarkList.WriteBookmarkList(); err != nil {
		return err
	}

	return err
}

func RemoveBookmark(path string) error {
	var (
		bookmarkList BookmarkList
		files        []string
		err          error
	)

	if bookmarkList, err = ReadBookmarkList(); err != nil {
		return err
	}

	for _, file := range bookmarkList.Files {
		if file != path {
			files = append(files, file)
		}
	}
	bookmarkList.Files = files

	if err = bookmarkList.WriteBookmarkList(); err != nil {
		return err
	}

	return err
}

func (list *BookmarkList) WriteBookmarkList() error {
	var (
		fileByte []byte
		err      error
	)

	if fileByte, err = json.Marshal(list); err != nil {
		return err
	}

	if err = os.WriteFile(bookmarkListFile, fileByte, 0644); err != nil {
		return err
	}

	return nil
}
