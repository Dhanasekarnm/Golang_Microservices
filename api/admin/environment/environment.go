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
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Environment API is available!\n")
}

func CheckEnvironmentUniqueName(uniquename string, tenantuniquename string) (exists bool) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + tenantuniquename + `" }},
	{ "term": { "environmentUniqueName": "` + uniquename + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		log.Println("environmentuniquename:" + uniquename + " not exist")
		return
	}
	return exist
}

func CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createEnviroment request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t createEnvironment
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	envuniquename := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(t.EnvironmentLabel, ""))
	envuniquename = strings.ReplaceAll(envuniquename, " ", "")
	exist := CheckEnvironmentUniqueName(envuniquename, t.TenantUniqueName)
	if exist {
		resbody := "EnvironmentUniqueName " + envuniquename + " already exist"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp1)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "createEnvironment"
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = envuniquename
	cfg.User = t.Owner
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	resbody := ""
	client := elasticsearch.ESclient()

	if t.Owner == "rpa.dxc.ia@dxc.com" {
		res, err := client.Index(
			"automate-orch-environments",
			strings.NewReader(`{				
				"environmentLabel":"`+t.EnvironmentLabel+`",
				"tenantUniqueName":"`+t.TenantUniqueName+`",
				"environmentUniqueName":"`+envuniquename+`",
				"state":"active",									
				"createdDate":"`+timestamp+`",													
				"timestamp":"`+timestamp+`",					
				"roles": {
				"owner": ["rpa.dxc.ia@dxc.com"],
				"developer": ["rpa.dxc.ia@dxc.com"],
				"runOnly":["rpa.dxc.ia@dxc.com"]
				}
        	}`),
			client.Index.WithPretty(),
		)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := ApiResponse{Response: err}
			resp = append(resp, response)
			resp1 := resp[len(resp)-1]
			json.NewEncoder(w).Encode(resp1)
			return
		}
		cfg.Message = "Environment '" + t.EnvironmentLabel + "' created. Response status:" + res.Status()
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
		defer res.Body.Close()

		header := res.Header["Location"][0]
		header = "Environment Id:" + strings.TrimPrefix(header, "/automate-orch-environments/_doc/")
		if res.StatusCode == 201 {
			resbody = header + " Created successfully"
		}
	} else {
		res, err := client.Index(
			"automate-orch-environments",
			strings.NewReader(`{				
				"environmentLabel":"`+t.EnvironmentLabel+`",
				"tenantUniqueName":"`+t.TenantUniqueName+`",
				"environmentUniqueName":"`+envuniquename+`",
				"state":"active",									
				"createdDate":"`+timestamp+`",													
				"timestamp":"`+timestamp+`",					
				"roles": {
				"owner": ["rpa.dxc.ia@dxc.com","`+t.Owner+`"],
				"developer": ["rpa.dxc.ia@dxc.com","`+t.Owner+`"],
				"runOnly":["rpa.dxc.ia@dxc.com","`+t.Owner+`"]
				}
        	}`),
			client.Index.WithPretty(),
		)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := ApiResponse{Response: err}
			resp = append(resp, response)
			resp1 := resp[len(resp)-1]
			json.NewEncoder(w).Encode(resp1)
			return
		}
		cfg.Message = "Environment '" + t.EnvironmentLabel + "' created. Response status:" + res.Status()
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
		defer res.Body.Close()
		header := res.Header["Location"][0]
		header = "Environment Id:" + strings.TrimPrefix(header, "/automate-orch-environments/_doc/")
		if res.StatusCode == 201 {
			resbody = header + " Created successfully"
		}
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp1)
}

func GetEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getEnvironment request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getEnvironment
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "getEnvironment"
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Level = "info"
	var mapResp map[string]interface{}
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }},
	 { "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}

	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := "No records found for Environment: " + t.EnvironmentUniqueName
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp1)
		return
	}
	var Env_list []listEnvironment
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Envlist := listEnvironment{EnvironmentId: id, Source: source}
		Env_list = append(Env_list, Envlist)
	}
	resp1 := Env_list[len(Env_list)-1]
	cfg.Message = "Environment '" + t.EnvironmentUniqueName + "' created. Response status:" + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp1)
}

func UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /updateEnvironment request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironment
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "updateEnvironment"
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.User = t.Owner
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-environments",
		t.EnvironmentId,
		strings.NewReader(`{"doc": {																			
				"state":"`+t.State+`",											
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	res.Body.Close()
	cfg.Message = "Environment " + t.EnvironmentUniqueName + " updated. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Environment Updated successfully"
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp1)
}

func DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deleteenvironment request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteEnvironment
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "deleteEnvironment"
	cfg.User = t.DeletedBy
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var mapResp map[string]interface{}
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var query1 = `{ "query":{"bool":{"must":[
		{ "term": { "_id": "` + t.EnvironmentId + `" }}
		]}}
		}`
	res1, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query1)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", strconv.Itoa(res1.StatusCode))
		json.NewEncoder(w).Encode(resp1)
		return
	}
	defer res1.Body.Close()
	if err := json.NewDecoder(res1.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	m := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if m == 0 {
		log.Println("No records found for Environment ID: " + t.EnvironmentId)
		return
	}
	exist := false
	var owner string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "owner" {
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					owner = hit1.([]interface{})[j].(string)
					if owner == t.DeletedBy {
						exist = true
					}
				}
			}
		}
	}
	if !exist {
		resbody := "Only Environments Owner can delete the environments"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", strconv.Itoa(res1.StatusCode))
		json.NewEncoder(w).Encode(resp1)
		return
	}
	var query = `{"query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	 { "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }},
	  { "term": { "state": "active" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}

	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n > 0 {
		resbody := "Active Bots found for Environment: " + t.EnvironmentUniqueName + " try again later"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp1)
		return
	}
	// res2, err := client.Delete(
	// 	"automate-orch-environments",
	// 	t.EnvironmentId,
	// 	client.Delete.WithPretty(),
	// )
	res2, err := client.Update(
		"automate-orch-environments",
		t.EnvironmentId,
		strings.NewReader(`{"doc": {																			
				"state":"deleted",											
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		w.Header().Set("StatusCode", strconv.Itoa(res2.StatusCode))
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	res2.Body.Close()
	cfg.Message = "Environment '" + t.EnvironmentUniqueName + "' deleted with following Status: " + res2.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res2.StatusCode == 200 {
		resbody = "Environment Deleted successfully"
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(res2.StatusCode))
	json.NewEncoder(w).Encode(resp1)
}

func ListEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listEnvironment request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getEnvironment
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "listEnvironment"
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Level = "info"
	var mapResp map[string]interface{}
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }}	
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := "No records found in Environments Index"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp1)
		return
	}
	var Env_list []listEnvironment
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Envlist := listEnvironment{EnvironmentId: id, Source: source}
		Env_list = append(Env_list, Envlist)
		//
	}
	response := ApiResponse{Response: Env_list}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	//resp1 := Env_list[len(Env_list)-1]
	cfg.Message = "List Environment Response status:" + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp1)
}

func EnvironmentCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listEnvironment request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getEnvironment
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var mapResp map[string]interface{}
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := t.TenantUniqueName + " Not found"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp1)
		return
	}
	m := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		state := hit.(map[string]interface{})["_source"].(map[string]interface{})["state"].(string)
		if state == "active" {
			m++
		}
	}
	var environmentcount []environment
	environmentcount = append(environmentcount, environment{TotalEnvironment: n, ActiveEnvironment: m})
	resp1 := environmentcount[len(environmentcount)-1]
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp1)
}

func AddEnvironmentOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /AddEnvRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironmentRoles
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	own := strings.ReplaceAll(t.Owner, " ", "")
	if own == "" {
		resbody := "Environment Owner details should not be empty"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", "400")
		json.NewEncoder(w).Encode(resp1)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addEnvironmentOwner"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	var status string
	var statuscode int
	if strings.Contains(own, ",") {
		owners := strings.Split(own, ",")
		for i := 0; i < len(owners); i++ {
			own := owners[i]
			status, statuscode = AddEnvOwner(t.EnvironmentId, own)
			if statuscode == 400 {
				break
			}
		}
	} else {
		status, statuscode = AddEnvOwner(t.EnvironmentId, own)
	}
	cfg.Message = "Add Environmentowner response status: " + status
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	response := ApiResponse{Response: status}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(statuscode))
	json.NewEncoder(w).Encode(resp1)
}

func AddEnvOwner(EnvironmentId string, own string) (status string, statuscode int) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{"query":{"bool":{"must":[
	{ "term": { "_id": "` + EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		status = "No records found for Environment ID: " + EnvironmentId
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	var owner_list, developer_list, runonly_list []string
	var owner, developer, runonly string
	exist := false
	exist1 := false
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "owner" {
				owner_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					owner = hit1.([]interface{})[j].(string)
					if owner == own {
						status = own + " already exist in the Environment Owner"
						statuscode = http.StatusBadRequest
						return status, statuscode
					}
					owner_list = append(owner_list, "\""+owner+"\",")
				}
				owner_list = append(owner_list, "\""+own+"\"")
				owner = fmt.Sprint(owner_list)
			}
		}
	}
	for _, hit2 := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit2.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "developer" {
				developer_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					developer = hit1.([]interface{})[j].(string)
					if developer == own {
						exist = true
					}
					developer_list = append(developer_list, "\""+developer+"\",")

				}
				developer_list = append(developer_list, "\""+own+"\"")
				developer = fmt.Sprint(developer_list)
			}
		}
	}
	for _, hit3 := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit3.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "runOnly" {
				runonly_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					runonly = hit1.([]interface{})[j].(string)
					if runonly == own {
						exist1 = true
					}
					runonly_list = append(runonly_list, "\""+runonly+"\",")
				}
				runonly_list = append(runonly_list, "\""+own+"\"")
				runonly = fmt.Sprint(runonly_list)
			}
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	stringreader := `{}`
	if !exist && !exist1 {
		stringreader = `{"doc":{
			"timestamp":"` + timestamp + `",
			"roles": 
				{
				"owner": ` + owner + `,
				"developer": ` + developer + `,	
				"runOnly": ` + runonly + `								
				}
  		}}`
	} else if !exist && exist1 {
		stringreader = `{"doc":{
			"timestamp":"` + timestamp + `",
			"roles": 
				{
				"owner": ` + owner + `,
				"developer": ` + developer + `											
				}
  		}}`
	} else if exist && !exist1 {
		stringreader = `{"doc":{
			"timestamp":"` + timestamp + `",
			"roles": 
				{
				"owner": ` + owner + `,
				"runOnly": ` + runonly + `											
				}
  		}}`
	} else {
		stringreader = `{"doc":{
			"timestamp":"` + timestamp + `",
			"roles": 
				{
				"owner": ` + owner + `												
				}
  		}}`
	}
	res1, err := client.Update(
		"automate-orch-environments",
		EnvironmentId,
		strings.NewReader(stringreader),
		client.Update.WithPretty(),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	time.Sleep(1 * time.Second)
	res1.Body.Close()
	status = "Environment Owner Added successfully"
	statuscode = http.StatusOK
	return status, statuscode
}

