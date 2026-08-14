package main

import (
	"log"
	"net/http"
)

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[37m"
	White   = "\033[97m"
)

func main() {
	router := NewRouter()
	http.Handle("/", &MyServer{router})
	log.Println("Access Schedule API at http://localhost:30007")
	http.ListenAndServe(":30007", nil)
}
