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
	log.Println("Access Bot API at http://localhost:30006")
	http.ListenAndServe(":30006", nil)
}
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createBot request"+Reset, err)
		return
	}
	var t tokenBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	if t.Token == "" {
		log.Println("Azure Token is missing, could not access Bot API")
		return
	}
	valid, err := azure.Azure(t.Token)
	if valid {
		switch r.URL.Path {
		case "/":
			Index(w, r)
		case "/getbot":
			GetBot(w, r)
		case "/createbot":
			CreateUpload(w, r)
		case "/updatebot":
			UpdateUpload(w, r)
		case "/deletebot":
			DeleteBot(w, r)
		case "/deletebotpermenant":
			DeleteBotPermenant(w, r)
		case "/listbot":
			ListBot(w, r)
		case "/enablebot":
			EnableBot(w, r)
		case "/disablebot":
			DisableBot(w, r)
		}
	} else {
		log.Println(err)
	}
}
