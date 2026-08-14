package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pythonrpa/elasticsearch"
	"pythonrpa/orch"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "getbotdetails microservice!\n")
}

func GetBotDetails(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {

		log.Println("Error reading body of request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getbotdetailsRequest
	err = json.Unmarshal(body, &t)
	log.Println("request received:", t)

	if err != nil {
		log.Println("Wrong format of /Getbotdetails request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	executionId := t.ExecutionId

	// Check for connection errors to the Elasticsearch cluster
	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}

	query := `{
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "status": "pending" }},
					{"term": { "_id": "` + executionId + `" }}
					]}}
		  }`

	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-executions"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {
		queueId := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["queueId"])
		s3InventoryPath := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["s3InventoryPath"])
		md5Checksum := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["md5Checksum"])
		executionId := fmt.Sprint(docs[0]["_id"])
		botUniqueName := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botUniqueName"])
		workerId := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["workerId"])

		randomText := t.RandomText
		chipherText := t.ChipherText

		err = orch.ValidateRSAAuth(workerId, randomText, chipherText)
		if err != nil {
			log.Println("RSA Authentication Failed", err)
			return
		} else {
			res1, err := client.Update(
				"automate-orch-queue", queueId,
				strings.NewReader(`{"doc": {"status": "executing","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
				client.Update.WithPretty(),
			)
			if err != nil {
				log.Fatalln("Error in updating ES document queue:", err)
			}

			res2, err := client.Update(
				"automate-orch-executions", executionId,
				strings.NewReader(`{"doc": {"status": "executing","startTime":"`+time.Now().UTC().Format(time.RFC3339)+`","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
				client.Update.WithPretty(),
			)
			if err != nil {
				log.Fatalln("Error in updating ES document queue:", err)
			}

			results := getbotdetailsResult{BotUniqueName: botUniqueName, QueueId: queueId, ExecutionId: executionId, Status: "executing", TenantUniqueName: t.TenantUniqueName, EnvironmentUniqueName: t.EnvironmentUniqueName, S3InventoryPath: s3InventoryPath, Md5Checksum: md5Checksum}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(results)

			res1.Body.Close()
			res2.Body.Close()

		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}

}
