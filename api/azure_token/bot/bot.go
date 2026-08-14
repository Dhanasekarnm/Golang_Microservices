package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"pythonrpa/s3storage"
	"regexp"
	"strings"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "BOT API is available!\n")
}

func Validatezip(file_path string, file_name string, botuniquename string) (cond bool, robo bool, bot bool, tas bool, py bool, botid string) {
	var conda, robot, botfile, task, pyfile bool
	conda, robot, botfile, task, pyfile = false, false, false, false, false
	bots_base_path := os.Getenv("BOTSHOMEDIR")
	unzippath := bots_base_path + botuniquename + "/unzip/"
	_, err := operations.Unzip(file_path, unzippath)
	if err != nil {
		log.Println("Error creating zip file ", err)
	}
	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		log.Println("Error in generating queue UUID", err)
		return
	}
	botId := strings.TrimSuffix(string(uuid), "\n")
	file, err := os.Create(unzippath + ".bot.id") //create a new file
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	_, errs := file.WriteString(botId)
	if errs != nil {
		fmt.Println("Failed to write to .bot.id:", errs)
		return
	}
	log.Println(".bot.id is created successfully", botId)
	err1 := operations.Zip(unzippath, file_path)
	if err1 != nil {
		log.Println("Error creating zip file ", err1)
	}
	// Open the zip file
	read, err := zip.OpenReader(file_path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer read.Close()
	// Iterate through the files in the zip archive
	for _, f := range read.File {
		// Open the current file
		v, err := f.Open()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer v.Close()
		if f.Name == ".bot.id" {
			botfile = true
		} else if f.Name == "conda.yaml" {
			conda = true
		} else if f.Name == "robot.yaml" {
			robot = true
		} else if f.Name == "tasks.py" {
			task = true
		} else {
			var extension = filepath.Ext(f.Name)
			if extension == ".py" {
				pyfile = true
			}
		}
	}
	if !task && pyfile {
		log.Println("tasks.py is missing but other .py file exists")
	}
	return conda, robot, botfile, task, pyfile, botId
}

