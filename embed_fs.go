package go_mysti

import (
	"embed"
	"html/template"
	"io/fs"
)

// 嵌入所有静态资源
//
//go:embed assets
var embeddedFiles embed.FS

// 创建子文件系统
func GetAssetFS(subPath string) fs.FS {
	if len(subPath) < 1 {
		return embeddedFiles
	}
	subFS, err := fs.Sub(embeddedFiles, subPath)
	if err != nil {
		panic(err)
	}
	return subFS
}

func WalkDir(subPath string) (*template.Template, error) {
	tmpl := template.New("")
	fsys := GetAssetFS(subPath)

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			return nil
		}

		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		t := tmpl.New(path)
		_, err = t.Parse(string(b))
		if err != nil {
			return err
		}
		return nil
	})

	return tmpl, err
}
