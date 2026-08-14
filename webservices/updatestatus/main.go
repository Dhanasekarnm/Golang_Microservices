package main

import (
	"log"
	"net/http"
)

func main() {
	router := NewRouter()
	log.Println("Access the API at http://localhost:8085/updatestatus")
	http.ListenAndServe(":8085", router)
}
