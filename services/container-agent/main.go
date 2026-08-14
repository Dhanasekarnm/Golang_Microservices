package main

import (
	"log"
	"net/http"
	"os"
	"pythonrpa/operations"
	"pythonrpa/orch"
	"time"
)

func main() {

	var getBotDetailsCfg getbotdetailsRequest
	var executionPath, zipfile string
	var extraParams map[string]string
	var gbdr getbotdetailsResult
	var request *http.Request
	var resp *http.Response
	var err error

	client := &http.Client{}

	getBotDetailsUrl := os.Getenv("getBotDetailsUrl")
	downloadBotUrl := os.Getenv("downloadBotUrl")
	sendResultUrl := os.Getenv("sendResultUrl")
	workerId := os.Getenv("workerId")

	getBotDetailsCfg.RandomText = os.Getenv("randomText")
	getBotDetailsCfg.ChipherText = os.Getenv("chipherText")
	getBotDetailsCfg.TenantUniqueName = os.Getenv("tenantUniqueName")
	getBotDetailsCfg.EnvironmentUniqueName = os.Getenv("environmentUniqueName")
	getBotDetailsCfg.ExecutionId = os.Getenv("executionId")

	if getBotDetailsUrl == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("getBotDetailsUrl was not configured, exiting now.")
	}

	if downloadBotUrl == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("downloadBotUrl was not configured, exiting now.")
	}

	if sendResultUrl == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("sendResultUrl was not configured, exiting now.")
	}

	if getBotDetailsCfg.TenantUniqueName == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Tenant Unique Name was not configured, exiting now.")
	}

	if getBotDetailsCfg.EnvironmentUniqueName == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Environment Unique Name was not configured, exiting now.")
	}

	if getBotDetailsCfg.ExecutionId == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Execution ID was not configured, exiting now.")
	}

	if getBotDetailsCfg.RandomText == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Random Text was not configured, exiting now.")
	}

	if getBotDetailsCfg.ChipherText == "" {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Chipher Text was not configured, exiting now.")
	}
	log.Println(getBotDetailsCfg)
	time.Sleep(1 * time.Second)
	gbdr, err = GetBotDetails(getBotDetailsCfg, getBotDetailsUrl)

	if err != nil {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Error getting bot details, Execution_ID: "+getBotDetailsCfg.ExecutionId, err)

	} else {
		if gbdr.ExecutionId != "" {
			log.Println("Succesfully received bot details for execution:" + getBotDetailsCfg.ExecutionId)
		} else {
			orch.ReleaseWorker(workerId)
			log.Println("There is nothing in the queue for Execution ID: " + getBotDetailsCfg.ExecutionId)
		}

	}
	log.Println("Execution ID:", gbdr.ExecutionId, "Status:", gbdr.Status)
	time.Sleep(1 * time.Second)
	if gbdr.ExecutionId != "" {
		out, err := DownloadBot(gbdr, getBotDetailsCfg, downloadBotUrl)
		if err != nil {
			orch.ReleaseWorker(workerId)
			log.Fatalln("Error downloading and unpacking bot: "+gbdr.BotUniqueName+", Execution ID: "+getBotDetailsCfg.ExecutionId, err)
		}
		executionPath = out
	} else {
		orch.ReleaseWorker(workerId)
		log.Println("Received empty execution ID, exiting.")
		return
	}

	log.Println("Execution Started.")
	log.Println("Execution Path:", executionPath)
	ExecuteBot(executionPath, getBotDetailsCfg.ExecutionId)
	log.Println("Bot execution finished.")

	extraParams = map[string]string{
		"executionId": gbdr.ExecutionId,
		"randomText":  getBotDetailsCfg.RandomText,
		"chipherText": getBotDetailsCfg.ChipherText,
	}
	zipfile = gbdr.BotUniqueName
	err = operations.Zip(executionPath, zipfile)
	if err != nil {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Error creating zip file ", err)
	} else {
		log.Println("Zip file created")
	}

	request, err = UploadResult(sendResultUrl, extraParams, "bot_result", zipfile)
	if err != nil {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Error uploading results", err)
	}

	resp, err = client.Do(request)
	if err != nil {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Upload of zip  file failed,error: ", err)
	} else {
		var bodyContent []byte
		log.Println("Zip file upload finished with HTTP code ", resp.StatusCode)
		resp.Body.Read(bodyContent)
		resp.Body.Close()
	}
	orch.ReleaseWorker(workerId)
}
