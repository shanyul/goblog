package main

import (
	"goblog/app/middlewares"
	"goblog/bootstrap"
	"net/http"
	
)

func main() {
	bootstrap.SetupDB()
	router := bootstrap.SetupRoute()

	http.ListenAndServe(":3000", middlewares.RemoveTrailingSlash(router))
}
