package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"pythonrpa/elasticsearch"
	"pythonrpa/orch"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "triggerbot microservice!\n")
}

func TriggerBot(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var query, s3InventoryPath, md5Checksum, botUniqueName, botLabel, botType string
	var t triggerBot

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body of /triggerbot request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &t)
	log.Println("request received:", t)

	if err != nil {
		log.Println("Wrong format of /triggerbot request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	client, err := elasticsearch.ESConnect()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Fatalln("Elasticsearch connection error:", err)
	}

	query = `{
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "_id": "` + t.BotId + `" }}
					]}}
		  }`
	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithSeqNoPrimaryTerm(true),
	)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) == 1 {
		s3InventoryPath = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["s3InventoryPath"])
		md5Checksum = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["md5Checksum"])
		botUniqueName = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botUniqueName"])
		botLabel = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botLabel"])
		botType = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botType"])
	} else if len(docs) > 1 {
		log.Println("Mutiple bots available with same ID, ID:", t.BotId)
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode("Mutiple bots available with same ID, ID:" + t.BotId)
		return
	} else if len(docs) == 0 {
		log.Println("No bot available with bot ID:", t.BotId)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode("No bot available with bot ID:" + t.BotId)
		return
	}

	s3ExecutionPath := t.TenantUniqueName + "/" + t.EnvironmentUniqueName + "/bots/" + botType + "/executions/" + botUniqueName + ".zip"

	switch t.BotMode {
	case "cloud":
	findIdleWorker:
		query = `{
			"size": 1,
			"query":{
			  "bool":{
				"must":[
				  {"term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
				  {"term": { "state.keyword": "idle" }},
				  {"term": { "workerMode.keyword": "` + t.BotMode + `" }}
				]
			  }
			}
		  }`

		searchresp, err := client.Search(
			client.Search.WithIndex("automate-orch-workers"),
			client.Search.WithBody(strings.NewReader(query)),
			client.Search.WithTrackTotalHits(true),
			client.Search.WithSeqNoPrimaryTerm(true),
		)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			log.Println("Search failed, Check search query", err)
		}
		docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

		if len(docs) > 0 {
			workerId := fmt.Sprint(docs[0]["_id"])
			seqNumber, _ := strconv.Atoi(fmt.Sprint(docs[0]["_seq_no"]))
			primaryTerm, _ := strconv.Atoi(fmt.Sprint(docs[0]["_primary_term"]))

			res1, err := client.Update(
				"automate-orch-workers", workerId,
				strings.NewReader(`{"doc": {"state": "allocated","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
				client.Update.WithPretty(),
				client.Update.WithIfSeqNo(seqNumber),
				client.Update.WithIfPrimaryTerm(primaryTerm),
			)
			if err != nil {
				log.Println("Error in updating worker:", err)
				w.WriteHeader(http.StatusNotFound)
			}
			if res1.StatusCode == 409 {
				log.Println("Statuscode:", res1.StatusCode)
				res1.Body.Close()
				goto findIdleWorker
			}

			if res1.StatusCode != 409 && res1.StatusCode != 200 {
				log.Println("Statuscode:", res1.StatusCode)
				log.Println("Worker not allocated")
				res1.Body.Close()
			}

			log.Println("Allocated worker with ID:", workerId)
			res1.Body.Close()

			quuid, err := exec.Command("uuidgen").Output()

			if err != nil {
				log.Println("Error in generating queue UUID", err)
				orch.ReleaseWorker(workerId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			euuid, err := exec.Command("uuidgen").Output()

			if err != nil {
				log.Println("Error in generating execution UUID", err)
				orch.ReleaseWorker(workerId)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			//plannedExecutionTime := time.Now().UTC().Format(time.RFC3339)
			plannedExecutionTime, err := time.Parse(time.RFC3339, t.PlannedExecutionTime)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			expirationTime := plannedExecutionTime.Add(24 * time.Hour).Format(time.RFC3339)
			queueId := strings.TrimSuffix(string(quuid), "\n")
			executionId := strings.TrimSuffix(string(euuid), "\n")

			ebody := executionBody{
				TenantUniqueName:      t.TenantUniqueName,
				EnvironmentUniqueName: t.EnvironmentUniqueName,
				QueueId:               queueId,
				WorkerId:              workerId,
				Timestamp:             time.Now().UTC().Format(time.RFC3339),
				PlannedExecutionTime:  t.PlannedExecutionTime,
				Status:                "ready",
				BotLabel:              botLabel,
				BotUniqueName:         botUniqueName,
				BotType:               botType,
				BotId:                 t.BotId,
				S3InventoryPath:       s3InventoryPath,
				S3ExecutionPath:       s3ExecutionPath,
				Md5Checksum:           md5Checksum,
				Priority:              t.Priority,
			}

			edata, _ := json.Marshal(ebody)

			res2, err := client.Index("automate-orch-executions", bytes.NewReader(edata), client.Index.WithDocumentID(executionId), client.Index.WithPretty())
			defer res2.Body.Close()
			if err != nil {
				log.Println("Error in updating execution:", err)
				orch.ReleaseWorker(workerId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			qbody := queueBody{
				TenantUniqueName:      t.TenantUniqueName,
				EnvironmentUniqueName: t.EnvironmentUniqueName,
				WorkerId:              workerId,
				Timestamp:             time.Now().UTC().Format(time.RFC3339),
				PlannedExecutionTime:  t.PlannedExecutionTime,
				Status:                "ready",
				Priority:              t.Priority,
				ExpirationTime:        expirationTime,
				ExecutionId:           executionId,
				BotId:                 t.BotId,
				BotMode:               t.BotMode,
			}

			qdata, _ := json.Marshal(qbody)

			res3, err := client.Index("automate-orch-queue", bytes.NewReader(qdata), client.Index.WithDocumentID(queueId), client.Index.WithPretty())
			defer res3.Body.Close()
			if err != nil {
				log.Println("Error in updating queue:", err)
				orch.ReleaseWorker(workerId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			tbody := triggerBody{
				Timestamp:            time.Now().UTC().Format(time.RFC3339),
				TriggeredBy:          t.TriggeredBy,
				TriggerMode:          t.TriggerMode,
				QueueId:              queueId,
				ExecutionId:          executionId,
				PlannedExecutionTime: t.PlannedExecutionTime,
				BotId:                t.BotId,
			}

			tdata, _ := json.Marshal(tbody)

			res4, err := client.Index("automate-orch-triggers", bytes.NewReader(tdata), client.Index.WithPretty())
			defer res4.Body.Close()
			if err != nil {
				log.Println("Error in creating triggers:", err)
				orch.ReleaseWorker(workerId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			log.Println(botLabel, "has been triggered")
			w.WriteHeader(http.StatusOK)
		} else {
			log.Println("No Idle worker is available for tenant", t.TenantUniqueName)
			w.WriteHeader(http.StatusPreconditionFailed)
			json.NewEncoder(w).Encode("No Idle worker is available for tenant" + t.TenantUniqueName)
		}
	}
}
