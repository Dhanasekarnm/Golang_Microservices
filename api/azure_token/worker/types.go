package main

type getWorker struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	WorkerId         string `json:"workerId"`
	User             string `json:"user"`
}
type createWorker struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	WorkerMode       string `json:"workerMode"`
	WorkerPoolId     string `json:"workerPoolId"`
	NfsStoragePath   string `json:"nfsStoragePath"`
	User             string `json:"user"`
}
type updateWorker struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	WorkerId         string `json:"workerId"`
	WorkerMode       string `json:"workerMode"`
	WorkerPoolId     string `json:"workerPoolId"`
	NfsStoragePath   string `json:"nfsStoragePath"`
	User             string `json:"user"`
}

type deleteWorker struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	WorkerId         string `json:"workerId"`
	State            string `json:"state"`
	User             string `json:"user"`
}

type worker struct {
	TotalWorker   int `json:"totalWorker"`
	IdleWorker    int `json:"idleWorker"`
	RunningWorker int `json:"runningWorker"`
}

type workerBody struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	WorkerMode       string `json:"workerMode"`
	WorkerPoolId     string `json:"workerPoolId"`
	NfsStoragePath   string `json:"nfsStoragePath"`
	State            string `json:"state"`
	WorkerLabel      string `json:"workerLabel"`
	PrivateKey       string `json:"privateKey"`
	PublicKey        string `json:"publicKey"`
	Timestamp        string `json:"timestamp"`
}

type listWorker struct {
	WorkerId string      `json:"_id"`
	Source   interface{} `json:"_source"`
}

type ApiResponse struct {
	StatusCode int         `json:"StatusCode"`
	Header     interface{} `json:"Header"`
	Body       interface{} `json:"Body"`
}

type tokenWorker struct {
	Token string `json:"token"`
}
