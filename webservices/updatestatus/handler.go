package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pythonrpa/elasticsearch"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "updatestatus microservice!\n")
}

func UpdateStatus(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var t updateStatus

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body of /updatestatus request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &t)
	log.Println("request received:", t)

	if err != nil {
		log.Println("Wrong format of /updatestatus request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	client, err := elasticsearch.ESConnect()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Fatalln("Elasticsearch connection error:", err)
	}
	if t.Result == "Started" {
		rBody := resultStartBody{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Result:    t.Result,
			StartTime: time.Now().UTC().Format(time.RFC3339),
		}

		rdata, _ := json.Marshal(rBody)
		res, err := client.Update(
			"automate-orch-executions", t.ExecutionId,
			bytes.NewReader(rdata),
			client.Update.WithPretty(),
		)

		defer res.Body.Close()
		if err != nil {
			log.Println("Error in updating result:", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		rBody := resultEndBody{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Result:    t.Result,
			EndTime:   time.Now().UTC().Format(time.RFC3339),
		}

		rdata, _ := json.Marshal(rBody)
		res, err := client.Update(
			"automate-orch-executions", t.ExecutionId,
			bytes.NewReader(rdata),
			client.Update.WithPretty(),
		)

		defer res.Body.Close()
		if err != nil {
			log.Println("Error in updating result:", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

}
