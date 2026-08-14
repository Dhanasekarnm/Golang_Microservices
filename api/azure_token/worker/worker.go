package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"pythonrpa/auth"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"strings"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Worker API is available!\n")
}

func CreateWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createworker request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	var t createWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var resp []ApiResponse
	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	workerId := strings.TrimSuffix(string(uuid), "\n")
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "createWorker"
	cfg.User = t.User
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	privateKey, publicKey := auth.GenerateRSAkey()
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client := elasticsearch.ESclient()
	wbody := workerBody{
		TenantUniqueName: t.TenantUniqueName,
		WorkerMode:       t.WorkerMode,
		WorkerPoolId:     t.WorkerPoolId,
		NfsStoragePath:   t.NfsStoragePath,
		State:            "idle",
		WorkerLabel:      "cloud worker",
		PrivateKey:       privateKey,
		PublicKey:        publicKey,
		Timestamp:        timestamp,
	}
	wdata, _ := json.Marshal(wbody)
	res, err := client.Index("automate-orch-workers", bytes.NewReader(wdata), client.Index.WithDocumentID(workerId), client.Index.WithPretty())
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	//randomtext := res.Header["Location"][0]
	//randomtext = strings.TrimPrefix(randomtext, "/automate-orch-workers/_doc/")
	randomtext := workerId
	chiphertext := auth.GenerateChipherText(randomtext, privateKey)
	cfg.Level = "info"
	cfg.Message = "WorkerId '" + randomtext + "' Created. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	res1, err := client.Update(
		"automate-orch-workers",
		randomtext,
		strings.NewReader(`{"doc": {				
				"chipherText":"`+chiphertext+`",
				"randomText":"`+randomtext+`",																		
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
	res1.Body.Close()
	cfg.Message = "WorkerID '" + randomtext + "' chiphertext updated status: " + res1.Status()
	sendlog = operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	header := "Id:" + workerId
	if res.StatusCode == 201 {
		resbody = "Worker created sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: header, Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func GetWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getWorker request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.WorkerId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
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
		resbody := "No records found for WorkerID: " + t.WorkerId
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var Worker_list []listWorker
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Workerlist := listWorker{WorkerId: id, Source: source}
		Worker_list = append(Worker_list, Workerlist)
	}
	var resbody []listWorker
	if res.StatusCode == 200 {
		resbody = Worker_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func UpdateWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body of /updateworker request", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	var t updateWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "updateWorker"
	cfg.User = t.User
	cfg.Level = "info"
	cfg.TenantUniqueName = t.TenantUniqueName
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-workers",
		t.WorkerId,
		strings.NewReader(`{"doc": {								
				"tenantUniqueName":"`+t.TenantUniqueName+`",
				"workerMode":"`+t.WorkerMode+`",
				"workerPoolId":"`+t.WorkerPoolId+`",
				"nfsStoragePath":"`+t.NfsStoragePath+`",				
				"owner":"`+t.User+`",										
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
	cfg.Message = "WorkerId " + t.WorkerId + " updated. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Worker Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DeleteWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body of /deleteworker request", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	var t deleteWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "deleteWorker"
	cfg.User = t.User
	cfg.Level = "info"
	cfg.TenantUniqueName = t.TenantUniqueName
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.WorkerId + `" }},
	{ "term": { "state": "idle" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
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
		resbody := " WorkerID: " + t.WorkerId + " not found with Idle, please try again later"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1, err := client.Delete(
		"automate-orch-workers",
		t.WorkerId,
		client.Delete.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res1.Body.Close()
	cfg.Message = "WorkerId " + t.WorkerId + "' Deleted. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Worker Deleted sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
	// var buf bytes.Buffer
	// if err := json.NewEncoder(&buf).Encode(query); err != nil {
	// 	log.Fatal("Error on json.NewEncoder()")
	// }
	// body1 := strings.NewReader(buf.String())
	// // DeleteByQuery request
	// res, err := client.DeleteByQuery([]string{"automate-orch-workers"}, body1)
	// log.Println(res)
	// if err != nil {
	// 	log.Println("Elasticsearch DeleteByQuery() API ERROR")
	// 	return
	// }
	// defer res.Body.Close()
	// if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
	// 	log.Println("Error parsing the response body,")
	// 	return
	// }
}

func ListWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /ListWorker request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{"size":100, "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
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
		resbody := "No records found in Worker Index"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var Worker_list []listWorker
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Workerlist := listWorker{WorkerId: id, Source: source}
		Worker_list = append(Worker_list, Workerlist)
	}
	var resbody []listWorker
	if res.StatusCode == 200 {
		resbody = Worker_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func WorkerCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /ListWorker request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getWorker
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var mapResp map[string]interface{}
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var query = `{"size":100, "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
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
		resbody := "No records found in Worker Index"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	m := 0
	o := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		state := hit.(map[string]interface{})["_source"].(map[string]interface{})["state"].(string)
		if state == "Idle" || state == "idle" {
			m++
		} else if state == "running" || state == "Running" {
			o++
		}
	}
	var workercount []worker
	workercount = append(workercount, worker{TotalWorker: n, IdleWorker: m, RunningWorker: o})
	var resbody []worker
	if res.StatusCode == 200 {
		resbody = workercount
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}
