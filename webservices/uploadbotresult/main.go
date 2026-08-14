package main

import (
	"log"
	"net/http"
)

func main() {
	router := NewRouter()
	log.Println("Access the API at http://localhost:8084/uploadbotresult")
	http.ListenAndServe(":8084", router)
}