func CreateUpload(w http.ResponseWriter, r *http.Request) {
	bots_base_path := os.Getenv("BOTSHOMEDIR")
	Bucketname := r.FormValue("TenantUniqueName")
	env := r.FormValue("EnvironmentUniqueName")
	BotLabel := r.FormValue("BotLabel")
	Email := r.FormValue("Email")
	Account := r.FormValue("Account")
	UcmsId := r.FormValue("UcmsId")
	if Bucketname == "" || env == "" || BotLabel == "" || Email == "" || Account == "" {
		fmt.Fprintf(w, "Required details missing\n")
		return
	}
	botuniquename := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(BotLabel, ""))
	botuniquename = strings.ReplaceAll(botuniquename, " ", "")
	exist := CheckBotUniqueName(botuniquename, Bucketname, env)
	if exist {
		fmt.Fprintf(w, "BotUniqueName "+botuniquename+" already exist\n")
		return
	}
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)
	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("BotInventory")
	var resp []ApiResponse
	if err != nil {
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer file.Close()
	log.Println("Uploaded File name:", handler.Filename, "File Size:", handler.Size)

	filename := handler.Filename
	filepath := bots_base_path + botuniquename + "/" + filename
	err1 := os.RemoveAll(bots_base_path + botuniquename)
	if err1 != nil {
		log.Println(err1)
		return
	}
	err2 := os.MkdirAll(bots_base_path+botuniquename, 0777)
	if err2 != nil {
		log.Println(err2)
		return
	}
	dst, err := os.Create(filepath)
	if err != nil {
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer dst.Close()
	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("File Uploaded to container successfully")

	conda, robot, botfile, task, pyfile, botId := Validatezip(filepath, filename, botuniquename)
	log.Println("conda:", conda, "robot:", robot, "botfile:", botfile, "task:", task, "pyfile:", pyfile)
	if !conda || !robot || !botfile || (!task && !pyfile) {
		http.Error(w, "Uploaded file doesnt have required rcc packages, Upload failed", http.StatusInternalServerError)
		return
	}
	var uploadPath, versionId, md5Checksum string
	s := s3storage.S3Connect()
	uploadPath = env + "/bots/python/inventory/" + filename
	versionId, md5Checksum, err = s3storage.S3UploadFile(s, filepath, Bucketname, uploadPath)
	if err != nil {
		response := ApiResponse{StatusCode: 400, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	md5Checksum = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(md5Checksum, ""))
	log.Println("File Uploaded to S3 Successfully,  VersionID:", versionId, ",and md5checksum:", md5Checksum)
	fmt.Fprintf(w, "File Uploaded Successfully\n")
	dirName := bots_base_path + botuniquename
	err3 := os.RemoveAll(dirName)
	if err3 != nil {
		log.Println("error deleting the bot folder in container", err3)
		return
	}
	log.Println("Directory", dirName, "removed from bot container successfully")
	status := CreateBot(Bucketname, env, Email, BotLabel, botuniquename, Account, UcmsId, uploadPath, md5Checksum, versionId, botId)
	if !status {
		fmt.Fprintf(w, "Bot entry not created in elastic\n")
		elasticsearch.ESclient().FeaturesResetFeatures.WithHuman()
	}
	fmt.Fprintf(w, "Bot details created successfully\n")
}

func CreateBot(tenantUniqueName string, environmentUniqueName string, email string, botLabel string, botUniqueName string, account string, ucmsId string, s3InventoryPath string, md5Checksum string, s3InventoryVersionId string, botId string) (state bool) {
	status := false
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "createBot"
	cfg.TenantUniqueName = tenantUniqueName
	cfg.EnvironmentUniqueName = environmentUniqueName
	cfg.User = email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client := elasticsearch.ESclient()
	botbody := botBody{
		TenantUniqueName:      tenantUniqueName,
		EnvironmentUniqueName: environmentUniqueName,
		BotType:               "python",
		BotLabel:              botLabel,
		BotUniqueName:         botUniqueName,
		S3InventoryPath:       s3InventoryPath,
		S3InventoryVersionId:  s3InventoryVersionId,
		Md5checksum:           md5Checksum,
		State:                 "active",
		CreatedBy:             email,
		CreatedOn:             timestamp,
		Timestamp:             timestamp,
	}
	bdata, _ := json.Marshal(botbody)
	res, err := client.Index("automate-orch-bots", bytes.NewReader(bdata), client.Index.WithDocumentID(botId), client.Index.WithPretty())
	if err != nil {
		log.Println("Create Bot error: ", err)
		return
	}
	log.Println("Bot created status:", res)
	defer res.Body.Close()
	res1, err := client.Index(
		tenantUniqueName+"-"+environmentUniqueName+"-inventory",
		strings.NewReader(`{
				"tenantUniqueName":"`+tenantUniqueName+`",
				"environmentUniqueName":"`+environmentUniqueName+`",															
				"botId":"`+botId+`",				
				"botType":"python",	
				"botLabel":"`+botLabel+`",
				"botUniqueName":"`+botUniqueName+`",
				"account":"`+account+`",
				"state":"active",	
				"email":"`+email+`",
				"ucmsId":"`+ucmsId+`",
				"createdOn":"`+timestamp+`",		
				"createdBy":"`+email+`",																							
				"timestamp":"`+timestamp+`"
        	}`),
		client.Index.WithPretty(),
	)
	if err != nil {
		log.Println("Create Bot Inventory error", err)
		return
	}
	defer res1.Body.Close()
	status = true
	log.Println("Inventory created status:", res1.StatusCode)
	cfg.Message = "CreateBot response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	return status
}

func UpdateUpload(w http.ResponseWriter, r *http.Request) {
	bots_base_path := os.Getenv("BOTSHOMEDIR")
	Bucketname := r.FormValue("TenantUniqueName")
	env := r.FormValue("EnvironmentUniqueName")
	Email := r.FormValue("Email")
	Account := r.FormValue("Account")
	UcmsId := r.FormValue("UcmsId")
	BotId := r.FormValue("BotId")
	if BotId == "" || Bucketname == "" || env == "" || Email == "" || Account == "" {
		fmt.Fprintf(w, "Required data is missing, please fill all the details\n")
		return
	}
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + BotId + `" }}
	]}}
	}`
	client := elasticsearch.ESclient()
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := "No records found with BotId: " + BotId
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var filename string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		Inventory := hit.(map[string]interface{})["_source"].(map[string]interface{})["inventory"].(string)
		_, filename = filepath.Split(Inventory)
	}
	var uploadPath, versionId, md5Checksum string
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)
	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("BotInventory")
	if filename != handler.Filename {
		resbody := "Bot Filename is differnt from Existing name in S3, please update and upload again"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	if file != nil {
		if err != nil {
			response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
			resp = append(resp, response)
			json.NewEncoder(w).Encode(resp)
			return
		}
		defer file.Close()
		fmt.Printf("Uploaded File: %+v\n", handler.Filename)
		fmt.Printf("File Size: %+v\n", handler.Size)

		// Create file
		filename := handler.Filename
		filepath := bots_base_path + Bucketname + "/" + filename
		err1 := os.MkdirAll(bots_base_path+Bucketname, 0777)
		if err1 != nil {
			log.Println(err)
			return
		}
		dst, err := os.Create(filepath)
		if err != nil {
			response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
			resp = append(resp, response)
			json.NewEncoder(w).Encode(resp)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the created file on the filesystem
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("File Uploaded to container successfully")
		botUniqueName := ""
		conda, robot, botfile, task, pyfile, botId := Validatezip(filepath, filename, botUniqueName)
		log.Println("conda:", conda, "robot:", robot, "botfile:", botfile, "task:", task, "pyfile:", pyfile, "botId:", botId)
		if !conda || !robot || !botfile || (!task && !pyfile) {
			http.Error(w, "Uploaded file doesnt have required rcc packages, Upload failed", http.StatusInternalServerError)
			return
		}
		s := s3storage.S3Connect()
		uploadPath = env + "/bots/python/inventory/" + filename
		versionId, md5Checksum, err = s3storage.S3UploadFile(s, filepath, Bucketname, uploadPath)
		if err != nil {
			response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
			resp = append(resp, response)
			json.NewEncoder(w).Encode(resp)
			return
		}
		md5Checksum = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(md5Checksum, ""))
		log.Println("File Uploaded Successfully,  VersionID:", versionId, ",and md5checksum:", md5Checksum)
		fmt.Fprintf(w, "File Uploaded Successfully\n")
		status := UpdateBot(BotId, Bucketname, env, Email, Account, UcmsId, uploadPath, md5Checksum, versionId)
		if !status {
			fmt.Fprintf(w, "Bot entry not updated in elastic\n")
			return
		}
		fmt.Fprintf(w, "Bot details updated successfully\n")
	} else {
		status := UpdateBot(BotId, Bucketname, env, Email, Account, UcmsId, uploadPath, md5Checksum, versionId)

		if !status {
			fmt.Fprintf(w, "Bot entry not updated in elastic\n")
			return
		}
		fmt.Fprintf(w, "Bot details updated successfully\n")
	}
}

func UpdateBot(botId string, tenantUniqueName string, environmentUniqueName string, email string, account string, ucmsId string, s3InventoryPath string, md5Checksum string, s3InventoryVersionId string) (state bool) {
	status := false
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.Entity = "updateBot"
	cfg.TenantUniqueName = tenantUniqueName
	cfg.EnvironmentUniqueName = environmentUniqueName
	cfg.User = email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client := elasticsearch.ESclient()
	if s3InventoryVersionId != "" && s3InventoryPath != "" && md5Checksum != "" {
		res, err := client.Update(
			"automate-orch-bots",
			botId,
			strings.NewReader(`{"doc": {
					"s3InventoryPath":"`+s3InventoryPath+`",
					"s3InventoryVersionId":"`+s3InventoryVersionId+`",
					"md5Checksum":"`+md5Checksum+`",
					"modifiedBy":"`+email+`",
					"timestamp":"`+timestamp+`"
				}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Println("Failed to update Bot", err)
			return
		}
		res.Body.Close()
		log.Println("Update bot " + botId + " Status: " + res.Status())
	} else {
		res, err := client.Update(
			"automate-orch-bots",
			botId,
			strings.NewReader(`{"doc": {					
					"modifiedBy":"`+email+`",
					"timestamp":"`+timestamp+`"
				}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Println("Failed to update Bot", err)
			return
		}
		res.Body.Close()
		log.Println("Update bot " + botId + " Status: " + res.Status())
	}
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botId.keyword": "` + botId + `" }}
	]}}
	}`
	res1, err := client.Search(
		client.Search.WithIndex(tenantUniqueName+"-"+environmentUniqueName+"-inventory"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println("Updatebot search Bot Id error: ", err)
		return
	}
	defer res1.Body.Close()
	if err := json.NewDecoder(res1.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		log.Println("No records found in inventory for BotId: " + botId)
		return
	}
	var inventoryid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		inventoryid = hit.(map[string]interface{})["_id"].(string)
	}
	res2, err := client.Update(
		tenantUniqueName+"-"+environmentUniqueName+"-inventory",
		inventoryid,
		strings.NewReader(`{"doc": {				
				"account":"`+account+`",
				"email":"`+email+`",
				"ucmsid":"`+ucmsId+`",
				"modifiledBy":"`+email+`",
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		log.Println("Bot inventory updated error: ", err)
		return
	}
	res2.Body.Close()
	log.Println("Inventory updated status:", res2.StatusCode)
	status = true
	cfg.Message = "UpdateBot status: " + res2.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	return status
}

