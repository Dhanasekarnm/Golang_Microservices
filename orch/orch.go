package orch

import (
	"errors"
	"fmt"
	"log"
	"os"
	"pythonrpa/auth"
	"pythonrpa/elasticsearch"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
)

func ReleaseWorker(workerId string) {
	esclient, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}
	res, err := esclient.Update(
		"automate-orch-workers", workerId,
		strings.NewReader(`{"doc": {"state": "idle","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
		esclient.Update.WithPretty(),
	)
	if err != nil {
		log.Println("Unable to release worker with ID:", workerId, ",error:", err)
		return
	}
	res.Body.Close()
	log.Println("Released worker with ID:", workerId)
}

func RetrieveWorkerEncryption(workerId string) (string, string, error) {

	var randomText, chipherText string
	esclient, err := elasticsearch.ESConnect()
	if err != nil {
		log.Println("Elasticsearch connection error:", err)
		return "", "", err
	}
	query := `{
		"query":{
		  "bool":{
			"must":[
			  {"term": { "_id": "` + workerId + `" }}
			]
		  }
		}
	  }`

	searchresp, err := esclient.Search(
		esclient.Search.WithIndex("automate-orch-workers"),
		esclient.Search.WithBody(strings.NewReader(query)),
		esclient.Search.WithTrackTotalHits(true),
		esclient.Search.WithSeqNoPrimaryTerm(true),
	)
	if err != nil {
		log.Println("Search failed, Check search query", err)
		return "", "", err
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {
		randomText = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["randomText"])
		chipherText = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["chipherText"])
	}
	if randomText == "" && chipherText == "" {
		log.Println("Both randomText and chipherText are blanks")
		return randomText, chipherText, errors.New("both randomtext and chiphertext are blanks")
	}
	return randomText, chipherText, nil
}

func ValidateRSAAuth(workerId string, randomText string, chipherText string) error {
	if randomText == "" || chipherText == "" {
		log.Println("Authenication key is missing. Exiting now.")
		return errors.New("authenication key is missing")
	}
	//RSA Authentication
	decryptedtxt := auth.CheckAuth(workerId, chipherText)
	if randomText != decryptedtxt {
		log.Println("RSA Authentication Failed")
		return errors.New("rsa authentication failed")
	} else {
		log.Println("RSA Authentication Successful")
		return nil
	}
}

func encryptDecrypt(input, key string) string {
	output := make([]byte, len(input))
	for i := range input {
		output[i] = input[i] ^ key[i%len(key)]
	}
	return string(output)
}

func GenerateLicense(LicenseFile string, EncryptedLicenseFile string) error {
	data, err := os.ReadFile(LicenseFile)
	if err != nil {
		return err
	} else {
		hostname, _ := os.Hostname()
		privateKey := strings.ToLower(hostname)
		encryptedText := encryptDecrypt(string(data), privateKey)
		encryptedLicenseFile, err := os.Create(EncryptedLicenseFile)
		if err != nil {
			return err
		}
		defer encryptedLicenseFile.Close()
		_, err = encryptedLicenseFile.WriteString(encryptedText)
		if err != nil {
			return err
		}
		return nil

	}
}

func DecryptLicense(EncryptedLicenseFile string) (string, string, error) {
	data, err := os.ReadFile(EncryptedLicenseFile)
	if err != nil {
		return "", "", err
	} else {
		privateKey, _ := os.Hostname()
		decryptedText := strings.Split(encryptDecrypt(string(data), privateKey), ",")
		return decryptedText[0], decryptedText[1], nil

	}
}
