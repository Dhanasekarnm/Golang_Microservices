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
	fmt.Fprint(w, "Schedule API is available!\n")
}

func CreateSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var resp []ApiResponse
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
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
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	cfg.Message = "createShedule response status" + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	defer res.Body.Close()
	header := res.Header["Location"][0]
	header = strings.TrimPrefix(header, "/automate-orch-schedules/_doc/")
	resbody := ""
	if res.StatusCode == 201 {
		resbody = "Schedule created successfully, ScheduleID:" + header
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp1)
}

func GetSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var resp []ApiResponse
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var mapResp map[string]interface{}

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
		resbody := "No records found with scheduleId: " + t.ScheduleId
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		//fmt.Fprintf(w, "No records found with scheduleId: " + t.ScheduleId)
		return
	}
	var schedule_list []listSchedule
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		schedulelist := listSchedule{ScheduleId: id, Source: source}
		schedule_list = append(schedule_list, schedulelist)
	}
	resp1 := schedule_list[len(schedule_list)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp1)
}

func UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var resp []ApiResponse
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
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
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {	  	           
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
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	res.Body.Close()
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' updated. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Schedule Updated successfully"
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp1)
}

func DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE")
	var resp []ApiResponse
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deleteschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
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
	cfg.User = t.Email
	timestamp := time.Now().UTC().Format(time.RFC3339)

	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {				
			"state":"deleted",
			"deletedBy":"`+t.Email+`",							
			"timestamp":"`+timestamp+`"
		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	res.Body.Close()
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' deleted. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	cfg.Level = "info"
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := "Error in Schedule Delete"
	if res.StatusCode == 200 {
		resbody = "Schedule Deleted successfully"
	} else if res.StatusCode == 400 {
		resbody = "Bad request, Schedule Not Deleted"
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp1)
}

func ListSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var resp []ApiResponse
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	var t getSchedule
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{"size":100, "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	{ "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }},
	{ "term": { "botId.keyword": "` + t.BotId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
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
		resbody := "No records found in Schedules"
		response := ApiResponse{Response: resbody}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		//fmt.Fprintf(w, "No records found with schedules")
		return
	}
	var schedule_list []listSchedule
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		schedulelist := listSchedule{ScheduleId: id, Source: source}
		schedule_list = append(schedule_list, schedulelist)
	}
	response := ApiResponse{Response: schedule_list}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	//resp1 := schedule_list[len(schedule_list)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp1)
}

func EnableSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var resp []ApiResponse
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deleteschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
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
	cfg.User = t.Email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)

	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {				
			"state":"active",	
			"createdBy":"`+t.Email+`",																	
			"timestamp":"`+timestamp+`"
		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	res.Body.Close()
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' enabled. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := "Error in Schedule Update"
	if res.StatusCode == 200 {
		resbody = "Schedule Updated successfully"
	} else if res.StatusCode == 400 {
		resbody = "Bad request, Schedule Not Updated"
	}
	response := ApiResponse{Response: resbody}
	resp = append(resp, response)
	resp1 := resp[len(resp)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp1)
}

func DisableSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	var resp []ApiResponse
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /disableschedule request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
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
	cfg.User = t.Email
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-schedules",
		t.ScheduleId,
		strings.NewReader(`{"doc": {				
			"state":"disabled",
			"createdBy":"`+t.Email+`",									
			"timestamp":"`+timestamp+`"
		}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{Response: err}
		resp = append(resp, response)
		resp1 := resp[len(resp)-1]
		json.NewEncoder(w).Encode(resp1)
		return
	}
	res.Body.Close()
	cfg.Message = "ScheduleId '" + t.ScheduleId + "' disabled. Response Status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resp1 := "Error in Schedule Update"
	if res.StatusCode == 200 {
		resp1 = "Schedule Updated successfully"
	} else if res.StatusCode == 400 {
		resp1 = "Bad request, Schedule Not Updated"
	}
	response := ApiResponse{Response: resp1}
	resp = append(resp, response)
	resp2 := resp[len(resp)-1]
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp2)
}
