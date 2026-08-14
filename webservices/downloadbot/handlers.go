package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"pythonrpa/orch"
	"pythonrpa/s3storage"

	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/elastic/go-elasticsearch/esapi"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "downloadbot microservice!\n")
}

func DownloadBot(w http.ResponseWriter, r *http.Request) {

	var bot_file *os.File
	s := s3storage.S3Connect()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body of request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t downloadbotRequest
	err = json.Unmarshal(body, &t)
	log.Println("request received:", t)

	if err != nil {
		log.Println("Wrong format of /Downloadbot request")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	executionId := t.ExecutionId

	// Check for connection errors to the Elasticsearch cluster
	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Println("Elasticsearch connection error:", err)
		return
	}

	query := `{
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "status": "executing" }},
					{"term": { "_id": "` + executionId + `" }}
					]}}
		  }`

	log.Println("Query:", query)

	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-executions"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println("Search failed, Check search query", err)
		return
	}

	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))
	log.Println("Doc size", len(docs))

	if len(docs) > 0 {

		log.Println("In Download process")
		download_flag := true

		s3InventoryPath := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["s3InventoryPath"])
		md5Checksum := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["md5Checksum"])
		workerId := fmt.Sprint(docs[0]["_source"].(map[string]interface{})["workerId"])

		randomText := t.RandomText
		chipherText := t.ChipherText

		err = orch.ValidateRSAAuth(workerId, randomText, chipherText)
		if err != nil {
			log.Println("RSA Authentication Failed", err)
			return
		} else {
			dir, filename := filepath.Split(s3InventoryPath)
			bots_base_path := os.Getenv("BOTSHOMEDIR")
			bots_path := bots_base_path + "/" + dir
			bots_full_path := bots_path + filename

			log.Println(bots_full_path)

			if !operations.PathExists(bots_path) {
				os.MkdirAll(bots_path, 0755)
			}

			if operations.PathExists(bots_full_path) {
				bot_file, err = os.Open(bots_full_path)
				if err != nil {
					log.Println("Failed to open folder", err)
					return
				}
				hash := md5.New()
				_, err = io.Copy(hash, bot_file)
				if err != nil {
					log.Println("Failed to copy hash file", err)
					return
				}
				log.Println("md5_checksum:", md5Checksum)
				log.Println("hash:", string(hex.EncodeToString(hash.Sum(nil))))

				if md5Checksum == string(hex.EncodeToString(hash.Sum(nil))) {
					download_flag = false
				}
			}

			if download_flag {
				log.Println("Download Flag:", download_flag)
				bucket := aws.String(t.TenantUniqueName)
				key := aws.String(s3InventoryPath)

				bot_file, err = os.Create(bots_full_path)
				if err != nil {
					log.Println("Failed to create file", err)
					return
				}

				err = s3storage.S3DownloadCurrentVersion(s, bot_file, bucket, key)
				if err != nil {
					log.Println("Failed to download file", err)
					return
				}
				log.Println("File Downloaded successfully")
			}

			//now serve file
			w.WriteHeader(http.StatusOK)
			data, _ := os.ReadFile(bots_full_path)

			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", "attachment; filename='"+filename+"'")
			w.Write(data)
		}
	}
	bot_file.Close()
}
