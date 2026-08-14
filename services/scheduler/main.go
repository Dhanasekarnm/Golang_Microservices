package main

import (
	"fmt"
	"log"
	"os"
	"pythonrpa/elasticsearch"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
)

func main() {

	var botId, scheduleId, tenantUniqueName, environmentUniqueName, frequency string
	var interval int
	var days []interface{}
	var nextExecutionTime time.Time

	tenantUniqueName = os.Getenv("TENANT_UNAME")

	for {

		esclient, err := elasticsearch.ESConnect()
		if err != nil {
			log.Println("Elasticsearch connection error:", err)
			return
		}

		query := `{"size": 1,"query":{"bool":{"must":
		[
			{"range": {"plannedExecutionTime": {"lte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"range": {"endTime": {"gte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"term": { "tenantUniqueName.keyword": "` + tenantUniqueName + `" }},
			{"term": { "state": "active" }}
			]}},
		"sort" : [
    		{"priority":"asc"},
    		{"plannedExecutionTime":"asc"}
    		]
		}`

		searchresp, err := esclient.Search(
			esclient.Search.WithIndex("automate-orch-schedules"),
			esclient.Search.WithBody(strings.NewReader(query)),
			esclient.Search.WithTrackTotalHits(true),
			esclient.Search.WithSeqNoPrimaryTerm(true),
		)
		if err != nil {
			log.Println("Search failed, Check search query", err)
			return
		}
		docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

		if len(docs) > 0 {
			scheduleId = fmt.Sprint(docs[0]["_id"])
			botId = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botId"])
			botMode := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botMode"])
			seqNumber, _ := strconv.Atoi(fmt.Sprint(docs[0]["_seq_no"]))
			primaryTerm, _ := strconv.Atoi(fmt.Sprint(docs[0]["_primary_term"]))
			priority, _ := strconv.Atoi(fmt.Sprint(docs[0]["_source"].(map[string]interface{})["priority"]))
			environmentUniqueName = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["environmentUniqueName"])
			plannedExecutionTime, err := time.Parse(time.RFC3339, fmt.Sprint(docs[0]["_source"].(map[string]interface{})["plannedExecutionTime"]))
			if err != nil {
				log.Println(err)
			}
			interval, _ = strconv.Atoi(fmt.Sprint(docs[0]["_source"].(map[string]interface{})["interval"]))
			frequency = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["frequency"])
			days = docs[0]["_source"].(map[string]interface{})["days"].([]interface{})

			nextExecutionTime = ComputeNextCycle(plannedExecutionTime, interval, frequency, days)
			log.Println("scheduleID:", scheduleId, "priority:", priority, "botId:", botId, "tenantUniqueName:", tenantUniqueName, "environmentUniqueName:", environmentUniqueName, "plannedExecutionTime:", plannedExecutionTime, "interval:", interval, "frequency:", frequency, "days:", days, "nextExecutionTime:", nextExecutionTime)

			res1, err := esclient.Update(
				"automate-orch-schedules", scheduleId,
				strings.NewReader(`{"doc": {"plannedExecutionTime": "`+nextExecutionTime.Format(time.RFC3339)+`" ,"timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
				esclient.Update.WithPretty(),
				esclient.Update.WithIfSeqNo(seqNumber),
				esclient.Update.WithIfPrimaryTerm(primaryTerm),
			)
			defer res1.Body.Close()
			if err != nil {
				log.Println("Error in updating schedules:", err)
			}
			if res1.StatusCode != 200 {
				log.Println("Statuscode:", res1.StatusCode, "for bot ID:", botId)
			} else if res1.StatusCode == 200 {
				triggeredBy := scheduleId
				triggerMode := "Schedule"
				responseStatusCode, err := TriggerBot(tenantUniqueName, environmentUniqueName, botMode, botId, priority, triggeredBy, triggerMode, plannedExecutionTime.Format(time.RFC3339))
				if err != nil {
					log.Println("Error in triggering bot:", err)
				}
				if responseStatusCode == 200 {
					log.Println("Bot with", botId, "triggered by scheduler with ID", scheduleId)
				}
			}
		}
	}
}
