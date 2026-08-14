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
