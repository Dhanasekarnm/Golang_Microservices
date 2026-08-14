package main

type checkqueueRequest struct {
	WorkerId   string `json:"workerId"`
	Rt         string `json:"rt"`
	Ct         string `json:"ct"`
	WorkerMode string `json:"workerMode"`
}

type checkqueueResult struct {
	QueueId               string `json:"queueId"`
	WorkerId              string `json:"workerId"`
	ExecutionId           string `json:"executionId"`
	Status                string `json:"status"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
}

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

type updateStatusRequest struct {
	ExecutionId string `json:"executionId"`
	Result      string `json:"result"`
}
