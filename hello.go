package main

import (
	"pythonrpa/orch"
)

func main() {

	// err := orch.GenerateLicense("/root/go/pythonrpa/services/machine-agent/license.orig", "/root/go/pythonrpa/services/machine-agent/license")
	// if err != nil {
	// 	println(err)
	// }

	workerID, cipherText, err := orch.DecryptLicense("/root/go/pythonrpa/services/machine-agent/license")
	if err != nil {
		println(err)
	}
	println(workerID)
	println(cipherText)

	/* 	dir_path, err := operations.UnPackBot("/home/worker/output.zip")

	   	log.Println(dir_path)
	   	log.Println(err) */

	/*log.Println(time.Now().UTC())
	log.Println(time.Now().Format(time.RFC3339))
	log.Println((time.Now().UTC()).Format(time.RFC3339))
	log.Println(os.Getenv("ELASTIC_MASTER1"))

	auth.GenerateRSAkey("TvNtbZAB1GvyVC9dq_BQ")
	ciphertext, _ := os.ReadFile("public.pem")
	byteSlice := []byte(ciphertext)
	encoded := base64.StdEncoding.EncodeToString(byteSlice)
	fmt.Println(encoded)*/

	/* Multipart start
	sendresulturl := "http://10.164.39.200:30084/uploadbotresult"

	extraParams := map[string]string{
		"executionId": "kfOP_ZAB1GvyVC9d2PG7",
	}
	zipfile := "printmessage.zip"
	request, err := fileUploadRequest(sendresulturl, extraParams, "bot_result", zipfile)
	if err != nil {
		log.Println("Error uploading results", err)
	}

	log.Println("Trying to upload Zip file ")
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal("Upload of zip  file failed,error: ", err)
	} else {
		var bodyContent []byte
		log.Println("Zip file upload finished with HTTP code ", resp.StatusCode)
		fmt.Println(resp.Header)
		resp.Body.Read(bodyContent)
		resp.Body.Close()
		fmt.Println(bodyContent)
	}*/ //Multipart end

}

/*
	func fileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
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
*/
