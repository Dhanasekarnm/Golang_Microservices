package elasticsearch

import (
	"encoding/json"
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
)

const Red = "\033[31m"

func ESConnect() (*elasticsearch.Client, error) {
	esNode1 := os.Getenv("ELASTIC_MASTER1")
	esNode2 := os.Getenv("ELASTIC_MASTER2")
	esNode3 := os.Getenv("ELASTIC_MASTER3")
	esUser := os.Getenv("ELASTIC_USERNAME")
	esPassword := os.Getenv("ELASTIC_PASSWORD")

	cfg := elasticsearch.Config{
		Addresses: []string{
			esNode1,
			esNode2,
			esNode3,
		},
		Username: esUser,
		Password: esPassword,
	}
	es, err := elasticsearch.NewClient(cfg)
	return es, err
}

func ESparsehits(searchresult *esapi.Response) []map[string]interface{} {
	var mapResp map[string]interface{}
	docs := []map[string]interface{}{}

	if err := json.NewDecoder(searchresult.Body).Decode(&mapResp); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n != 0 {
		for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
			doc := hit.(map[string]interface{})
			docs = append(docs, doc)
		}
	}
	return docs
}

func ESclient() *elasticsearch.Client {
	client, err := ESConnect()
	if err != nil {
		log.Println(Red+"Elasticsearch API connection error: ", err)
	}
	return client
}
