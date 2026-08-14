package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"pythonrpa/orch"
	"pythonrpa/s3storage"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "uploadbotresult microservice!\n")
}

func UploadBotResult(w http.ResponseWriter, r *http.Request) {

	s := s3storage.S3Connect()

	executionids, ok := r.URL.Query()["executionId"]
	if !ok || len(executionids[0]) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	executionId := executionids[0]

	randomTexts, ok := r.URL.Query()["randomText"]
	if !ok || len(randomTexts[0]) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	randomText := randomTexts[0]

	chipherTexts, ok := r.URL.Query()["chipherText"]
	if !ok || len(chipherTexts[0]) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	chipherText := chipherTexts[0]

	// Check for connection errors to the Elasticsearch cluster
	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}

	log.Println(executionId, randomText, chipherText)

	query := `{
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "status": "executing" }},
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
		bucket := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["tenantUniqueName"])
		s3ExecutionPath := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["s3ExecutionPath"])
		workerId := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["workerId"])

		err = orch.ValidateRSAAuth(workerId, randomText, chipherText)
		if err != nil {
			log.Println("RSA Authentication Failed", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {

			var max_upload_file_size int64
			max_size_string := os.Getenv("MAX_UPLOAD_SIZE_MB")
			max_upload_file_size, err = strconv.ParseInt(max_size_string, 10, 64)
			if err != nil {
				max_upload_file_size = 50
			}
			log.Println(max_upload_file_size, max_upload_file_size<<20)
			r.Body = http.MaxBytesReader(w, r.Body, max_upload_file_size<<20)
			err2 := r.ParseMultipartForm(1000)
			if err2 != nil {
				log.Println(err2)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			mForm := r.MultipartForm

			for k := range mForm.File {
				if k == "bot_result" {

					file, fileHeader, err := r.FormFile(k)
					if err != nil {
						log.Println(err)
						w.WriteHeader(http.StatusNoContent)
						return
					}
					defer file.Close()

					bots_base_path := os.Getenv("BOTSHOMEDIR")
					resultFile := bots_base_path + "/" + s3ExecutionPath
					resultDir, _ := filepath.Split(resultFile)
					if !operations.PathExists(resultDir) {
						os.MkdirAll(resultDir, 0755)
					}
					out, err := os.Create(resultFile)
					if err != nil {
						log.Printf("failed to open the file %s for writing", resultFile)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					defer out.Close()
					_, err = io.Copy(out, file)
					if err != nil {
						log.Printf("copy file err:%s\n", err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					versionId, _, err := s3storage.S3UploadFile(s, resultFile, bucket, s3ExecutionPath)
					if err != nil {
						log.Println("Failed to upload file to S3: ", fileHeader.Filename, err)
					}

					res1, err := client.Update(
						"automate-orch-queue", queueId,
						strings.NewReader(`{"doc": {"status": "completed","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
						client.Update.WithPretty(),
					)
					if err != nil {
						log.Fatalln("Error in updating ES document queue:", err)
					}

					res2, err := client.Update(
						"automate-orch-executions", executionId,
						strings.NewReader(`{"doc": {"status": "completed", "s3ExecutionVersionId" : "`+versionId+`", "endTime":"`+time.Now().UTC().Format(time.RFC3339)+`","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
						client.Update.WithPretty(),
					)
					if err != nil {
						log.Fatalln("Error in updating ES document queue:", err)
					}

					res1.Body.Close()
					res2.Body.Close()
				}
			}
		}
	}
}
