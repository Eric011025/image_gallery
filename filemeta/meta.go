package filemeta

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

type FileMeta struct {
	Hide         bool   `json:"hide"`
	HideTime     string `json:"hide_time"`
	Bookmark     bool   `json:"bookmark"`
	BookmarkTime string `json:"bookmark_time"`
	Resolution   string `json:"resolution"`
	Exif         Exif   `json:"exif"`
}

type Exif struct {
	SourceFile          string `json:"source_file"`
	FileName            string `json:"file_name"`
	FileSize            string `json:"file_size"`
	FileModifyDate      string `json:"file_modify_date"`
	FileAccessDate      string `json:"file_access_date"`
	FileInodeChangeDate string `json:"file_inode_change_date"`
	FileType            string `json:"file_type"`
	FileTypeExtension   string `json:"file_type_extension"`
	ImageWidth          int    `json:"image_width"`
	ImageHeight         int    `json:"image_height"`
	BitDepth            int    `json:"bit_depth"`
	ColorType           string `json:"color_type"`
	Parameters          string `json:"parameters"`
}

func ReadMeta(path string) (FileMeta, error) {
	var (
		meta     FileMeta
		metaPath = path + ".meta"
		fileByte []byte
		err      error
	)

	if fileByte, err = os.ReadFile(metaPath); err != nil {
		if os.IsNotExist(err) {
			if err = meta.ReadExif(path); err != nil {
				fmt.Println("err : ", err.Error())
			}

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
	if err = os.WriteFile(path+".meta", fileMetaByte, 0644); err != nil {
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

func (meta *FileMeta) ReadExif(path string) error {
	var (
		cmd                 *exec.Cmd
		output              []byte
		outputMap           = make([]map[string]interface{}, 0)
		outputSnakeCase     = make(map[string]interface{}, 0)
		outputSnakeCaseByte []byte
		exif                Exif
		err                 error
	)
	cmd = exec.Command("exiftool", path, "-j")
	if output, err = cmd.Output(); err != nil {
		return err
	}

	if err = json.Unmarshal(output, &outputMap); err != nil {
		return err
	}

	for k, v := range outputMap[0] {
		outputSnakeCase[toSnakeCase(k)] = v
	}

	if outputSnakeCaseByte, err = json.Marshal(outputSnakeCase); err != nil {
		return err
	}

	if err = json.Unmarshal(outputSnakeCaseByte, &exif); err != nil {
		return err
	}

	meta.Exif = exif
	return nil
}

func toSnakeCase(str string) string {
	runes := []rune(str)
	var result []string

	for i := 0; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) {
			if i != 0 {
				result = append(result, "_")
			}
			result = append(result, string(unicode.ToLower(runes[i])))
		} else {
			result = append(result, string(runes[i]))
		}
	}

	return strings.Join(result, "")
}