func AddDeveloper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /AddEnvRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironmentRoles
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}

	dev := strings.ReplaceAll(t.Developer, " ", "")
	if dev == "" {
		resbody := "Developer details should not be empty"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", "400")
		json.NewEncoder(w).Encode(resp1)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addDeveloper"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	var status string
	var statuscode int
	if strings.Contains(dev, ",") {
		devlopers := strings.Split(dev, ",")
		for i := 0; i < len(devlopers); i++ {
			dev := devlopers[i]
			status, statuscode = AddDev(t.EnvironmentId, dev)
			if statuscode == 400 {
				break
			}
		}
	} else {
		status, statuscode = AddDev(t.EnvironmentId, dev)
	}
	cfg.Message = "AddDeveloper response status: " + status
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	response := ApiResponse{Response: status}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(statuscode))
	json.NewEncoder(w).Encode(resp1)
}

func AddDev(EnvironmentId string, dev string) (status string, statuscode int) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		status = "No records found for Environment ID: " + EnvironmentId
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	var developer_list, runonly_list []string
	var developer, runonly string
	exist := false
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "developer" {
				developer_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					developer = hit1.([]interface{})[j].(string)
					if developer == dev {
						status = dev + " already exist in Developer"
						statuscode = http.StatusBadRequest
						return status, statuscode
					}
					developer_list = append(developer_list, "\""+developer+"\",")
				}
				developer_list = append(developer_list, "\""+dev+"\"")

				developer = fmt.Sprint(developer_list)
			}
		}
	}
	for _, hit2 := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit3 := range hit2.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "runOnly" {
				runonly_list = nil
				i := len(hit3.([]interface{}))
				for j := 0; j < i; j++ {
					runonly = hit3.([]interface{})[j].(string)
					if runonly == dev {
						exist = true
					}
					runonly_list = append(runonly_list, "\""+runonly+"\",")
				}
				runonly_list = append(runonly_list, "\""+dev+"\"")
				runonly = fmt.Sprint(runonly_list)
			}
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	stringreader := `{}`
	if !exist {
		stringreader = `{"doc":{
				"timestamp":"` + timestamp + `",
				"roles": 
					{				
					"developer": ` + developer + `,
					"runOnly": ` + runonly + `								
					}
			  }}`
	} else {
		stringreader = `{"doc":{
				"timestamp":"` + timestamp + `",
				"roles": 
					{				
					"developer": ` + developer + `					
					}
			  }}`
	}
	res1, err := client.Update(
		"automate-orch-environments",
		EnvironmentId,
		strings.NewReader(stringreader),
		client.Update.WithPretty(),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	time.Sleep(1 * time.Second)
	res1.Body.Close()
	status = "Developer Added successfully"
	statuscode = http.StatusOK
	return status, statuscode
}

