package main

type updateStatus struct {
	ExecutionId string `json:"executionId"`
	Result      string `json:"result"`
}

type resultStartBody struct {
	Timestamp string `json:"timestamp"`
	Result    string `json:"result"`
	StartTime string `json:"startTime"`
}

type resultEndBody struct {
	Timestamp string `json:"timestamp"`
	Result    string `json:"result"`
	EndTime   string `json:"endTime"`
}
