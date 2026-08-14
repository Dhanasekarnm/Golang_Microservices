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
	"strings"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Environment API is available!\n")
}

func CheckEnvironmentUniqueName(uniquename string) (exists bool) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
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
	exist := CheckEnvironmentUniqueName(envuniquename)
	if exist {
		resbody := "EnvironmentUniqueName " + envuniquename + " already exist"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
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
	client := elasticsearch.ESclient()
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
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "Environment '" + t.EnvironmentLabel + "' created. Response status:" + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	defer res.Body.Close()
	resbody := ""
	header := res.Header["Location"][0]
	header = "Id:" + strings.TrimPrefix(header, "/automate-orch-environments/_doc/")
	if res.StatusCode == 201 {
		resbody = "Environment created sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: header, Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)

}

func GetEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		resbody := "No records found for Environment: " + t.EnvironmentUniqueName
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var Env_list []listEnvironment
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Envlist := listEnvironment{EnvironmentId: id, Source: source}
		Env_list = append(Env_list, Envlist)
	}
	var resbody []listEnvironment
	if res.StatusCode == 200 {
		resbody = Env_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res.Body.Close()
	cfg.Message = "Environment " + t.EnvironmentUniqueName + " updated. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Environment Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
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
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
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
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var query = `{"query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	 { "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
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
	if n > 0 {
		resbody := "Active Bots found for Environment: " + t.EnvironmentUniqueName + " try again later"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res2, err := client.Delete(
		"automate-orch-environments",
		t.EnvironmentId,
		client.Delete.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res2.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "Failed to Delete Environment " + fmt.Sprint(err)
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
		return
	}
	res2.Body.Close()
	cfg.Message = "Environment '" + t.EnvironmentUniqueName + "' deleted with following Status: " + res2.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res2.StatusCode == 200 {
		resbody = "Environment Deleted sucessfully"
	}
	response := ApiResponse{StatusCode: res2.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res2.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func ListEnvironment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		resbody := "No records found in Environments Index"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var Env_list []listEnvironment
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Envlist := listEnvironment{EnvironmentId: id, Source: source}
		Env_list = append(Env_list, Envlist)
	}
	var resbody []listEnvironment
	if res.StatusCode == 200 {
		resbody = Env_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func EnvironmentCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		resbody := t.TenantUniqueName + " Not found"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
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
	var environmentcount []environment
	environmentcount = append(environmentcount, environment{TotalEnvironment: n, ActiveEnvironment: m})
	var resbody []environment
	if res.StatusCode == 200 {
		resbody = environmentcount
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func AddEnvironmentOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addEnvironmentOwner"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{"query":{"bool":{"must":[
	{ "term": { "_id": "` + t.EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		resbody := "No records found for Environment ID: " + t.EnvironmentId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "No records found for Environment ID: " + t.EnvironmentId
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
		return
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
					if owner == t.Owner {
						resbody := t.Owner + " already exist in the Environment Owner"
						response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
						resp = append(resp, response)
						json.NewEncoder(w).Encode(resp)
						return
					}
					owner_list = append(owner_list, "\""+owner+"\",")
				}
				owner_list = append(owner_list, "\""+t.Owner+"\"")
				owner = fmt.Sprint(owner_list)
			} else if l == "developer" {
				developer_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					developer = hit1.([]interface{})[j].(string)
					if developer == t.Owner {
						exist = true
					}
					developer_list = append(developer_list, "\""+developer+"\",")

				}
				developer_list = append(developer_list, "\""+t.Owner+"\"")
				developer = fmt.Sprint(developer_list)
			} else if l == "runOnly" {
				runonly_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					runonly = hit1.([]interface{})[j].(string)
					if runonly == t.Owner {
						exist1 = true
					}
					runonly_list = append(runonly_list, "\""+runonly+"\",")
				}
				runonly_list = append(runonly_list, "\""+t.Owner+"\"")
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
		t.EnvironmentId,
		strings.NewReader(stringreader),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "Failed to Add Environment roles " + fmt.Sprint(err)
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
		return
	}
	res1.Body.Close()
	cfg.Message = "AddEnvironmentowner response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "AddEnvironmentowner Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func AddDeveloper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addDeveloper"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		resbody := "No records found for Environment ID: " + t.EnvironmentId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "No records found for Environment ID: " + t.EnvironmentId
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
		return
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
					if developer == t.Developer {
						resbody := t.Developer + " already exist in Developer"
						response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
						resp = append(resp, response)
						json.NewEncoder(w).Encode(resp)
						return
					}
					developer_list = append(developer_list, "\""+developer+"\",")
				}
				developer_list = append(developer_list, "\""+t.Developer+"\"")

				developer = fmt.Sprint(developer_list)
			} else if l == "runOnly" {
				runonly_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					runonly = hit1.([]interface{})[j].(string)
					if runonly == t.Developer {
						exist = true
					}
					runonly_list = append(runonly_list, "\""+runonly+"\",")
				}
				runonly_list = append(runonly_list, "\""+t.Developer+"\"")
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
		t.EnvironmentId,
		strings.NewReader(stringreader),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1.Body.Close()
	cfg.Message = "AddDeveloper response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "Developer Added sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func AddRunOnly(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addRunOnly"
	cfg.User = t.RunOnly
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		resbody := "No records found for Environment ID: " + t.EnvironmentId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "No records found for Environment ID: " + t.EnvironmentId
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
		return
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
					if runonly == t.RunOnly {
						resbody := t.RunOnly + " already exist in runonly user"
						response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
						resp = append(resp, response)
						json.NewEncoder(w).Encode(resp)
						return
					}
					runonly_list = append(runonly_list, "\""+runonly+"\",")
				}
				runonly_list = append(runonly_list, "\""+t.RunOnly+"\"")
				runonly = fmt.Sprint(runonly_list)
			}
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		t.EnvironmentId,
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
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1.Body.Close()
	cfg.Message = "AddRunonly response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "AddRunonly Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func RemoveEnvironmentOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
	own := strings.ReplaceAll(t.Owner, " ", "")
	if own == "" {
		resbody := "Environment Owner details should not be empty"
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeEnvironmentOwner"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{"query":{"bool":{"must":[
	{ "term": { "_id": "` + t.EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		resbody := "No records found for EnvironmentId: " + t.EnvironmentId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "No records found for EnvironmentId: " + t.EnvironmentId
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
		return
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
					if owner == t.Owner {
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
		resbody := t.Owner + " not exist in the Environment Owner"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		t.EnvironmentId,
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
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1.Body.Close()
	cfg.Message = "RemoveEnvironmentowner reseponse status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "RemoveEnvironmentowner Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func RemoveDeveloper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeDeveloper"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		resbody := "No records found for Environment ID: " + t.EnvironmentId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "No records found for EnvironmentId: " + t.EnvironmentId
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
		return
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
					if developer == t.Developer {
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
		resbody := t.Developer + " not exist in the Environment Developer"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		t.EnvironmentId,
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
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1.Body.Close()
	cfg.Message = "RemoveDeveloper response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "RemoveDeveloper Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func RemoveRunOnly(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeRunOnly"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.EnvironmentId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
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
		resbody := "No records found for EnvironmentId: " + t.EnvironmentId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		cfg.Message = "No records found for EnvironmentId: " + t.EnvironmentId
		cfg.Level = "error"
		sendlog := operations.SendLog(cfg, sendlog_url)
		log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
		return
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
					if runonly == t.RunOnly {
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
		resbody := t.RunOnly + " not exist in the RunOnly roles"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-environments",
		t.EnvironmentId,
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
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1.Body.Close()
	cfg.Message = "RemoveRunOnly response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "RemoveRunOnly Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}
