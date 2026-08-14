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
	log.Println("Access Environment API at http://localhost:30003")
	http.ListenAndServe(":30003", nil)
}
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createEnvironment request"+Reset, err)
		return
	}
	var t tokenEnvironment
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	if t.Token == "" {
		log.Println("Azure Token is missing, could not access Environment API")
		return
	}
	valid, err := azure.Azure(t.Token)
	if valid {
		switch r.URL.Path {
		case "/":
			Index(w, r)
		case "/getenvironment":
			GetEnvironment(w, r)
		case "/createenvironment":
			CreateEnvironment(w, r)
		case "/updateenvironment":
			UpdateEnvironment(w, r)
		case "/deleteenvironment":
			DeleteEnvironment(w, r)
		case "/listenvironment":
			ListEnvironment(w, r)
		case "/environmentcount":
			EnvironmentCount(w, r)
		case "/addenvironmentowner":
			AddEnvironmentOwner(w, r)
		case "/adddeveloper":
			AddDeveloper(w, r)
		case "/addrunonly":
			AddRunOnly(w, r)
		case "/removeenvironmentowner":
			RemoveEnvironmentOwner(w, r)
		case "/removedeveloper":
			RemoveDeveloper(w, r)
		case "/removerunonly":
			RemoveRunOnly(w, r)
		}
	} else {
		log.Println(err)
	}
}
