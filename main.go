package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter().StrictSlash(true)
var db *sql.DB

func initDB() {
	var err error
	config := mysql.Config{
		User:                 "root",
		Passwd:               "root",
		Addr:                 "127.0.0.1:3306",
		Net:                  "tcp",
		DBName:               "goblog",
		AllowNativePasswords: true,
	}

	db, err = sql.Open("mysql", config.FormatDSN())
	checkError(err)

	// 设置最大连接数
	db.SetMaxOpenConns(25)
	// 设置最大空闲连接数
	db.SetMaxIdleConns(25)
	// 设置每个链接的过期时间
	db.SetConnMaxLifetime(5 * time.Minute)

	// 尝试链接失败报错
	err = db.Ping()
	checkError(err)
}

func createTables() {
	createArticlesSQL := `CREATE TABLE IF NOT EXISTS articles(
    id bigint(20) PRIMARY KEY AUTO_INCREMENT NOT NULL,
    title varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    body longtext COLLATE utf8mb4_unicode_ci
	); `

	_, err := db.Exec(createArticlesSQL)
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>欢迎来到 goblog！</h1>")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "此博客是用以记录编程笔记，如您有反馈或建议，请联系 "+
		"<a href=\"mailto:summer@example.com\">summer@example.com</a>")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "<h1>请求页面未找到 :(</h1><p>如有疑惑，请联系我们。</p>")
}

// Article  对应一条文章数据
type Article struct {
	ID          int64
	Title, Body string
}

// Link 方法生成文章链接
func (a Article) Link() string {
	showURL, err := router.Get("articles.show").URL("id", strconv.FormatInt(a.ID, 10))
	if err != nil {
		checkError(err)
		return ""
	}

	return showURL.String()
}

// Delete 删除文章
func (a Article) Delete() (rowsAffected int64, err error) {
	rs, err := db.Exec("delete from articles where id = " + strconv.FormatInt(a.ID, 10))

	if err != nil {
		return 0, err
	}

	if n, _ := rs.RowsAffected(); n > 0 {
		return n, nil
	}

	return 0, err
}

func articlesShowHandler(w http.ResponseWriter, r *http.Request) {
	// 1.获取URL参数
	id := getRouteVariable("id", r)

	// 2. 获取对应文章
	article, err := getArticleByID(id)

	// 3. 错误处理
	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 数据未找到
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章未找到")
		} else {
			// 3.2 数据库未找到
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 服务器内部错误")
		}
	} else {
		// tmpl, err := template.ParseFiles("resources/views/articles/show.gohtml")
		tmpl, err := template.New("show.gohtml").
			Funcs(template.FuncMap{
				"RouteName2URL": RouteName2URL,
				"Int64ToString": Int64ToString,
			}).ParseFiles("resources/views/articles/show.gohtml")
		checkError(err)

		tmpl.Execute(w, article)
	}
}

// RouteName2URL 通过路由名称来获取 URL
func RouteName2URL(routeName string, pairs ...string) string {
	url, err := router.Get(routeName).URL(pairs...)
	if err != nil {
		checkError(err)
		return ""
	}

	return url.String()
}

// Int64ToString 将 int64 转换为 string
func Int64ToString(num int64) string {
	return strconv.FormatInt(num, 10)
}

func articlesIndexHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 执行查询语句，返回文章列表
	rows, err := db.Query("select * from articles")
	checkError(err)
	defer rows.Close()

	var articles []Article

	// 2. 循环读取结果
	for rows.Next() {
		var article Article
		// 2.1 扫描每一行将其赋值到 article 对象中
		err := rows.Scan(&article.ID, &article.Title, &article.Body)
		checkError(err)
		// 2.2 将内容追加到 articles 数组中
		articles = append(articles, article)
	}

	err = rows.Err()
	checkError(err)

	// 加载模板
	tmpl, err := template.ParseFiles("resources/views/articles/index.gohtml")
	checkError(err)

	tmpl.Execute(w, articles)
}

type ArticleFormData struct {
	Title, Body string
	URL         *url.URL
	Errors      map[string]string
}

func articlesStoreHandler(w http.ResponseWriter, r *http.Request) {
	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	errors := validateArticleFormData(title, body)

	// 检查是否有错误
	if len(errors) == 0 {
		lastInsertID, err := saveArticleToDb(title, body)
		if lastInsertID > 0 {
			fmt.Fprint(w, "博客保存成功，返回ID为"+strconv.FormatInt(lastInsertID, 10))
		} else {
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 服务器内部错误")
		}
	} else {
		storeURL, _ := router.Get("articles.store").URL()
		data := ArticleFormData{
			Title:  title,
			Body:   body,
			URL:    storeURL,
			Errors: errors,
		}
		tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
		if err != nil {
			panic(err)
		}

		tmpl.Execute(w, data)
	}
}

func validateArticleFormData(title, body string) map[string]string {

	errors := make(map[string]string)

	// 验证标题

	if title == "" {
		errors["title"] = "标题不能为空"
	} else if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 40 {
		errors["title"] = "标题长度需介于 3-40"
	}

	// 验证内容
	if body == "" {
		errors["body"] = "内容不能为空"
	} else if utf8.RuneCountInString(body) < 10 {
		errors["body"] = "内容长度需大于或等于 10 个字节"
	}

	return errors
}

