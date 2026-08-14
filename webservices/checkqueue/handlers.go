package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pythonrpa/auth"
	"pythonrpa/elasticsearch"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "checkqueue microservice!\n")
}

func CheckQueue(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body of /checkqueue request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t checkqueueRequest
	err = json.Unmarshal(body, &t)
	log.Println("request received:", t)

	if err != nil {
		log.Println("Wrong format of /checkqueue request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var query string

	workerMode := t.WorkerMode
	workerId := t.WorkerId

	rt := t.Rt
	ct := t.Ct

	if rt == "" || ct == "" {
		log.Fatalln("Authenication key is missing. Exiting now.")
	}
	//RSA Authentication
	decryptedtxt := auth.CheckAuth(workerId, ct)
	if rt != decryptedtxt {
		log.Fatalln("RSA Authentication Failed")
	}
	log.Println("RSA Authentication Successful")

	// Check for connection errors to the Elasticsearch cluster
	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}

	if workerMode == "cloud" {
		query = `{"size": 1, "query":{"bool":{"must":
		[
			{"range": {"plannedExecutionTime": {"lte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"range": {"expirationTime": {"gte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"term": { "status": "ready" }}
			]}},
		"sort" : [
    		{"priority":"asc"},
    		{"plannedExecutionTime":"asc"}
    		]
		}`
	} else if workerMode == "machine" {
		query = `{"size": 1, "query":{"bool":{"must":
		[
			{"range": {"plannedExecutionTime": {"lte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"range": {"expirationTime": {"gte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"term": { "status": "ready" }},
			{ "term": { "workerId": "` + workerId + `" }}
			]}},
		"sort" : [
    		{"priority":"asc"},
    		{"plannedExecutionTime":"asc"}
    		]
		}`
	}

	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-queue"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {
		executionId := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["executionId"])
		queueId := fmt.Sprint(docs[0]["_id"])
		tenantUniqueName := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["tenantUniqueName"])
		environmentUniqueName := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["environmentUniqueName"])

		res1, err := client.Update(
			"automate-orch-queue", queueId,
			strings.NewReader(`{"doc": {"status": "pending","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Fatalln("Error in updating ES document queue:", err)
		}

		res2, err := client.Update(
			"automate-orch-executions", executionId,
			strings.NewReader(`{"doc": {"status": "pending","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Fatalln("Error in updating ES document queue:", err)
		}

		results := checkqueueResult{QueueId: queueId, WorkerId: workerId, ExecutionId: executionId, Status: "pending", TenantUniqueName: tenantUniqueName, EnvironmentUniqueName: environmentUniqueName}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)

		res1.Body.Close()
		res2.Body.Close()

	} else {
		w.WriteHeader(http.StatusNoContent)
	}

}
