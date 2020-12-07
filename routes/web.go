package routes

import (
	"goblog/app/http/controllers"
	"goblog/app/middlewares"
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterWebRoutes 注册网页相关路由
func RegisterWebRoutes(r *mux.Router) {

	pc := new(controllers.PagesController)
	// 静态页面
	r.HandleFunc("/about", pc.About).Methods("GET").Name("about")
	r.NotFoundHandler = http.HandlerFunc(pc.NotFound)

	ac := new(controllers.ArticlesController)
	r.HandleFunc("/", ac.Index).Methods("Get").Name("home")
	r.HandleFunc("/articles/{id:[0-9]+}", ac.Show).Methods("GET").Name("articles.show")
	r.HandleFunc("/articles", ac.Index).Methods("GET").Name("articles.index")
	r.HandleFunc("/articles", middlewares.Auth(ac.Store)).Methods("POST").Name("articles.store")
	r.HandleFunc("/articles/create", middlewares.Auth(ac.Create)).Methods("Get").Name("articles.create")
	r.HandleFunc("/articles/{id:[0-9]+}/edit", middlewares.Auth(ac.Edit)).Methods("Get").Name("articles.edit")
	r.HandleFunc("/articles/{id:[0-9]+}", middlewares.Auth(ac.Update)).Methods("Post").Name("articles.update")
	r.HandleFunc("/articles/{id:[0-9]+}/delete", middlewares.Auth(ac.Delete)).Methods("Post").Name("articles.delete")

	// 用户注册
	auth := new(controllers.AuthController)
	r.HandleFunc("/auth/register", middlewares.Guest(auth.Register)).Methods("Get").Name("auth.register")
	r.HandleFunc("/auth/do-register", middlewares.Guest(auth.DoRegister)).Methods("Post").Name("auth.doregister")
	r.HandleFunc("/auth/login", middlewares.Guest(auth.Login)).Methods("Get").Name("auth.login")
	r.HandleFunc("/auth/do-login", middlewares.Guest(auth.DoLogin)).Methods("Post").Name("auth.dologin")
	r.HandleFunc("/auth/logout", middlewares.Guest(auth.Logout)).Methods("Post").Name("auth.logout")

	// 用户认证
    uc := new(controllers.UserController)
    r.HandleFunc("/users/{id:[0-9]+}", uc.Show).Methods("GET").Name("users.show")

	// 静态文件
	r.PathPrefix("/css/").Handler(http.FileServer(http.Dir("./public")))
	r.PathPrefix("/js/").Handler(http.FileServer(http.Dir("./public")))

	// --- 全局中间件 ---

	// 中间件：强制内容类型为 HTML
	// r.Use(middlewares.ForceHTML)

	// 开始会话
	r.Use(middlewares.StartSession)
}
