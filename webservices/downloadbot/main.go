package main

import (
	"log"
	"net/http"
)

func main() {

	router := NewRouter()
	log.Println("Access the API at http://localhost:8083/downloadbot")
	http.ListenAndServe(":8083", router)
}
