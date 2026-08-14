package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pythonrpa/elasticsearch"
	"strings"
	"time"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Sendlog API is available!\n")
}

func CreateLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET;POST;OPTIONS")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(Red+"Error reading body of /sendlog request"+Reset, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	var t Sendlogreq
	var resp Sendlogresp
	err = json.Unmarshal(body, &t)
	if err == nil {
		log.Println("request received:", t)
	}
	yyyymm := time.Now().UTC().Format(YYYYMM)
	var index string
	if t.EnvironmentUniqueName == "" && t.TenantUniqueName == "" {
		index = "admin-logs-" + yyyymm
	} else if t.EnvironmentUniqueName == "" {
		index = t.TenantUniqueName + "-logs-" + yyyymm
	} else {
		index = t.TenantUniqueName + "-" + t.EnvironmentUniqueName + "-logs-" + yyyymm
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Println(Red+"Elasticsearch connection error:"+Reset, err)
		return
	}
	res, err := client.Index(
		index,
		strings.NewReader(`{
			"tenantUniqueName": "`+t.TenantUniqueName+`",
			"environmentUniqueName": "`+t.EnvironmentUniqueName+`",
			"entity":"`+t.Entity+`",
			"user": "`+t.User+`",
			"message":"`+t.Message+`",
			"timestamp":"`+timestamp+`",
			"level": "`+t.Level+`"
        }`),
		client.Index.WithPretty(),
	)
	if err != nil {
		log.Println(err)
		w.WriteHeader(res.StatusCode)
		resp.Message = fmt.Sprint(json.NewEncoder(w).Encode(err))
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer res.Body.Close()
	w.WriteHeader(res.StatusCode)
	log.Println(res.Status())
	json.NewEncoder(w).Encode(res)
}
