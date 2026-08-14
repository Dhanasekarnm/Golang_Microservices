package main

type triggerBot struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotMode               string `json:"botMode"`
	BotId                 string `json:"botId"`
	Priority              int    `json:"priority"`
	TriggeredBy           string `json:"triggeredBy"`
	TriggerMode           string `json:"triggerMode"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
}

type executionBody struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotLabel              string `json:"botLabel"`
	BotUniqueName         string `json:"botUniqueName"`
	BotType               string `json:"botType"`
	BotId                 string `json:"botId"`
	Priority              int    `json:"priority"`
	QueueId               string `json:"queueId"`
	WorkerId              string `json:"workerId"`
	Status                string `json:"status"`
	S3InventoryPath       string `json:"s3InventoryPath"`
	S3ExecutionPath       string `json:"s3ExecutionPath"`
	Md5Checksum           string `json:"md5Checksum"`
	Timestamp             string `json:"timestamp"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
}

type queueBody struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	Priority              int    `json:"priority"`
	WorkerId              string `json:"workerId"`
	Status                string `json:"status"`
	Timestamp             string `json:"timestamp"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
	ExpirationTime        string `json:"expirationTime"`
	ExecutionId           string `json:"executionId"`
	BotId                 string `json:"botId"`
	BotMode               string `json:"botMode"`
}

type triggerBody struct {
	Timestamp            string `json:"timestamp"`
	TriggeredBy          string `json:"triggeredBy"`
	TriggerMode          string `json:"triggerMode"`
	QueueId              string `json:"queueId"`
	ExecutionId          string `json:"executionId"`
	PlannedExecutionTime string `json:"plannedExecutionTime"`
	BotId                string `json:"botId"`
}
