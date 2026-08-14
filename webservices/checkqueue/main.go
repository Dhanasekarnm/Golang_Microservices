package main

import (
	"log"
	"net/http"
)

func main() {
	router := NewRouter()
	log.Println("Access the API at http://localhost:8080/checkqueue")
	http.ListenAndServe(":8080", router)
}
