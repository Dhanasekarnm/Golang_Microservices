package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"pythonrpa/elasticsearch"
	"pythonrpa/operations"
	"strings"
	"time"
)

func GetBotDetails(gbr getbotdetailsRequest, getbotdetails_url string) (getbotdetailsResult, error) {

	var gbrout getbotdetailsResult

	gbr_json, _ := json.Marshal(gbr)
	gbr_json_byte := []byte(gbr_json)

	client := &http.Client{}

	req, err := http.NewRequest("POST", getbotdetails_url, bytes.NewBuffer(gbr_json_byte))
	if err != nil {
		log.Println("NewRequest: ", err)
		return getbotdetailsResult{}, errors.New("HTTP Post failed")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	response, error := client.Do(req)
	if error != nil {
		log.Println("NewRequest: ", err)
		return getbotdetailsResult{}, errors.New("HTTP Post failed")
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(body, &gbrout)
	return gbrout, nil
}

func DownloadBot(gbdr getbotdetailsResult, getBotDetailsCfg getbotdetailsRequest, downloadbot_url string) (string, error) {

	download_flag := true

	dir, filename := filepath.Split(gbdr.S3InventoryPath)
	bots_base_path := os.Getenv("botsHomeDir")
	bots_path := bots_base_path + "/" + dir
	bots_full_path := bots_path + filename

	if !operations.PathExists(bots_path) {
		os.MkdirAll(bots_path, 0755)
	}

	if operations.PathExists(bots_full_path) {
		bot_file, err := os.Open(bots_full_path)

		if err != nil {
			log.Println("Failed to open folder", err)
			return "", err
		}

		hash := md5.New()
		_, err = io.Copy(hash, bot_file)
		if err != nil {
			log.Println("Failed to copy hash file", err)
			return "", err
		}

		if gbdr.Md5Checksum == string(hex.EncodeToString(hash.Sum(nil))) {
			download_flag = false
		} else {
			log.Println("File exists, download not started")
			fileInfo, err := os.Stat(bots_full_path)
			if err != nil {
				log.Println(err)
			}
			fileSize := fileInfo.Size()
			log.Printf("Before unpack, Size of the file: %d bytes\n", fileSize)
			workdir_path, err := operations.UnPackBot(bots_full_path)
			if err != nil {
				return "", err
			}
			return workdir_path, nil
		}
	}

	if download_flag {
		log.Println("Download started")
		botcfg_json, _ := json.Marshal(getBotDetailsCfg)
		botcfg_json_byte := []byte(botcfg_json)

		client := &http.Client{}

		req, err := http.NewRequest("POST", downloadbot_url, bytes.NewBuffer(botcfg_json_byte))
		if err != nil {
			log.Println("DownloadBot error1: ", err)
			return "", err
		}
		req.Header.Add("Content-Type", "application/json")

		response, err := client.Do(req)
		log.Println("Download response status code", response.StatusCode)
		if err != nil {
			log.Println("DownloadBot error2: ", err)
			return "", err
		}
		defer response.Body.Close()

		if !operations.PathExists(bots_full_path) {
			os.MkdirAll(bots_path, 0755)
		}

		file_to_save, err := os.Create(bots_full_path)

		if err != nil {
			log.Println("DownloadBot error3: ", err)
			return "", err
		}
		defer file_to_save.Close()

		data_buffer, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println("DownloadBot error4: ", err)
			return "", err
		}
		bytesWritten, err := file_to_save.Write(data_buffer)
		log.Println("Bytes written", bytesWritten)
		if err != nil {
			log.Println("Error in file written", err)
		}

		fileInfo, err := os.Stat(bots_full_path)
		if err != nil {
			log.Println(err)
		}
		fileSize := fileInfo.Size()
		log.Printf("Before unpack, Size of the file: %d bytes\n", fileSize)

		workdir_path, err := operations.UnPackBot(bots_full_path)
		if err != nil {
			return "", err
		}

		os.Remove(bots_full_path)
		return workdir_path, nil
	}

	return "", errors.New("something failed")
}

func ExecuteBot(executionPath string, executionId string) {
	goExecutable, _ := exec.LookPath("rcc")
	cmd := exec.Cmd{
		Path:   goExecutable,
		Args:   []string{goExecutable, "run"},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Dir:    executionPath,
	}
	log.Println(cmd.String())

	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}
	res, err := client.Update(
		"automate-orch-executions", executionId,
		strings.NewReader(`{"doc": {"result": "started","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
		client.Update.WithPretty(),
	)
	if err != nil {
		log.Fatalln("Error in updating ES document queue:", err)
	}
	res.Body.Close()

	err = cmd.Run() //Executing the RCC bot

	log.Println("Execution error:", err)

	if err != nil {
		res, err := client.Update(
			"automate-orch-executions", executionId,
			strings.NewReader(`{"doc": {"result": "Failed","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Fatalln("Error in updating ES document queue:", err)
		}
		res.Body.Close()
	} else {
		res, err := client.Update(
			"automate-orch-executions", executionId,
			strings.NewReader(`{"doc": {"result": "Succeeded","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}}`),
			client.Update.WithPretty(),
		)
		if err != nil {
			log.Fatalln("Error in updating ES document queue:", err)
		}
		res.Body.Close()
	}
}

func UploadResult(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fi.Name())
	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	base, err := url.Parse(uri)
	if err != nil {
		log.Println(err)
	}

	// Path params
	//base.Path += "this will get automatically encoded"

	// Query params
	params2 := url.Values{}

	for key, val := range params {
		_ = writer.WriteField(key, val)
		params2.Add(key, val)
	}
	base.RawQuery = params2.Encode()

	err = writer.Close()
	if err != nil {
		return nil, err

	}

	//return http.NewRequest("POST", uri, body)
	req, err := http.NewRequest("POST", base.String(), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}