func AddRunOnly(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /AddEnvRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironmentRoles
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	run := strings.ReplaceAll(t.RunOnly, " ", "")
	if run == "" {
		resbody := "Runonly details should not be empty"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", "400")
		json.NewEncoder(w).Encode(resp1)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addRunOnly"
	cfg.User = t.RunOnly
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	var status string
	var statuscode int
	if strings.Contains(run, ",") {
		runner := strings.Split(run, ",")
		for i := 0; i < len(runner); i++ {
			run := runner[i]
			status, statuscode = AddRun(t.EnvironmentId, run)
			if statuscode == 400 {
				break
			}
		}
	} else {
		status, statuscode = AddRun(t.EnvironmentId, run)
	}
	cfg.Message = "Add Runonly user response status: " + status
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	response := ApiResponse{Response: status}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(statuscode))
	json.NewEncoder(w).Encode(resp1)
}

func AddRun(EnvironmentId string, run string) (status string, statuscode int) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		status = "No records found for Environment ID: " + EnvironmentId
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	var runonly_list []string
	var runonly string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "runOnly" {
				runonly_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					runonly = hit1.([]interface{})[j].(string)
					if runonly == run {
						status = run + " already exist in runonly user"
						statuscode = http.StatusBadRequest
						return status, statuscode
					}
					runonly_list = append(runonly_list, "\""+runonly+"\",")
				}
				runonly_list = append(runonly_list, "\""+run+"\"")
				runonly = fmt.Sprint(runonly_list)
			}
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		EnvironmentId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
				{				
				"runOnly":`+runonly+`				
				}
  		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	time.Sleep(1 * time.Second)
	res1.Body.Close()
	status = "RunOnly Added successfully"
	statuscode = http.StatusOK
	return status, statuscode
}

func RemoveEnvironmentOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /RemoveEnvRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironmentRoles
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeEnvironmentOwner"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	own := strings.ReplaceAll(t.Owner, " ", "")
	if own == "" {
		resbody := "Environment Owner details should not be empty"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", "400")
		json.NewEncoder(w).Encode(resp1)
		return
	}
	var status string
	var statuscode int
	if strings.Contains(own, ",") {
		owners := strings.Split(own, ",")
		for i := 0; i < len(owners); i++ {
			own := owners[i]
			status, statuscode = RemoveEnvOwner(t.EnvironmentId, own)
			if statuscode == 400 {
				break
			}
		}
	} else {
		status, statuscode = RemoveEnvOwner(t.EnvironmentId, own)
	}
	cfg.Message = "RemoveEnvironmentowner reseponse status: " + status
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	response := ApiResponse{Response: status}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(statuscode))
	json.NewEncoder(w).Encode(resp1)
}