func GetBot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /getBot request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botUniqueName": "` + t.BotUniqueName + `" }}
	]}}
	}`
	client := elasticsearch.ESclient()
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}

	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := "No records found with BotUniqueName: " + t.BotUniqueName
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var Bot_list []listBot
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Botlist := listBot{BotId: id, Source: source}
		Bot_list = append(Bot_list, Botlist)
	}
	var resbody []listBot
	if res.StatusCode == 200 {
		resbody = Bot_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DeleteBot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deletebot request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Entity = "deleteBot"
	cfg.User = t.Email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botId.keyword": "` + t.BotId + `" }}
	]}}
	}`
	client := elasticsearch.ESclient()
	res1, err := client.Update(
		"automate-orch-bots",
		t.BotId,
		strings.NewReader(`{"doc": {								
				"state":"deleted",
				"modifiedBy":"`+t.Email+`",
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "BotId '" + t.BotId + "' deleted with following Status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res1.StatusCode == 200 {
		resbody = "BOT Deleted sucessfully"
	}
	response := ApiResponse{StatusCode: res1.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res1.StatusCode)
	json.NewEncoder(w).Encode(resp)
	defer res1.Body.Close()
	res2, err := client.Search(
		client.Search.WithIndex(t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res2.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res2.Body.Close()
	if err := json.NewDecoder(res2.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	m := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if m == 0 {
		log.Println("BotId:" + t.BotId + " not found in inventory")
		return
	}
	var inventoryid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		inventoryid = hit.(map[string]interface{})["_id"].(string)
	}
	res3, err := client.Update(
		t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory",
		inventoryid,
		strings.NewReader(`{"doc": {								
				"state":"deleted",
				"modifiledBy":"`+t.Email+`",
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res3.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res3.Body.Close()
	log.Println("Inventory deleted status:", res3.StatusCode)
	res4, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res3.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res4.Body.Close()
	if err := json.NewDecoder(res4.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	o := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if o == 0 {
		log.Println("BotId: " + t.BotId + " not found in schedules")
		return
	}
	var scheduleid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		scheduleid = hit.(map[string]interface{})["_id"].(string)
	}
	res5, err := client.Update(
		"automate-orch-schedules",
		scheduleid,
		strings.NewReader(`{"doc": {				
				"state":"deleted",								
				"modifiledBy":"`+t.Email+`",
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res5.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res5.Body.Close()
	log.Println("Bot schedule deleted status:", res5.StatusCode)
}

func DeleteBotPermenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// read body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /deletebot request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Entity = "deleteBotPermenant"
	cfg.User = t.Email
	cfg.Level = "info"
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query1 = `{ "query":{"bool":{"must":[
	{ "term": { "_id": "` + t.BotId + `" }},
	{ "term": { "state.keyword": "deleted" }}
	]}}
	}`
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botId.keyword": "` + t.BotId + `" }}
	]}}
	}`
	client := elasticsearch.ESclient()
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query1)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := "BotId:" + t.BotId + "not found in Deleted state"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var s3InventoryPath string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		s3InventoryPath = hit.(map[string]interface{})["_source"].(map[string]interface{})["s3InventoryPath"].(string)
	}
	err1 := s3storage.S3Delete(t.TenantUniqueName, s3InventoryPath)
	if err1 != nil {
		log.Println(err1)
		return
	}
	_, filename := filepath.Split(s3InventoryPath)
	Path := strings.Split(s3InventoryPath, "/inventory")
	s3ExecutionPath := Path[0] + "/execution/" + filename
	err2 := s3storage.S3Delete(t.TenantUniqueName, s3ExecutionPath)
	if err2 != nil {
		log.Println(err2)
		return
	}
	res1, err := client.Delete(
		"automate-orch-bots",
		t.BotId,
		client.Delete.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "DeleteBotPermenant '" + t.BotId + " response status: " + res1.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status: ", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "Bot Deleted Permenantly"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
	defer res1.Body.Close()
	res2, err := client.Search(
		client.Search.WithIndex(t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res2.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res2.Body.Close()
	if err := json.NewDecoder(res2.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	m := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if m == 0 {
		log.Println("BotId:" + t.BotId + "not found in inventory")
		return
	}
	var inventoryid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		inventoryid = hit.(map[string]interface{})["_id"].(string)
	}
	res3, err := client.Delete(
		t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory",
		inventoryid,
		client.Delete.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res3.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res3.Body.Close()
	log.Println("Inventory deleted status:", res3.StatusCode)
	res4, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res4.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res4.Body.Close()
	if err := json.NewDecoder(res4.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	o := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if o == 0 {
		log.Println("BotId: " + t.BotId + "not found in schedules")
		return
	}
	var scheduleid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		scheduleid = hit.(map[string]interface{})["_id"].(string)
	}
	res5, err := client.Delete(
		"automate-orch-schedules",
		scheduleid,
		client.Delete.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res5.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res5.Body.Close()
	log.Println("Bot schedule deleted status:", res5.StatusCode)
}

func ListBot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listbot request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t getBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var resp []ApiResponse
	var query = `{"size":100, "query":{"bool":{"must":[
	{ "term": { "tenantUniqueName": "` + t.TenantUniqueName + `" }},
	{ "term": { "environmentUniqueName": "` + t.EnvironmentUniqueName + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		resbody := "No records found in BOT Index"
		response := ApiResponse{StatusCode: 200, Header: "Content-Type:application/json", Body: resbody}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	var Bot_list []listBot
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		id := hit.(map[string]interface{})["_id"].(string)
		Botlist := listBot{BotId: id, Source: source}
		Bot_list = append(Bot_list, Botlist)
	}
	var resbody []listBot
	if res.StatusCode == 200 {
		resbody = Bot_list
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
}

func DisableBot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /listbot request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Entity = "disableBot"
	cfg.User = t.Email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-bots",
		t.BotId,
		strings.NewReader(`{"doc": {					
					"state":"disabled",
					"modifiedBy":"`+t.Email+`",
					"timestamp":"`+timestamp+`"
				}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "Disablebot '" + t.BotId + "' response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "BOT disabled sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
	defer res.Body.Close()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botId.keyword": "` + t.BotId + `" }}
	]}}
	}`
	res1, err := client.Search(
		client.Search.WithIndex(t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res1.Body.Close()
	if err := json.NewDecoder(res1.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		log.Println("No records found in inventory for BotId: " + t.BotId)
		return
	}
	var inventoryid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		inventoryid = hit.(map[string]interface{})["_id"].(string)
	}
	res2, err := client.Update(
		t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory",
		inventoryid,
		strings.NewReader(`{"doc": {				
			"state":"disabled",				
			"modifiledBy":"`+t.Email+`",
			"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res2.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res2.Body.Close()
	log.Println("Bot Inventory disabled status:", res2.StatusCode)
	res3, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res3.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res3.Body.Close()
	if err := json.NewDecoder(res3.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	m := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if m == 0 {
		log.Println("No records found in schedules for BotId: " + t.BotId)
		return
	}
	var scheduleid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		scheduleid = hit.(map[string]interface{})["_id"].(string)
	}
	res4, err := client.Update(
		"automate-orch-schedules",
		scheduleid,
		strings.NewReader(`{"doc": {				
				"state":"disabled",								
				"modifiledBy":"`+t.Email+`",
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res4.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res4.Body.Close()
	log.Println("Bot schedule disabled status:", res4.StatusCode)
}

func EnableBot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /Enablebot request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var t deleteBot
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	sendlog_url := os.Getenv("SEND_LOG_URL")
	var cfg operations.Sendlogreq
	cfg.TenantUniqueName = t.TenantUniqueName
	cfg.EnvironmentUniqueName = t.EnvironmentUniqueName
	cfg.Entity = "enableBot"
	cfg.User = t.Email
	cfg.Level = "info"
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var resp []ApiResponse
	client := elasticsearch.ESclient()
	res, err := client.Update(
		"automate-orch-bots",
		t.BotId,
		strings.NewReader(`{"doc": {					
					"state":"active",
					"modifiedBy":"`+t.Email+`",
					"timestamp":"`+timestamp+`"
				}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	cfg.Message = "Enablebot '" + t.BotId + "' response status: " + res.Status()
	sendlog := operations.SendLog(cfg, sendlog_url)
	log.Println("Sendlog status:", fmt.Sprint(sendlog.Message))
	resbody := ""
	if res.StatusCode == 200 {
		resbody = "BOT Enabled sucessfully"
	}
	response := ApiResponse{StatusCode: res.StatusCode, Header: "ContenType:application/json", Body: resbody}
	resp = append(resp, response)
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(resp)
	defer res.Body.Close()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botId.keyword": "` + t.BotId + `" }}
	]}}
	}`
	res1, err := client.Search(
		client.Search.WithIndex(t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res1.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res1.Body.Close()
	if err := json.NewDecoder(res1.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		log.Println("No records found in inventory for BotId: " + t.BotId)
		return
	}
	var inventoryid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		inventoryid = hit.(map[string]interface{})["_id"].(string)
	}
	res2, err := client.Update(
		t.TenantUniqueName+"-"+t.EnvironmentUniqueName+"-inventory",
		inventoryid,
		strings.NewReader(`{"doc": {				
			"state":"active",				
			"modifiledBy":"`+t.Email+`",
			"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res2.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res2.Body.Close()
	log.Println("Bot Inventory enabled status:", res2.StatusCode)
	res3, err := client.Search(
		client.Search.WithIndex("automate-orch-schedules"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res3.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res3.Body.Close()
	if err := json.NewDecoder(res3.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	m := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if m == 0 {
		log.Println("No records found in schedules for BotId: " + t.BotId)
		return
	}
	var scheduleid string
	for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		scheduleid = hit.(map[string]interface{})["_id"].(string)
	}
	res4, err := client.Update(
		"automate-orch-schedules",
		scheduleid,
		strings.NewReader(`{"doc": {				
				"state":"active",								
				"modifiledBy":"`+t.Email+`",
				"timestamp":"`+timestamp+`"
			}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		response := ApiResponse{StatusCode: res4.StatusCode, Header: "Content-Type:application/json", Body: err}
		resp = append(resp, response)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res4.Body.Close()
	log.Println("Bot schedule enabled status:", res4.StatusCode)
}

func CheckBotUniqueName(botuniquename string, tenantuniquename string, envuniquename string) (exists bool) {
	client := elasticsearch.ESclient()
	var mapResp map[string]interface{}
	var query = `{ "query":{"bool":{"must":[
	{ "term": { "botUniqueName": "` + botuniquename + `" }}
	 { "term": { "tenantUniqueName": "` + tenantuniquename + `" }}
	  { "term": { "environmentUniqueName": "` + envuniquename + `" }}
	]}}
	}`
	res, err := client.Search(
		client.Search.WithIndex("automate-orch-bots"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Println(Red+"Elasticsearch Search() API ERROR: "+Reset, err)
		return
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
		log.Printf(Red+"Error parsing the response body: %s"+Reset, err)
		return
	}
	exist := true
	n := int(mapResp["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	if n == 0 {
		exist = false
		log.Println("botuniquename: " + botuniquename + " is valid")
		return
	}
	return exist
}
