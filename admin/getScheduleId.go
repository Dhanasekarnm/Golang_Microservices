package main

import (
	"fmt"
	"log"
	"pythonrpa/elasticsearch"
	"strings"

	"github.com/elastic/go-elasticsearch/esapi"
)

func GetScheduleId() {

	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}

	query := `{
		"size": 100,
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "state": "active" }},
					{"term": { "tenantUniqueName": "dxc" }},
					{"term": { "environmentUniqueName": "dxc" }}
					]}}
		  }`

	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {
		for k := range docs {
			botId := fmt.Sprint(docs[k]["_source"].(map[string]interface{})["botId"])
			scheduleId := fmt.Sprint(docs[k]["_id"])
			fmt.Println(k+1, ".", "\t", "Bot ID:", botId, "\t", "Schedule ID:", scheduleId)

		}

	} else {
		log.Println("No active bots")
	}
}