func RemoveEnvOwner(EnvironmentId string, own string) (status string, statuscode int) {
	if own == "rpa.dxc.ia@dxc.com" {
		status := "You cannot remove rpa admin account from Environment Owners"
		statuscode := 400
		return status, statuscode
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{"query":{"bool":{"must":[
	{ "term": { "_id": "` + EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}

	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		status = "No records found for EnvironmentId: " + EnvironmentId
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	var owner_list []string
	var owner string
	exist := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "owner" {
				owner_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					owner = hit1.([]interface{})[j].(string)
					if owner == own {
						exist = 1
					} else {
						if j == 0 {
							owner_list = append(owner_list, "\""+owner+"\"")
						} else if j == 1 && exist == 1 {
							owner_list = append(owner_list, "\""+owner+"\"")
						} else {
							owner_list = append(owner_list, ",\""+owner+"\"")
						}
					}
				}
				owner = fmt.Sprint(owner_list)
			}
		}
	}
	if exist == 0 {
		status = own + " not exist in the Environment Owner"
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		EnvironmentId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
				{
				"owner": `+owner+`								
				}
  		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	time.Sleep(1 * time.Second)
	status = "Environment Owner Removed successfully"
	statuscode = http.StatusOK
	res1.Body.Close()
	return status, statuscode
}

func RemoveDeveloper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /RemoveEnvRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironmentRoles
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	dev := strings.ReplaceAll(t.Developer, " ", "")
	if dev == "" {
		resbody := "Developer details should not be empty"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", "400")
		json.NewEncoder(w).Encode(resp1)
		return
	}

	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeDeveloper"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	var status string
	var statuscode int
	if strings.Contains(dev, ",") {
		developers := strings.Split(dev, ",")
		for i := 0; i < len(developers); i++ {
			dev := developers[i]
			status, statuscode = RemoveDev(t.EnvironmentId, dev)
			if statuscode == 400 {
				break
			}
		}
	} else {
		status, statuscode = RemoveDev(t.EnvironmentId, dev)
	}
	cfg.Message = "RemoveDeveloper response status: " + status
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	response := ApiResponse{Response: status}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(statuscode))
	json.NewEncoder(w).Encode(resp1)
}

func RemoveDev(EnvironmentId string, dev string) (status string, statuscode int) {
	if dev == "rpa.dxc.ia@dxc.com" {
		status := "You cannot remove rpa admin account from Devlopers"
		statuscode := 400
		return status, statuscode
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}

	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		status := "No records found for Environment ID: " + EnvironmentId
		statuscode = http.StatusBadRequest
		return status, statuscode
	}

	var developer_list []string
	var developer string
	exist := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "developer" {
				developer_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					developer = hit1.([]interface{})[j].(string)
					if developer == dev {
						exist = 1
					} else {
						if j == 0 {
							developer_list = append(developer_list, "\""+developer+"\"")
						} else if j == 1 && exist == 1 {
							developer_list = append(developer_list, "\""+developer+"\"")
						} else {
							developer_list = append(developer_list, ",\""+developer+"\"")
						}
					}
				}
				developer = fmt.Sprint(developer_list)
			}
		}
	}
	if exist == 0 {
		status := dev + " not exist in the Environment Developer"
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		EnvironmentId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
				{				
				"developer": `+developer+`							
				}
  		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	time.Sleep(1 * time.Second)
	res1.Body.Close()
	status = "Developer Removed successfully"
	statuscode = http.StatusOK
	return status, statuscode
}

func RemoveRunOnly(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /RemoveEnvRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateEnvironmentRoles
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	run := strings.ReplaceAll(t.RunOnly, " ", "")
	if run == "" {
		resbody := "RunOnly details should not be empty"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		w.Header().Set("StatusCode", "400")
		json.NewEncoder(w).Encode(resp1)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeRunOnly"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	var status string
	var statuscode int
	if strings.Contains(run, ",") {
		runonly := strings.Split(run, ",")
		for i := 0; i < len(runonly); i++ {
			run := runonly[i]
			status, statuscode = RemoveRun(t.EnvironmentId, run)
			if statuscode == 400 {
				break
			}
		}
	} else {
		status, statuscode = RemoveRun(t.EnvironmentId, run)
	}
	cfg.Message = "RemoveRunOnly response status: " + status
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	response := ApiResponse{Response: status}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.Header().Set("StatusCode", strconv.Itoa(statuscode))
	json.NewEncoder(w).Encode(resp1)
}

func RemoveRun(EnvironmentId string, run string) (status string, statuscode int) {
	if run == "rpa.dxc.ia@dxc.com" {
		status := "You cannot remove rpa admin account from Runonly"
		statuscode := 400
		return status, statuscode
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
		{ "term": { "_id": "` + EnvironmentId + `" }}
		]}}
		}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}

	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		status = "No records found for EnvironmentId: " + EnvironmentId
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	var runonly_list []string
	var runonly string
	exist := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{}) {
			if l == "runOnly" {
				runonly_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					runonly = hit1.([]interface{})[j].(string)
					if runonly == run {
						exist = 1
					} else {
						if j == 0 {
							runonly_list = append(runonly_list, "\""+runonly+"\"")
						} else if j == 1 && exist == 1 {
							runonly_list = append(runonly_list, "\""+runonly+"\"")
						} else {
							runonly_list = append(runonly_list, ",\""+runonly+"\"")
						}
					}
				}
				runonly = fmt.Sprint(runonly_list)
			}
		}
	}
	if exist == 0 {
		status = run + " not exist in the RunOnly roles"
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		EnvironmentId,
		strings.NewReader(`{"doc":{
				"timestamp":"`+timestamp+`",
				"roles": 
					{				
					"runOnly":`+runonly+`				
					}
			  }}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		status = err.Error()
		statuscode = http.StatusBadRequest
		return status, statuscode
	}
	time.Sleep(1 * time.Second)
	res1.Body.Close()
	status = "RunOnly Removed successfully"
	statuscode = http.StatusOK
	return status, statuscode
}
