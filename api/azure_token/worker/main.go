package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"pythonrpa/azure"
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
	log.Println("Access Worker API at http://localhost:30004")
	http.ListenAndServe(":30004", nil)
}
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createWorker request"+Reset, err)
		return
	}
	var t tokenWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	if t.Token == "" {
		log.Println("Azure Token is missing, could not access Worker API")
		return
	}
	valid, err := azure.Azure(t.Token)
	if valid {
		switch r.URL.Path {
		case "/":
			Index(w, r)
		case "/getworker":
			GetWorker(w, r)
		case "/createworker":
			CreateWorker(w, r)
		case "/updateworker":
			UpdateWorker(w, r)
		case "/deleteworker":
			DeleteWorker(w, r)
		case "/listworker":
			ListWorker(w, r)
		case "/workercount":
			WorkerCount(w, r)
		}
	} else {
		log.Println(err)
	}
}
