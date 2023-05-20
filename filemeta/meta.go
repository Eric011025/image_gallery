package filemeta

import (
	"encoding/json"
	"os"
	"time"
)

type FileMeta struct {
	Hide         bool   `json:"hide"`
	HideTime     string `json:"hide_time"`
	Bookmark     bool   `json:"bookmark"`
	BookmarkTime string `json:"bookmark_time"`
	Resolution   string `json:"resolution"`
	Exif         string `json:"exif"`
}

func ReadMeta(path string) (FileMeta, error) {
	var (
		meta     FileMeta
		fileByte []byte
		err      error
	)

	if fileByte, err = os.ReadFile(path); err != nil {
		if os.IsNotExist(err) {
			// create meta file
			if err = meta.WriteMetaFile(path); err != nil {
				return meta, err
			}
		}
		return meta, err
	} else {
		if err = json.Unmarshal(fileByte, &meta); err != nil {
			return FileMeta{}, err
		}
	}

	return meta, nil
}

func (meta *FileMeta) WriteMetaFile(path string) error {
	var (
		fileMetaByte []byte
		err          error
	)

	if fileMetaByte, err = json.Marshal(meta); err != nil {
		return err
	}
	if err = os.WriteFile(path, fileMetaByte, 0644); err != nil {
		return err
	}
	return nil
}

func (meta *FileMeta) FileHide() {
	meta.Hide = true
	meta.HideTime = time.Now().String()
}

func (meta *FileMeta) FileBookmark() {
	meta.Bookmark = !meta.Bookmark
	meta.BookmarkTime = time.Now().String()
}
