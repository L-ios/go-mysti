package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

//go:embed static/*
var staticFiles embed.FS

var (
	bindAddress = flag.String("bind", "", "Specify alternate bind address [default: all interfaces]")
	port        = flag.Int("port", 8000, "Specify alternate port [default: 8000]")
	devMode     = flag.Bool("dev", false, "Enable development mode (use local files)")
	directory   = flag.String("directory", ".", "Specify alternative directory [default: current directory]")
)

func main() {
	flag.Parse()

	var handler http.Handler

	if *devMode {
		// 开发模式：使用本地文件系统
		handler = createLocalFileHandler(*directory)
		fmt.Printf("Serving local files from directory %s\n", *directory)
	} else {
		// 生产模式：使用嵌入的文件系统
		staticRoot, _ := fs.Sub(staticFiles, "static")
		handler = createEmbedFileHandler(staticRoot)
		fmt.Println("Serving embedded files")
	}

	// 创建服务器
	addr := fmt.Sprintf("%s:%d", *bindAddress, *port)
	if *bindAddress == "" {
		addr = fmt.Sprintf(":%d", *port)
	}

	// 使用标准库 http.Server
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// 配置 HTTP/2
	http2.ConfigureServer(server, &http2.Server{})

	// 同时支持 HTTP/1.1 和 HTTP/2 (h2c)
	h2s := &http2.Server{}
	server.Handler = h2c.NewHandler(handler, h2s)

	fmt.Printf("Starting server at http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop the server")

	log.Fatal(server.ListenAndServe())
}

// 创建本地文件处理器
func createLocalFileHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构建文件路径
		path := filepath.Join(dir, r.URL.Path)

		// 清理路径，防止目录遍历攻击
		path = filepath.Clean(path)

		// 检查文件是否存在
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在，显示目录列表
				serveLocalDirectory(w, r, dir, r.URL.Path)
				return
			}
			http.Error(w, "Failed to access file", http.StatusInternalServerError)
			return
		}

		// 如果是目录，显示目录列表
		if info.IsDir() {
			// 确保目录路径以 / 结尾
			if !strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
				return
			}
			serveLocalDirectory(w, r, path, r.URL.Path)
			return
		}

		// 是文件，直接提供服务
		serveLocalFile(w, r, path)
	})
}

// 创建嵌入文件处理器
func createEmbedFileHandler(root fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// index.html
		// 移除前导斜杠获取路径
		path := strings.TrimPrefix(r.URL.Path, "/")

		path = filepath.Clean(path)

		// 如果路径为空，显示根目录内容
		if path == "" {
			serveEmbedDirectory(w, r, root, ".")
			return
		}

		// 检查文件是否存在
		file, err := root.Open(path)
		if err != nil {
			// 尝试在 static 目录下查找
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		// 获取文件信息
		info, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// 如果是目录，显示目录列表
		if info.IsDir() {
			// 确保目录路径以 / 结尾
			if !strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
				return
			}
			serveEmbedDirectory(w, r, root, path)
			return
		}

		// 是文件，提供文件内容
		serveEmbedFile(w, r, path)
	})
}

// 服务本地目录
func serveLocalDirectory(w http.ResponseWriter, r *http.Request, dirPath, urlPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 生成目录列表HTML
	html := generateDirectoryListing(urlPath, entries)
	w.Write([]byte(html))
}

// 服务嵌入目录
func serveEmbedDirectory(w http.ResponseWriter, r *http.Request, root fs.FS, dirPath string) {
	entries, err := fs.ReadDir(root, dirPath)
	if err != nil {
		// 尝试在 static 目录下查找
		entries, _ = fs.ReadDir(root, "static")

	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 生成目录列表HTML
	html := generateDirectoryListing(r.URL.Path, entries)
	w.Write([]byte(html))
}

// 服务本地文件
func serveLocalFile(w http.ResponseWriter, r *http.Request, filePath string) {
	// 设置内容类型
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// 使用标准库服务文件
	http.ServeFile(w, r, filePath)
}

// 服务嵌入文件
func serveEmbedFile(w http.ResponseWriter, r *http.Request, filePath string) {
	// 尝试直接读取文件
	data, err := staticFiles.ReadFile(filePath)
	if err != nil {
		// 尝试在 static 目录下查找
		data, err = staticFiles.ReadFile("static/" + filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	// 设置内容类型
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// 写入文件内容
	w.Write(data)
}

// 生成目录列表HTML
func generateDirectoryListing(urlPath string, entries interface{}) string {
	// 处理目录遍历
	parentDir := filepath.Dir(strings.TrimSuffix(urlPath, "/")) + "/"
	if parentDir == "." || parentDir == "./" || parentDir == "//" {
		parentDir = "/"
	}

	html := fmt.Sprintf(`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<title>Directory listing for %s</title>
</head>
<body>
<h1>Directory listing for %s</h1>
<hr>
<ul>
`, urlPath, urlPath)

	// 添加上级目录链接（如果不是根目录）
	if urlPath != "/" && urlPath != "." {
		html += fmt.Sprintf(`<li><a href="%s">../</a></li>`+"\n", parentDir)
	}

	// 统一处理不同类型的目录条目
	switch e := entries.(type) {
	case []os.DirEntry:
		for _, entry := range e {
			name := entry.Name()
			href := name

			if entry.IsDir() {
				name += "/"
				href += "/"
			}

			html += fmt.Sprintf(`<li><a href="%s">%s</a></li>`+"\n", href, name)
		}
	}

	html += `</ul>
<hr>
</body>
</html>`

	return html
}
