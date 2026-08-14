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
	log.Println("Access Tenant API at http://localhost:30002")
	http.ListenAndServe(":30002", nil)
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createTenant request"+Reset, err)
		return
	}
	var t tokenTenant
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	if t.Token == "" {
		log.Println("Azure Token is missing, could not access Tenant API")
		return
	}
	valid, err := azure.Azure(t.Token)
	if valid {
		switch r.URL.Path {
		case "/":
			Index(w, r)
		case "/gettenant":
			GetTenant(w, r)
		case "/createtenant":
			CreateTenant(w, r)
		case "/updatetenant":
			UpdateTenant(w, r)
		case "/deletetenant":
			DeleteTenant(w, r)
		case "/listtenant":
			ListTenant(w, r)
		case "/tenantcount":
			TenantCount(w, r)
		case "/addtenantowner":
			AddTenantOwner(w, r)
		case "/addenvironmentmaker":
			AddEnvironmentMaker(w, r)
		case "/removetenantowner":
			RemoveTenantOwner(w, r)
		case "/removeenvironmentmaker":
			RemoveEnvironmentMaker(w, r)
		}
	} else {
		log.Println(err)
	}
}
