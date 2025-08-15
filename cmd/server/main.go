package main

import (
	"database/sql"
	"fmt"
	mysti "go-mysti"
	"go-mysti/controllers"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"

	_ "github.com/go-sql-driver/mysql"
)

func initDb() *sql.DB {
	db, err := sql.Open("mysql", "root:PassW0rd@tcp(localhost:3306)/mysti")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return db
}

func testQuery() {
	db := initDb()
	defer db.Close()

	row, err := db.Query("select container_id, container_name from container_stats group by container_id, container_name")
	if err != nil {
		panic(err)
	}
	defer row.Close()

	containerIds := make([]string, 30)
	for row.Next() {
		var containerId string
		var containerName string
		err = row.Scan(&containerId, &containerName)
		if err != nil {
			panic(err)
		}
		containerIds = append(containerIds, containerId)
		fmt.Println("container_id:", containerId)
	}

	// Prepare statement for reading data
	stmtOut, err := db.Prepare("SELECT __time, container_id FROM container_stats WHERE container_id = ?")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer stmtOut.Close()
	row, err = stmtOut.Query("79cc29057c82b4f705b7dd0022bc95dc63ea60d92e21543985d1fc4b621967d4")
	if err != nil {
		panic(err.Error())
	}

	for row.Next() {
		var timestamp string
		var containerId string
		err = row.Scan(&timestamp, &containerId)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println("container_id:", containerId)
	}
}

func main() {
	db := initDb()
	defer db.Close()
	ginEngine := gin.Default()
	ginEngine.SetFuncMap(template.FuncMap{
		"formatAsDate": func(t time.Time) string {
			return t.Format("02/01/2006")
		},
	})
	// 设置模板引擎
	tmpl := template.Must(mysti.WalkDir("assets/templates"))
	ginEngine.SetHTMLTemplate(tmpl)

	// 设置静态文件服务，主要用于返回css,js文件，html文件只能返回/static/index.html
	// 如果要请求 `/stat/good/index.html`，则被 301 到 `/static/good/`，gin 就无法找到 /good/，导致响应 401
	assetFS := http.FS(mysti.GetAssetFS("assets/static"))
	ginEngine.StaticFS("/static", assetFS)

	controllers.RegisterRoutes(db, ginEngine.Group("/"))

	ginEngine.Run(":8080")
}

func readFileFS(fsys fs.FS) func(string) (string, []byte, error) {
	return func(file string) (name string, b []byte, err error) {
		name = path.Base(file)
		b, err = fs.ReadFile(fsys, file)
		return
	}
}
