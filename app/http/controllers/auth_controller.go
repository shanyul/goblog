package controllers

import (
	"fmt"
	"goblog/app/models/user"
	"goblog/app/requests"
	"goblog/pkg/auth"
	"goblog/pkg/flash"
	"goblog/pkg/view"
	"net/http"
)

// AuthController 用户控制类
type AuthController struct{}

// Register 用户注册
func (*AuthController) Register(w http.ResponseWriter, r *http.Request) {
	view.RenderSimple(w, view.D{}, "auth.register")
}

// DoRegister 保存用户数据
func (*AuthController) DoRegister(w http.ResponseWriter, r *http.Request) {
	// 1.初始化数据
	_user := user.User{
		Name:            r.PostFormValue("name"),
		Email:           r.PostFormValue("email"),
		Password:        r.PostFormValue("password"),
		PasswordConfirm: r.PostFormValue("password_confirm"),
	}

	// 开始认证
	errs := requests.ValidateRegiterForm(_user)

	if len(errs) > 0 {
		// 发生错误, 打印数据
		view.RenderSimple(w, view.D{
			"Errors": errs,
			"User":   _user,
		}, "auth.register")
	} else {
		_user.Create()
		if _user.ID > 0 {
			 // 登录用户并跳转到首页
			 flash.Success("恭喜您注册成功！")
			 auth.Login(_user)
			 http.Redirect(w, r, "/", http.StatusFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "创建用户失败，请联系管理员")
		}
	}
}

// Login 用户登录
func (*AuthController) Login(w http.ResponseWriter, r *http.Request) {

	view.RenderSimple(w, view.D{}, "auth.login")
}

// DoLogin 登录验证
func (*AuthController) DoLogin(w http.ResponseWriter, r *http.Request) {
	// 1. 初始化表单数据
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	
    // 2. 尝试登录
    if err := auth.Attempt(email, password); err == nil {
        // 登录成功
        flash.Success("欢迎回来！")
        http.Redirect(w, r, "/", http.StatusFound)
    } else {
        // 3. 失败，显示错误提示
        view.RenderSimple(w, view.D{
            "Error":    err.Error(),
            "Email":    email,
            "Password": password,
        }, "auth.login")
    }
}

// Logout 退出登录
func (*AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	auth.Logout()
	flash.Success("您已退出登录")
    http.Redirect(w, r, "/", http.StatusFound)
}
