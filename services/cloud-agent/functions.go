package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"pythonrpa/elasticsearch"
	"pythonrpa/kube"
	"pythonrpa/orch"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
	"golang.org/x/sys/unix"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CheckQueue(tenantUniqueName string) (string, string, string, string, string, string, string, error) {

	log.Println("Checkqueue started")

	var queueId, executionId, workerId, environmentUniqueName, botId, randomText, chipherText string
	var seqNumber int
	var primaryTerm int

	esclient, err := elasticsearch.ESConnect()
	if err != nil {
		log.Println("Elasticsearch connection error:", err)
		return "", "", "", "", "", "", "", err
	}

	query := `{"size": 1, "query":{"bool":{"must":
		[
			{"range": {"plannedExecutionTime": {"lte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"range": {"expirationTime": {"gte": "` + time.Now().UTC().Format(time.RFC3339) + `"}}},
			{"term": { "tenantUniqueName.keyword": "` + tenantUniqueName + `" }},
			{"term": { "botMode.keyword": "cloud" }},
			{"term": { "status": "ready" }}
			]}},
		"sort" : [
    		{"priority":"asc"},
    		{"plannedExecutionTime":"asc"}
    		]
		}`

	searchresp, err := esclient.Search(
		esclient.Search.WithIndex("automate-orch-queue"),
		esclient.Search.WithBody(strings.NewReader(query)),
		esclient.Search.WithTrackTotalHits(true),
		esclient.Search.WithSeqNoPrimaryTerm(true),
	)
	if err != nil {
		log.Println("Search failed, Check search query", err)
		return "", "", "", "", "", "", "", err
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {

		executionId = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["executionId"])
		queueId = fmt.Sprint(docs[0]["_id"])
		seqNumber, _ = strconv.Atoi(fmt.Sprint(docs[0]["_seq_no"]))
		primaryTerm, _ = strconv.Atoi(fmt.Sprint(docs[0]["_primary_term"]))
		botId = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["botId"])
		workerId = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["workerId"])
		environmentUniqueName = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["environmentUniqueName"])

		randomText, chipherText, err = orch.RetrieveWorkerEncryption(workerId)
		if err != nil {
			log.Println("Error in retrieving worker encryption text", err)
			return "", "", "", "", "", "", "", err
		}

		err = orch.ValidateRSAAuth(workerId, randomText, chipherText)
		if err != nil {
			log.Println("RSA Authentication Failed", err)
			return "", "", "", "", "", "", "", err
		}

		res1, err := esclient.Update(
			"automate-orch-queue", queueId,
			strings.NewReader(`{"doc": {"status": "pending","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			esclient.Update.WithPretty(),
			esclient.Update.WithIfSeqNo(seqNumber),
			esclient.Update.WithIfPrimaryTerm(primaryTerm),
		)
		if err != nil {
			log.Println("Error in updating ES document queue:", err)
			return "", "", "", "", "", "", "", err
		}

		log.Println("Status code:", res1.StatusCode)
		if res1.StatusCode != 200 {
			return "", "", "", "", "", "", "", err
		}

		res2, err := esclient.Update(
			"automate-orch-executions", executionId,
			strings.NewReader(`{"doc": {"status": "pending","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			esclient.Update.WithPretty(),
		)
		if err != nil {
			log.Println("Error in updating ES document queue:", err)
			return "", "", "", "", "", "", "", err
		}

		defer res1.Body.Close()
		defer res2.Body.Close()
	}
	log.Println("Checkqueue completed")
	return queueId, executionId, workerId, environmentUniqueName, botId, randomText, chipherText, err
}

func LaunchK8sJob(queueId string, executionId string, workerId string, tenantUniqueName string, environmentUniqueName string, botId string, randomText string, chipherText string) {

	log.Println("LaunchK8sJob started")

	getBotDetailsUrl := os.Getenv("GETBOTDETAILS_URL")
	if getBotDetailsUrl == "" {
		log.Println("Get Bot Details URL is missing. Exiting now.")
		orch.ReleaseWorker(workerId)
		return
	}

	downloadBotUrl := os.Getenv("DOWNLOADBOT_URL")
	if downloadBotUrl == "" {
		log.Println("Download Bot Details URL is missing. Exiting now.")
		orch.ReleaseWorker(workerId)
		return
	}

	sendResultUrl := os.Getenv("UPLOADRESULT_URL")
	if sendResultUrl == "" {
		log.Println("Send Result URL is missing. Exiting now.")
		orch.ReleaseWorker(workerId)
		return
	}

	botsHomeDir := os.Getenv("BOTSHOMEDIR")
	if botsHomeDir == "" {
		log.Println("Bots home directory is missing. Exiting now.")
		orch.ReleaseWorker(workerId)
		return
	}

	k8sclientset, err := kube.ReadConfig()
	if err != nil {
		log.Println("Failed to connect to Kubernetes")
		orch.ReleaseWorker(workerId)
		return
	}

	batchClient := k8sclientset.BatchV1()
	jobs := batchClient.Jobs("default")
	var backOffLimit int32 = 4

	nfs_info, err := GetStoragepath(workerId, tenantUniqueName)
	if err != nil {
		log.Println(err)
		orch.ReleaseWorker(workerId)
		return
	}
	_ = nfs_info
	log.Println("Storage Path retreived")

	labels := make(map[string]string)
	labels["QueueId"] = queueId
	labels["ExecutionId"] = executionId
	labels["WorkerId"] = workerId
	labels["BotType"] = "python"
	labels["TenantUniqueName"] = tenantUniqueName
	labels["EnvironmentUniqueName"] = environmentUniqueName

	rcc_proxy := os.Getenv("RCC_PROXY")
	rcc_no_proxy := os.Getenv("RCC_NO_PROXY")

	jobName := "botworker-" + tenantUniqueName + "-"

	ELASTIC_MASTER1 := os.Getenv("ELASTIC_MASTER1")
	ELASTIC_MASTER2 := os.Getenv("ELASTIC_MASTER2")
	ELASTIC_MASTER3 := os.Getenv("ELASTIC_MASTER3")
	ELASTIC_USERNAME := os.Getenv("ELASTIC_USERNAME")
	ELASTIC_PASSWORD := os.Getenv("ELASTIC_PASSWORD")

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobName,
			Namespace:    "default",
			Labels:       labels,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					ServiceAccountName: "default",
					Containers: []v1.Container{
						{
							Name:            "container-agent",
							Image:           "container-agent",
							ImagePullPolicy: "Never",
							Command:         []string{"container-agent"},
							Env: []v1.EnvVar{
								{
									Name:  "ELASTIC_MASTER1",
									Value: ELASTIC_MASTER1,
								},
								{
									Name:  "ELASTIC_MASTER2",
									Value: ELASTIC_MASTER2,
								},
								{
									Name:  "ELASTIC_MASTER3",
									Value: ELASTIC_MASTER3,
								},
								{
									Name:  "ELASTIC_USERNAME",
									Value: ELASTIC_USERNAME,
								},
								{
									Name:  "ELASTIC_PASSWORD",
									Value: ELASTIC_PASSWORD,
								},
								{
									Name:  "botsHomeDir",
									Value: botsHomeDir,
								},
								{
									Name:  "getBotDetailsUrl",
									Value: getBotDetailsUrl,
								},
								{
									Name:  "downloadBotUrl",
									Value: downloadBotUrl,
								},
								{
									Name:  "sendResultUrl",
									Value: sendResultUrl,
								},
								{
									Name:  "tenantUniqueName",
									Value: tenantUniqueName,
								},
								{
									Name:  "environmentUniqueName",
									Value: environmentUniqueName,
								},
								{
									Name:  "workerId",
									Value: workerId,
								},
								{
									Name:  "randomText",
									Value: randomText,
								},
								{
									Name:  "chipherText",
									Value: chipherText,
								},
								{
									Name:  "executionId",
									Value: executionId,
								},
								{
									Name:  "queueId",
									Value: queueId,
								},
								{
									Name:  "HTTP_PROXY",
									Value: rcc_proxy,
								},
								{
									Name:  "http_proxy",
									Value: rcc_proxy,
								},
								{
									Name:  "HTTPS_PROXY",
									Value: rcc_proxy,
								},
								{
									Name:  "https_proxy",
									Value: rcc_proxy,
								},
								{
									Name:  "no_proxy",
									Value: rcc_no_proxy,
								},
								{
									Name:  "NO_PROXY",
									Value: rcc_no_proxy,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: "/home/worker/.robocorp",
									Name:      "nfsvol",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "nfsvol",
							VolumeSource: v1.VolumeSource{
								NFS: &v1.NFSVolumeSource{
									Server:   nfs_info.NfsServerFqdn,
									Path:     nfs_info.NfsServerPath + "/" + tenantUniqueName,
									ReadOnly: false,
								},
							},
						},
					},

					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}
	_, err1 := jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err1 != nil {
		log.Println("Failed to create K8s job: " + err1.Error())
		orch.ReleaseWorker(workerId)
		//here update status of execution to failed
	} else {
		//print job details
		log.Println("Created K8s job successfully")
	}
}

func GetStoragepath(workerId string, tenantUniqueName string) (nfsShareInfo, error) {

	var nfsStoragePath string
	var selectedPath string

	nfsBasePath := os.Getenv("NFS_PATH")

	selectedIndex := -1
	selectedFreeSpace := 0
	_ = selectedFreeSpace
	var selectedRatio uint64
	selectedRatio = 0
	selectedMountPoint := ""

	client, err := elasticsearch.ESConnect()
	if err != nil {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Elasticsearch connection error:", err)

	}

	query := `{
		"query":
		  {"bool":
			{"must":
				[
					{"term": { "_id": "` + workerId + `" }}
					]}}
		  }`

	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		orch.ReleaseWorker(workerId)
		log.Fatalln("Search failed, Check search query", err)

	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) > 0 {
		nfsStoragePath = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["nfsStoragePath"])
	}

	shareinfo, err := NfsMount()
	if err != nil {
		log.Println("failed to get list of nfs storage", err)
		orch.ReleaseWorker(workerId)
	}

	for i, n := range shareinfo {
		// check if DB entry is good
		if n.FreeSpace == 0 {
			continue
		}
		if n.TotalSpace == 0 {
			continue
		}
		if n.NfsServerFqdn == "" {
			continue
		}
		if n.NfsServerPath == "" {
			continue
		}

		if nfsStoragePath == n.NfsServerFqdn+":"+n.NfsServerPath {
			// check if folder exists amd there is somedisk space left

			path_to_check := nfsBasePath + n.MountPoint + "/" + tenantUniqueName
			stat, err := os.Stat(path_to_check)
			if err == nil && stat.IsDir() {
				// path is a directory
				if n.FreeSpace > 10000000 {
					selectedIndex = i
					selectedPath = nfsStoragePath
					break
				}
			}
		} else {
			if n.FreeSpace > 10000000 {
				if n.TotalSpace/uint64(n.FolderCount+1) > uint64(selectedRatio) {
					selectedIndex = i
					//selected_free_space = int(n.Free_space)
					selectedRatio = n.TotalSpace / (uint64(n.FolderCount) + 1)
					selectedPath = n.NfsServerFqdn + ":" + n.NfsServerPath
					selectedMountPoint = n.MountPoint + "/" + tenantUniqueName

				}
			}
		}
	}
	if selectedMountPoint != "" {
		syscall.Umask(0)
		os.MkdirAll(nfsBasePath+selectedMountPoint, 0777)

		res, err := client.Update(
			"automate-orch-workers", workerId,
			strings.NewReader(`{"doc": {"nfsStoragePath": "`+selectedPath+`","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Println("Error in updating ES document queue:", err)
			orch.ReleaseWorker(workerId)
		}

		res.Body.Close()
	}

	if selectedIndex != -1 {
		return *shareinfo[selectedIndex], nil
	} else {
		orch.ReleaseWorker(workerId)
		return nfsShareInfo{}, errors.New("no NFS storage available")
	}

}

func NfsMount() ([]*nfsShareInfo, error) {

	var nsi []*nfsShareInfo
	var nfs_list []*nfsShareResult

	nfsBasePath := os.Getenv("NFS_PATH")

	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}

	query := `{"query":{
		"bool": {
				"filter": [
					{ "term": { "isEnabled": "true" }},
					{ "term": { "type": "nfs" }}
					]
				}
		}}`
	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-settings"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))
	for _, doc := range docs {

		mountPath := fmt.Sprint(doc["_source"].(map[string]interface{})["mountPath"])
		isEnabled := fmt.Sprint(doc["_source"].(map[string]interface{})["isEnabled"])
		nfsPath := fmt.Sprint(doc["_source"].(map[string]interface{})["nfsPath"])

		nfsShareResult := nfsShareResult{IsEnabled: isEnabled, MountPath: mountPath, NfsPath: nfsPath}
		nfs_list = append(nfs_list, &nfsShareResult)
	}

	for _, n := range nfs_list {
		mountPath := nfsBasePath + n.MountPath
		var infoEntry nfsShareInfo
		splitNfs := strings.Split(n.NfsPath, ":")
		if len(splitNfs) > 1 {
			infoEntry.NfsServerFqdn = splitNfs[0]
			infoEntry.NfsServerPath = splitNfs[1]
		} else {
			infoEntry.NfsServerFqdn = ""
			infoEntry.NfsServerPath = ""
		}
		infoEntry.MountPoint = n.MountPath

		var stat unix.Statfs_t
		unix.Statfs(mountPath, &stat)

		if err != nil {
			infoEntry.FreeSpace = 0
			infoEntry.TotalSpace = 0
		}

		infoEntry.FreeSpace = stat.Bavail * uint64(stat.Bsize)
		infoEntry.TotalSpace = stat.Blocks * uint64(stat.Bsize)

		files, _ := ioutil.ReadDir(mountPath)
		infoEntry.FolderCount = len(files)

		nsi = append(nsi, &infoEntry)
	}

	return nsi, nil
}
