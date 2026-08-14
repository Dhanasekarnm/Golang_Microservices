package main

import (
	"fmt"
	"log"
	"pythonrpa/elasticsearch"
	"strings"

	"github.com/elastic/go-elasticsearch/esapi"
)

func GetBotId() {

	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}

	query := `{
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "state": "active" }},
					{"wildcard": {"botUniqueName": "1hr*"}}
					]}}
		  }`

	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {
		for k := range docs {
			botUniqueName := fmt.Sprint(docs[k]["_source"].(map[string]interface{})["botUniqueName"])
			botId := fmt.Sprint(docs[k]["_id"])
			fmt.Println(k+1, ".", "\t", "Bot unique name:", botUniqueName, "\t", "Bot ID:", botId)

		}

	} else {
		log.Println("No active bots")
	}
}
