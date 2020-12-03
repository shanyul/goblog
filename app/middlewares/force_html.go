package middlewares

import "net/http"

// ForceHTML 强制标头返回 HTML 内容类型
func ForceHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// 1 设置标头
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		// 2 继续请求
		next.ServeHTTP(rw, r)
	})
}