package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"pythonrpa/s3storage"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Tenant API is available!\n")
}

func CheckTenantUniqueName(uniquename string) (exists bool) {

	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + uniquename + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		log.Println("tenantunquename: " + uniquename + " not exist")
		return
	}
	return exist
}

func CreateTenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /createTenant request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t createTenant
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}

	uniquename := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(t.TenantLabel, ""))
	uniquename = strings.ReplaceAll(uniquename, " ", "")
	if len(uniquename) < 3 || len(uniquename) > 40 || strings.Contains(uniquename, "..") || strings.HasPrefix(uniquename, "xn") || strings.HasSuffix(uniquename, "sthree") || strings.HasSuffix(uniquename, "s3") {
		log.Println("Invalid s3bucket Name")
		return
	}
	exist := CheckTenantUniqueName(uniquename)
	if exist {
		resbody := "TenantUniqueName " + uniquename + " already exist"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "createTenant"
	cfg.Level = "info"
	cfg.TenantUniqueName = uniquename
	cfg.User = t.Owner
	s := s3storage.S3Connect()
	bucket := uniquename
	svc := s3.New(s)
	_, err = svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	} else {
		log.Println("S3bucket '", bucket, "' created successfully")
	}
	s3storage.S3BucketVersioning(bucket)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client := elasticsearch.ESclient()
	res, err := client.Index(
		"automate-orch-tenants",
		strings.NewReader(`{
				"tenantLabel":"`+t.TenantLabel+`",
				"tenantUniqueName":"`+uniquename+`",
				"state":"active",
				"account":"`+t.Account+`",
				"workers":"`+t.Workers+`",				
				"createdDate":"`+timestamp+`",				
				"timestamp":"`+timestamp+`",
				"roles": 
				{
					"tenant": {
					"owner":["rpa.dxc.ia@dxc.com","`+t.Owner+`"]
					},
					"environment":{
					"maker":["rpa.dxc.ia@dxc.com","`+t.Owner+`"]
					}
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
	count, _ := strconv.Atoi(t.Workers)
	CreateWorker(uniquename, count)
	cfg.Message = "Tenant '" + t.TenantLabel + "' created. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	defer res.Body.Close()
	resbody := ""
	header := res.Header["Location"][0]
	header = "Id:" + strings.TrimPrefix(header, "/automate-orch-tenants/_doc/")
	if res.StatusCode == 201 {
		resbody = "Tenant created sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: header, Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func GetTenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getTenant request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getTenant
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
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found for tenantId: " + t.TenantId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var tenant_list []listTenant
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		id := hit.(map[string]interface{})["_id"].(string)
		source := hit.(map[string]interface{})["_source"]
		tenantlist := listTenant{TenantId: id, Source: source}
		tenant_list = append(tenant_list, tenantlist)
	}
	var resbody []listTenant
	if res.StatusCode == 200 {
		resbody = tenant_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func UpdateTenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /UpdateTenant request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateTenant
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "updateTenant"
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	cfg.User = t.Owner
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.TenantId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found with tenantId: " + t.TenantId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var workers int
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		worker := hit.(map[string]interface{})["_source"].(map[string]interface{})["workers"].(string)
		workers, _ = strconv.Atoi(worker)
	}
	count, _ := strconv.Atoi(t.Workers)
	if workers > count {
		deletecount := workers - count
		DeleteWorker(t.TenantUniqueName, deletecount)
	} else if workers < count {
		createcount := count - workers
		CreateWorker(t.TenantUniqueName, createcount)
	}
	res1, err := client.Update(
		"automate-orch-tenants",
		t.TenantId,
		strings.NewReader(`{"doc": {				
				"tenantUniqueName":"`+t.TenantUniqueName+`",
				"state":"`+t.State+`",
				"account":"`+t.Account+`",				
				"workers":"`+t.Workers+`",								
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
	cfg.Message = "Tenant '" + t.TenantUniqueName + "' Updated. Response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "Tenant Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DeleteTenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /DeleteTenant request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteTenant
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "deleteTenant"
	cfg.User = t.DeletedBy
	cfg.Level = "info"
	cfg.TenantUniqueName = t.TenantUniqueName
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.TenantId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found for tenantId: " + t.TenantId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	exist := false
	var owner string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{})["tenant"].(map[string]interface{}) {
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
		resbody := "Only Tenant Owners can delete the tenant"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
	}
	env := environmentexists(t.TenantUniqueName)
	if env {
		resbody := "Active Environment found with '" + t.TenantUniqueName + "' try again later"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	worker := workerexists(t.TenantUniqueName)
	if worker {
		resbody := "Active workers found with '" + t.TenantUniqueName + "' try again later"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	bucket := t.TenantUniqueName
	count := s3storage.S3ObjectCount(bucket)
	if count == 0 {
		s := s3storage.S3Connect()
		svc := s3.New(s)
		_, err1 := svc.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: &bucket,
		})
		if err1 != nil {
			log.Println("Error Deleting the Bucket", err1)
			return
		}
		log.Println("S3 bucket", bucket, "has been deleted successfully")
	} else {
		log.Println(count, " objects exists in ", bucket, ", Cannot delete bucket")
		resbody := "One or more objects exists in " + bucket + ", Cannot delete bucket"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	res1, err := client.Delete(
		"automate-orch-tenants",
		t.TenantId,
		client.Delete.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res1.Body.Close()
	cfg.Message = "TenantId '" + t.TenantId + "' deleted. Response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "Tenant Deleted sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func workerexists(tenantUniqueName string) (exist bool) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	exists := false
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + tenantUniqueName + `" }},
	{ "term": { "state.keyword": "idle" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	m := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if m > 0 {
		exists = true
		log.Println("Active workers found with " + tenantUniqueName + " try again later")
		return
	}
	return exists
}

func environmentexists(tenantUniqueName string) (exist bool) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	exists := false
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + tenantUniqueName + `" }},
	{ "term": { "state.keyword": "active" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-environments"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	o := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if o > 0 {
		exists = true
		log.Println("Active Environment found with: " + tenantUniqueName + " try again later")
		return
	}
	return exists
}

func ListTenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listTenant request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getTenant
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var resp []ApiResponse
	var mapResp map[string]interface{}
	client := elasticsearch.ESclient()
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found in Tenant Index"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var tenant_list []listTenant
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		id := hit.(map[string]interface{})["_id"].(string)
		source := hit.(map[string]interface{})["_source"]
		tenantlist := listTenant{TenantId: id, Source: source}
		tenant_list = append(tenant_list, tenantlist)
	}
	var resbody []listTenant
	if res.StatusCode == 200 {
		resbody = tenant_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func TenantCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listTenant request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getTenant
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found in Tenant Index"
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
	var tenantcount []tenant
	tenantcount = append(tenantcount, tenant{TotalTenant: n, ActiveTenant: m})
	var resbody []tenant
	if res.StatusCode == 200 {
		resbody = tenantcount
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func AddTenantOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /AddTenantRole request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateTenantRole
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	own := strings.ReplaceAll(t.Owner, " ", "")
	if own == "" {
		resbody := "Tenant Owner details should not be empty"
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addTenantOnwer"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.TenantId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found with tenantId: " + t.TenantId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var owner_list, maker_list []string
	var owner, maker string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{})["tenant"].(map[string]interface{}) {
			if l == "owner" {
				owner_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					owner = hit1.([]interface{})[j].(string)
					if owner == t.Owner {
						resbody := t.Owner + " already exist in the Tenant Owner"
						response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
						resp = append(resp, response)
						json.NewEncoder(w).Encode(resp)
						return
					}
					owner_list = append(owner_list, "\""+owner+"\",")
				}
				owner_list = append(owner_list, "\""+t.Owner+"\"")
				owner = fmt.Sprint(owner_list)
			}
		}
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{})["environment"].(map[string]interface{}) {
			if l == "maker" {
				maker_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					maker = hit1.([]interface{})[j].(string)
					if maker == t.Owner {
						resbody := t.Owner + " already exist in the Environment Maker"
						response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
						resp = append(resp, response)
						json.NewEncoder(w).Encode(resp)
						return
					}
					maker_list = append(maker_list, "\""+maker+"\",")
				}
				maker_list = append(maker_list, "\""+t.Owner+"\"")
				maker = fmt.Sprint(maker_list)
			}
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-tenants",
		t.TenantId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
			{
				"tenant": {
					"owner":`+owner+`
				},
				"environment": {
					"maker":`+maker+`
				}
			}
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
	cfg.Message = "AddTenantowner response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "AddTenantowner Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func AddEnvironmentMaker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /AddTenantRole request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateTenantRole
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	mak := strings.ReplaceAll(t.Maker, " ", "")
	if mak == "" {
		resbody := "Environment Maker details should not be empty"
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "addEnvironmentMaker"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.TenantId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found for tenantId: '" + t.TenantId + "'"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var maker_list []string
	var maker string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{})["environment"].(map[string]interface{}) {
			if l == "maker" {
				maker_list = nil
				i := len(hit1.([]interface{}))
				for j := 0; j < i; j++ {
					maker = hit1.([]interface{})[j].(string)
					if maker == t.Maker {
						resbody := t.Maker + " already exist in the Environment Maker"
						response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
						resp = append(resp, response)
						json.NewEncoder(w).Encode(resp)
						return
					}
					maker_list = append(maker_list, "\""+maker+"\",")
				}
			}
			maker_list = append(maker_list, "\""+t.Maker+"\"")
			maker = fmt.Sprint(maker_list)
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-tenants",
		t.TenantId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
			{				
				"environment":{
				"maker":`+maker+`
				}
			}
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
	cfg.Message = "AddEnvironmentMaker response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "AddEnvironmentMaker Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func RemoveTenantOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /RemoveTenantRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateTenantRole
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	own := strings.ReplaceAll(t.Owner, " ", "")
	if own == "" {
		resbody := "Tenant Owner details should not be empty"
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeTenantOnwer"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.TenantId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found for tenantId: " + t.TenantId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var owner_list []string
	var owner string
	exist := 0
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{})["tenant"].(map[string]interface{}) {
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
		resbody := "Given email not exist in TenantOwner"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-tenants",
		t.TenantId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
			{
				"tenant": {
					"owner":`+owner+`
				}
			}
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
	cfg.Message = "RemoveTenantOwner response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "RemoveTenantOwner Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func RemoveEnvironmentMaker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /RemoveTenantRoles request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t updateTenantRole
	var resp []ApiResponse
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	mak := strings.ReplaceAll(t.Maker, " ", "")
	if mak == "" {
		resbody := "Environment Maker details should not be empty"
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "removeEnvironmentMaker"
	cfg.User = t.Owner
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.Level = "info"
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.TenantId + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-tenants"),
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
		resbody := "No records found with tenantId: " + t.TenantId
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var maker_list []string
	var maker string
	exist := 0

	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		for l, hit1 := range hit.(map[string]interface{})["_source"].(map[string]interface{})["roles"].(map[string]interface{})["environment"].(map[string]interface{}) {
			if l == "maker" {
				maker_list = nil
				i := len(hit1.([]interface{}))

				for j := 0; j < i; j++ {
					maker = hit1.([]interface{})[j].(string)
					if maker == t.Maker {
						exist = 1
					} else {
						if j == 0 {
							maker_list = append(maker_list, "\""+maker+"\"")
						} else if j == 1 && exist == 1 {
							maker_list = append(maker_list, "\""+maker+"\"")
						} else {
							maker_list = append(maker_list, ",\""+maker+"\"")
						}
					}
				}
			}
			maker = fmt.Sprint(maker_list)
		}
	}
	if exist == 0 {
		resbody := "Given email not exist in Environment Maker"
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	res1, err := client.Update(
		"automate-orch-tenants",
		t.TenantId,
		strings.NewReader(`{"doc":{
			"timestamp":"`+timestamp+`",
			"roles": 
			{
				"environment":{
				"maker":`+maker+`
				}
			}
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
	cfg.Message = "RemoveEnvironmentMaker response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "RemoveEnvironmentMaker Updated sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func CreateWorker(TenantUniqueName string, createcount int) {
	nfs_path1 := os.Getenv("NFS_PATH1")
	nfs_path2 := os.Getenv("NFS_PATH2")
	createWorkerUrl := os.Getenv("CREATE_WORKER")
	var nfs_path string
	var createWorkerCfg workerRequest
	for i := 0; i < createcount; i++ {
		if i%2 == 0 {
			nfs_path = nfs_path1
		} else {
			nfs_path = nfs_path2
		}
		createWorkerCfg.TenantUniqueName = TenantUniqueName
		createWorkerCfg.NfsStoragePath = nfs_path
		if createWorkerUrl == "" {
			log.Fatalln("CreateWorkerURL was not configured, exiting now.")
		}
		log.Println(createWorkerCfg)
		cwr, err := CreateWorkerRequest(createWorkerCfg, createWorkerUrl)
		if err != nil {
			log.Println("Error creating worker details", err)
			return
		}
		log.Println("StatusCode:", cwr.StatusCode, "Header:", cwr.Header, "Body:", cwr.Body)
	}
}

func CreateWorkerRequest(cwr workerRequest, worker_url string) (ApiResponse, error) {
	var cwrout ApiResponse
	cwr_json, _ := json.Marshal(cwr)
	cwr_json_byte := []byte(cwr_json)

	client := &http.Client{}
	req, err := http.NewRequest("POST", worker_url, bytes.NewBuffer(cwr_json_byte))
	if err != nil {
		log.Println("NewRequest: ", err)
		return ApiResponse{}, errors.New("HTTP Post failed")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	response, error := client.Do(req)
	if error != nil {
		log.Println("NewRequest: ", err)
		return ApiResponse{}, errors.New("HTTP Post failed")
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	json.Unmarshal(body, &cwrout)
	return cwrout, nil
}

func DeleteWorker(tenantUniqueName string, deletecount int) {
	deleteWorkerUrl := os.Getenv("DELETE_WORKER")
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var deleteWorkerCfg deleteworkerRequest
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + tenantUniqueName + `" }},
	{ "term": { "state.keyword": "idle" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println("Failed to check state WorkerId of: " + tenantUniqueName + fmt.Sprint(err))
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		log.Println(" WorkerId not found with Idle state, please try again later")
		return
	}
	if n > deletecount {
		i := 0
		for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
			i++
			if i > deletecount {
				log.Println(deletecount, " Workers deleted")
				return
			}
			WorkerId := hit.(map[string]interface{})["_id"].(string)

			deleteWorkerCfg.WorkerId = WorkerId
			if deleteWorkerUrl == "" {
				log.Println("DeleteWorkerURL was not configured, exiting now.")
				return
			}
			log.Println(deleteWorkerCfg)
			cwr, err := DeleteWorkerRequest(deleteWorkerCfg, deleteWorkerUrl)
			if err != nil {
				log.Println("Error creating worker details", err)
				return
			}
			log.Println("StatusCode:", cwr.StatusCode, "Header:", cwr.Header, "Body:", cwr.Body)

		}
	}
}

func DeleteWorkerRequest(dwr deleteworkerRequest, worker_url string) (ApiResponse, error) {
	var dwrout ApiResponse
	dwr_json, _ := json.Marshal(dwr)
	dwr_json_byte := []byte(dwr_json)

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", worker_url, bytes.NewBuffer(dwr_json_byte))
	if err != nil {
		log.Println("NewRequest: ", err)
		return ApiResponse{}, errors.New("HTTP Post failed")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	response, error := client.Do(req)
	if error != nil {
		log.Println("NewRequest: ", err)
		return ApiResponse{}, errors.New("HTTP Post failed")
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	json.Unmarshal(body, &dwrout)
	return dwrout, nil
}
