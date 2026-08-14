package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"strings"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "User API is available!\n")
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createUser request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t createUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "createUser"
	cfg.User = t.Email
	cfg.Level = "info"
	var resp []ApiResponse
	resbody := ""
	exist := CheckUserExist(t.Email)
	if exist {
		resbody = "User email " + t.Email + " already exist"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client := elasticsearch.ESclient()
	res, err := client.Index(
		"automate-orch-users",
		strings.NewReader(`{				
				"email":"`+t.Email+`",				
				"state":"active",	
				"type":"`+t.Type+`",																	
				"timestamp":"`+timestamp+`"
        	}`),
		client.Index.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "createUser response status:" + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	defer res.Body.Close()
	header := res.Header["Location"][0]
	header = "Id:" + strings.TrimPrefix(header, "/automate-orch-users/_doc/")
	if res.StatusCode == 201 {
		resbody = "User created sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: header, Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getUser request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "email.keyword": "` + t.Email + `" }}
	]}}
	}`
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-users"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody1 := "User " + t.Email + " not found"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody1}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var User_list []listUser
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		id := hit.(map[string]interface{})["_id"].(string)
		source := hit.(map[string]interface{})["_source"]
		Userlist := listUser{UserId: id, Source: source}
		User_list = append(User_list, Userlist)
	}
	var resbody []listUser
	if res.StatusCode == 200 {
		resbody = User_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET;POST;OPTIONS")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /UpdateUser request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "updateUser"
	cfg.User = t.Email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-users",
		t.UserId,
		strings.NewReader(`{"doc": {								
				"state":"`+t.State+`",
				"type":"`+t.Type+`",															
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res.Body.Close()
	cfg.Message = "User '" + t.Email + "' updated. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "User Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deleteuser request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "deleteUser"
	cfg.User = t.Email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-users",
		t.UserId,
		strings.NewReader(`{"doc": {				
			"state":"deleted",
			"deletedBy":"`+t.Email+`",							
			"timestamp":"`+timestamp+`"
		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res.Body.Close()
	cfg.Message = "User '" + t.UserId + "' deleted. response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "User Deleted sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func ListUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listuser request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-users"),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: "No records found in User Index"}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var User_list []listUser
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Userlist := listUser{UserId: id, Source: source}
		User_list = append(User_list, Userlist)
	}
	var resbody []listUser
	if res.StatusCode == 200 {
		resbody = User_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func CheckUserRole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listUser request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t checkUserRole
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query1 = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	 { "term": { "roles.tenant.owner.keyword": "` + t.Email + `" }}
	]}}
	}`
	res1, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
		client.Search.WithBody(strings.NewReader(query1)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {

		log.Println("Failed to GET Tenant " + t.TenantUniqueName + fmt.Sprint(err))
		return
	}
	defer res1.Body.Close()
	if err := json.NewDecoder(res1.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	tenantowner := false
	n1 := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n1 == 1 {
		tenantowner = true
	}

	var query2 = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	 { "term": { "roles.environment.maker.keyword": "` + t.Email + `" }}
	]}}
	}`
	res2, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
		client.Search.WithBody(strings.NewReader(query2)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {

		log.Println("Failed to GET Tenant " + t.TenantUniqueName + fmt.Sprint(err))
		return
	}
	defer res2.Body.Close()
	if err := json.NewDecoder(res2.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	envmaker := false
	n2 := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n2 == 1 {
		envmaker = true
	}
	var query3 = `{ "query":{"bool":{"must":[
	{ "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }},
	{ "term": { "roles.owner.keyword": "` + t.Email + `" }}
	]}}
	}`
	res3, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query3)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println("Failed to GET Environment " + t.EnvironmentUniqueName + fmt.Sprint(err))
		return
	}
	defer res3.Body.Close()
	if err := json.NewDecoder(res3.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	envowner := false
	n3 := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n3 == 1 {
		envowner = true
	}
	var query4 = `{ "query":{"bool":{"must":[
		{ "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }},
		{ "term": { "roles.developer.keyword": "` + t.Email + `" }}
		]}}
		}`
	res4, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query4)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println("Failed to GET Environment " + t.EnvironmentUniqueName + fmt.Sprint(err))
		return
	}
	defer res4.Body.Close()
	if err := json.NewDecoder(res4.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	developer := false
	n4 := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n4 == 1 {
		developer = true
	}
	var query5 = `{ "query":{"bool":{"must":[
		{ "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }},
		{ "term": { "roles.runOnly.keyword": "` + t.Email + `" }}
		]}}
		}`
	res5, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query5)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println("Failed to GET Environment " + t.EnvironmentUniqueName + fmt.Sprint(err))
		return
	}
	defer res5.Body.Close()
	if err := json.NewDecoder(res5.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	runonly := false
	n5 := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n5 == 1 {
		runonly = true
	}
	var User_role []userRoleResponse
	userrole := userRoleResponse{TenantOwner: tenantowner, EnvironmentMaker: envmaker, EnvironmentOwner: envowner, Developer: developer, RunOnly: runonly}
	User_role = append(User_role, userrole)
	var resbody []userRoleResponse
	if res5.StatusCode == 200 {
		resbody = User_role
	}
	var resp []ApiResponse
	response := ApiResponse{StatusCode: res5.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res5.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func CheckUserExist(email string) (exists bool) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{
				"match_phrase_prefix" : {
						"email": "` + email + `"
					}
				}}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-users"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println(Red+"Elasticsearch Search() API ERROR: "+Reset, err)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	exist := true
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		exist = false
		log.Println("User " + email + " not exist")
		return
	}
	return exist
}

func UserCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /usercount request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getUser
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-users"),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: "No records found in User Index"}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	m := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		state := hit.(map[string]interface{})["_source"].(map[string]interface{})["state"].(string)
		if state == "active" {
			m++
		}
	}
	var usercount []user
	usercount = append(usercount, user{TotalUser: n, ActiveUser: m})
	var resbody []user
	if res.StatusCode == 200 {
		resbody = usercount
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}
