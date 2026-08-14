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
	fmt.Fprint(w, "Scheduler API is available!\n")
}

func CreateSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t createSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "createSchedule"
	cfg.User = t.User
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Index(
		"automate-orch-schedules",
		strings.NewReader(`{
		  "tenantUniqueName": "`+t.TenantUniqueName+`",
	      "environmentUniqueName": "`+t.EnvironmentUniqueName+`",
	      "botId": "`+t.BotId+`",
	      "plannedExecutionTime": "`+t.PlannedExecutionTime+`",
		  "endTime": "`+t.EndTime+`",
	      "interval": "`+t.Interval+`",
	      "frequency": "`+t.Frequency+`",
	      "days": `+t.Days+`,
	      "timestamp": "`+timestamp+`",
	      "botMode": "`+t.BotMode+`",
	      "priority": `+t.Priority+`,
	      "machineName": "`+t.MachineName+`",
	      "createdBy": "`+t.User+`",
		  "state": "active"
		}`),
		client.Index.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "createShedule response status" + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	defer res.Body.Close()
	header := res.Header["Location"][0]
	header = "Id:" + strings.TrimPrefix(header, "/automate-orch-schedules/_doc/")
	resbody := ""
	if res.StatusCode == 201 {
		resbody = "Schedule created sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: header, Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func GetSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.ScheduleId + `" }}
	]}}
	}`
	client := elasticsearch.ESclient()
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
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
		resbody := "No records found with scheduleId: " + t.ScheduleId
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var schedule_list []listSchedule
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		schedulelist := listSchedule{ScheduleId: id, Source: source}
		schedule_list = append(schedule_list, schedulelist)
	}
	var resbody []listSchedule
	if res.StatusCode == 200 {
		resbody = schedule_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /UpdateSchedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "updateschedule"
	cfg.User = t.User
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {	  
	      "tenantUniqueName": "`+t.TenantUniqueName+`",
	      "environmentUniqueName": "`+t.EnvironmentUniqueName+`",
	      "botId": "`+t.BotId+`",
	      "plannedExecutionTime": "`+t.PlannedExecutionTime+`",
		  "endTime": "`+t.EndTime+`",
	      "interval": "`+t.Interval+`",
	      "frequency": "`+t.Frequency+`",
	      "days": `+t.Days+`,
	      "timestamp": "`+timestamp+`",
	      "botMode": "`+t.BotMode+`",
	      "priority": "`+t.Priority+`",
	      "machineName": "`+t.MachineName+`",
	      "modifiedBy": "`+t.User+`"		  
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
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' updated. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Schedule Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deleteschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "deleteschedule"
	cfg.User = t.User
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {				
			"state":"deleted",
			"deletedBy":"`+t.User+`",							
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
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' deleted. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	cfg.Level = "info"
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Schedule Deleted sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func ListSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{"size":100, "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	 { "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
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
		resbody := "No records found in Schedule"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var schedule_list []listSchedule
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		schedulelist := listSchedule{ScheduleId: id, Source: source}
		schedule_list = append(schedule_list, schedulelist)
	}
	var resbody []listSchedule
	if res.StatusCode == 200 {
		resbody = schedule_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func EnableSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deleteschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "enableSchedule"
	cfg.User = t.User
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {				
			"state":"active",
			"deletedBy":"`+t.User+`",							
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
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' enabled. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "User Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DisableSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /disableschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "disableSchedule"
	cfg.User = t.User
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {				
			"state":"disabled",
			"deletedBy":"`+t.User+`",							
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
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' disabled. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "User Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}
