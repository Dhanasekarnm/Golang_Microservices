package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"pythonrpa/operations"
	"pythonrpa/orch"

	"gopkg.in/yaml.v2"
)

func main() {

	var checkQueueRequest checkqueueRequest
	var cqr checkqueueResult

	var randomText, chipherText, workerId, workerMode string

	var getBotDetailsCfg getbotdetailsRequest
	var executionPath, zipfile string
	var extraParams map[string]string
	var gbdr getbotdetailsResult
	var request *http.Request
	var resp *http.Response

	client := &http.Client{}

	for {

		// Parsing settings.yaml
		type Settings struct {
			Endpoints struct {
				CheckQueue         string `yaml:"checkQueue"`
				GetBotDetails      string `yaml:"getBotDetails"`
				DownloadBot        string `yaml:"downloadBot"`
				SendResult         string `yaml:"sendResult"`
				UpdateStatusResult string `yaml:"updateStatusResult"`
			} `yaml:"endpoints"`
		}

		yamlFile, err := os.ReadFile("settings.yaml")
		if err != nil {
			fmt.Println("Error reading YAML file:", err)
			return
		}

		var settings Settings
		err = yaml.Unmarshal(yamlFile, &settings)
		if err != nil {
			fmt.Println("Error parsing YAML:", err)
			return
		}

		checkQueueUrl := settings.Endpoints.CheckQueue
		getBotDetailsUrl := settings.Endpoints.GetBotDetails
		downloadBotUrl := settings.Endpoints.DownloadBot
		sendResultUrl := settings.Endpoints.SendResult
		updateStatusResultUrl := settings.Endpoints.UpdateStatusResult

		//Extract worker ID and cipher text from encrypted License
		workerId, chipherText, _ = orch.DecryptLicense("license")

		randomText = workerId
		workerMode = "machine"

		checkQueueRequest.Ct = chipherText
		checkQueueRequest.Rt = randomText
		checkQueueRequest.WorkerId = workerId
		checkQueueRequest.WorkerMode = workerMode

		cqr, err = CheckQueue(checkQueueRequest, checkQueueUrl)

		if cqr.QueueId != "" && err == nil {
			getBotDetailsCfg.RandomText = randomText
			getBotDetailsCfg.ChipherText = chipherText
			getBotDetailsCfg.TenantUniqueName = cqr.TenantUniqueName
			getBotDetailsCfg.EnvironmentUniqueName = cqr.EnvironmentUniqueName
			getBotDetailsCfg.ExecutionId = cqr.ExecutionId
			gbdr, err = GetBotDetails(getBotDetailsCfg, getBotDetailsUrl)

			if err != nil {
				//orch.ReleaseWorker(workerId)
				log.Fatalln("Error getting bot details, Execution_ID: "+getBotDetailsCfg.ExecutionId, err)

			} else {
				if gbdr.ExecutionId != "" {
					log.Println("Succesfully received bot details for execution:" + getBotDetailsCfg.ExecutionId)
				} else {
					//orch.ReleaseWorker(workerId)
					log.Println("There is nothing in the queue for Execution ID: " + getBotDetailsCfg.ExecutionId)
				}

			}
			log.Println("Execution ID:", gbdr.ExecutionId, "Status:", gbdr.Status)
			//time.Sleep(1 * time.Second)
			if gbdr.ExecutionId != "" {
				out, err := DownloadBot(gbdr, getBotDetailsCfg, downloadBotUrl)
				if err != nil {
					//orch.ReleaseWorker(workerId)
					log.Fatalln("Error downloading and unpacking bot: "+gbdr.BotUniqueName+", Execution ID: "+getBotDetailsCfg.ExecutionId, err)
				}
				executionPath = out
			} else {
				//orch.ReleaseWorker(workerId)
				log.Println("Received empty execution ID, exiting.")
				return
			}

			log.Println("Execution Started.")
			log.Println("Execution Path:", executionPath)
			ExecuteBot(executionPath, getBotDetailsCfg.ExecutionId, updateStatusResultUrl)
			log.Println("Bot execution finished.")

			extraParams = map[string]string{
				"executionId": gbdr.ExecutionId,
				"randomText":  getBotDetailsCfg.RandomText,
				"chipherText": getBotDetailsCfg.ChipherText,
			}
			zipfile = gbdr.BotUniqueName
			err = operations.Zip(executionPath, zipfile)
			if err != nil {
				//orch.ReleaseWorker(workerId)
				log.Fatalln("Error creating zip file ", err)
			} else {
				log.Println("Zip file created")
			}

			request, err = UploadResult(sendResultUrl, extraParams, "bot_result", zipfile)
			if err != nil {
				//orch.ReleaseWorker(workerId)
				log.Fatalln("Error uploading results", err)
			}

			resp, err = client.Do(request)
			if err != nil {
				//orch.ReleaseWorker(workerId)
				log.Fatalln("Upload of zip  file failed,error: ", err)
			} else {
				var bodyContent []byte
				log.Println("Zip file upload finished with HTTP code ", resp.StatusCode)
				resp.Body.Read(bodyContent)
				resp.Body.Close()
			}
		}

	}
}
