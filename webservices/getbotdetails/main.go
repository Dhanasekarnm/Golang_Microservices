package main

import (
	"log"
	"net/http"
)

func main() {
	router := NewRouter()
	log.Println("Access the API at http://localhost:8082/getbotdetails")
	http.ListenAndServe(":8082", router)
}