func saveArticleToDb(title string, body string) (int64, error) {
	// 初始化变量
	var (
		id   int64
		err  error
		rs   sql.Result
		stmt *sql.Stmt
	)

	// 1. 获取一个prepare声明语句
	stmt, err = db.Prepare("INSERT INTO articles (title, body) VALUES(?,?)")
	// 例行的错误
	if err != nil {
		return 0, err
	}
	// 在此函数运行结束后关闭链接，防止占用sql链接
	defer stmt.Close()

	// 3. 执行请求，传参进绑定的内容
	rs, err = stmt.Exec(title, body)
	if err != nil {
		return 0, err
	}
	// 插入成功，返回自增ID
	if id, err = rs.LastInsertId(); id > 0 {
		return id, nil
	}

	return 0, err
}

func forceHTMLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// 1 设置标头
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		// 2 继续请求
		next.ServeHTTP(rw, r)
	})
}

func removeTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// 将首页排除在外
		if r.URL.Path != "/" {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}
		next.ServeHTTP(rw, r)
	})
}

func getRouteVariable(paramName string, r *http.Request) string {
	vars := mux.Vars(r)
	return vars[paramName]
}

func getArticleByID(id string) (Article, error) {
	article := Article{}
	query := "SELECT * FROM articles WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Body)

	return article, err
}

func articleCreateHandle(rw http.ResponseWriter, r *http.Request) {
	storeURL, _ := router.Get("articles.store").URL()
	data := ArticleFormData{
		Title:  "",
		Body:   "",
		URL:    storeURL,
		Errors: nil,
	}
	tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
	if err != nil {
		panic(err)
	}

	tmpl.Execute(rw, data)
}

func articleEditHandle(w http.ResponseWriter, r *http.Request) {
	// 1.获取URL参数
	id := getRouteVariable("id", r)

	// 2. 获取对应文章
	article, err := getArticleByID(id)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章未找到")
		} else {
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 服务器内部错误")
		}
	} else {
		updateURL, _ := router.Get("articles.update").URL("id", id)
		data := ArticleFormData{
			Title:  article.Title,
			Body:   article.Body,
			URL:    updateURL,
			Errors: nil,
		}
		tmpl, err := template.ParseFiles("resources/views/articles/edit.gohtml")
		checkError(err)

		tmpl.Execute(w, data)
	}
}

func articleUpdateHandle(w http.ResponseWriter, r *http.Request) {
	// 获取更新ID
	id := getRouteVariable("id", r)
	// 获取对应的文章数据
	_, err := getArticleByID(id)

	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 数据未找到
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章未找到")
		} else {
			// 3.2 数据库错误
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 服务器内部错误")
		}
	} else {
		title := r.PostFormValue("title")
		body := r.PostFormValue("body")

		errors := validateArticleFormData(title, body)

		if len(errors) == 0 {
			// 验证通过，报错数据
			query := "update articles set title = ?, body = ? where id = ?"
			rs, err := db.Exec(query, title, body, id)
			if err != nil {
				checkError(err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "500 服务器内部错误")
			}

			// √ 更新成功，跳转到文章详情页
			if n, _ := rs.RowsAffected(); n > 0 {
				showURL, _ := router.Get("articles.show").URL("id", id)
				http.Redirect(w, r, showURL.String(), http.StatusFound)
			} else {
				fmt.Fprint(w, "未做任何更改！")
			}
		} else {
			// 4.3 表单验证不通过，显示理由
			updateURL, _ := router.Get("articles.update").URL("id", id)
			data := ArticleFormData{
				Title:  title,
				Body:   body,
				URL:    updateURL,
				Errors: errors,
			}

			tmpl, err := template.ParseFiles("resources/views/articles/edit.gohtml")
			checkError(err)

			tmpl.Execute(w, data)
		}
	}
}

func articleDeleteHandle(w http.ResponseWriter, r *http.Request) {
	id := getRouteVariable("id", r)

	article, err := getArticleByID(id)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章不存在")
		} else {
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 服务器内部错误")
		}
	} else {
		rowsAffected, err := article.Delete()

		if err != nil {
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "服务器内部错误")
		} else {
			if rowsAffected > 0 {
				// 重定向到文章列表页
				indexURL, _ := router.Get("articles.index").URL()
				http.Redirect(w, r, indexURL.String(), http.StatusFound)
			} else {
				// Edge case
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, "404 文章未找到")
			}
		}
	}
}

func main() {
	initDB()
	createTables()

	router.HandleFunc("/", homeHandler).Methods("GET").Name("home")
	router.HandleFunc("/about", aboutHandler).Methods("GET").Name("about")
	router.HandleFunc("/articles/{id:[0-9]+}", articlesShowHandler).Methods("GET").Name("articles.show")
	router.HandleFunc("/articles", articlesIndexHandler).Methods("GET").Name("articles.index")
	router.HandleFunc("/articles", articlesStoreHandler).Methods("POST").Name("articles.store")
	router.HandleFunc("/articles/create", articleCreateHandle).Methods("GET").Name("articles.create")
	router.HandleFunc("/articles/{id:[0-9]+}/edit", articleEditHandle).Methods("Get").Name("articles.edit")
	router.HandleFunc("/articles/{id:[0-9]+}", articleUpdateHandle).Methods("Post").Name("articles.update")
	router.HandleFunc("/articles/{id:[0-9]+}/delete", articleDeleteHandle).Methods("Post").Name("articles.delete")

	// 自定义404页面
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	router.Use(forceHTMLMiddleware)

	http.ListenAndServe(":3000", removeTrailingSlash(router))
}
