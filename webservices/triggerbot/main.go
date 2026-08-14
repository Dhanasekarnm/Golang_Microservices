package main

import (
	"log"
	"net/http"
)

func main() {
	router := NewRouter()
	log.Println("Access the API at http://localhost:8080/triggerbot")
	http.ListenAndServe(":8080", router)
}
