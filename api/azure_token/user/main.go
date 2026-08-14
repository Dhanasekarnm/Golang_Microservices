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
	log.Println("Access User API at http://localhost:30005")
	http.ListenAndServe(":30005", nil)
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createUser request"+Reset, err)
		return
	}
	var t tokenUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	if t.Token == "" {
		log.Println("Azure Token is missing, could not access User API")
		return
	}
	valid, err := azure.Azure(t.Token)
	if valid {
		switch r.URL.Path {
		case "/":
			Index(w, r)
		case "/getuser":
			GetUser(w, r)
		case "/createuser":
			CreateUser(w, r)
		case "/updateuser":
			UpdateUser(w, r)
		case "/deleteuser":
			DeleteUser(w, r)
		case "/listuser":
			ListUser(w, r)
		case "/usercount":
			UserCount(w, r)
		case "/checkuserrole":
			CheckUserRole(w, r)
		}
	} else {
		log.Println(err)
	}
}
