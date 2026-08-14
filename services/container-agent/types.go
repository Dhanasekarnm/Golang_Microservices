package main

type getbotdetailsRequest struct {
	ExecutionId           string `json:"executionId"`
	RandomText            string `json:"randomText"`
	ChipherText           string `json:"chipherText"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
}

type getbotdetailsResult struct {
	ExecutionId           string `json:"executionId"`
	QueueId               string `json:"queue_id"`
	Status                string `json:"status"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	S3InventoryPath       string `json:"s3InventoryPath"`
	Md5Checksum           string `json:"md5Checksum"`
	BotUniqueName         string `json:"botUniqueName"`
}
