package main

// type tokenTenant struct {
// 	Token string `json:"token"`
// }

type getTenant struct {
	TenantId         string `json:"tenantId"`
	TenantUniqueName string `json:"tenantUniqueName"`
	Owner            string `json:"owner"`
}
type createTenant struct {
	TenantLabel string `json:"tenantLabel"`
	Account     string `json:"account"`
	Workers     string `json:"workers"`
	Owner       string `json:"owner"`
}
type updateTenant struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	TenantId         string `json:"tenantId"`
	Account          string `json:"account"`
	Workers          string `json:"workers"`
	Owner            string `json:"owner"`
	State            string `json:"state"`
}

type deleteTenant struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	TenantId         string `json:"tenantId"`
	DeletedBy        string `json:"deletedBy"`
}

type updateTenantRole struct {
	TenantId         string `json:"tenantId"`
	TenantUniqueName string `json:"tenantUniqueName"`
	Owner            string `json:"owner"`
	Maker            string `json:"maker"`
}

type tenant struct {
	TotalTenant  int `json:"totalTenant"`
	ActiveTenant int `json:"activeTenant"`
}

// type workerBody struct {
// 	TenantUniqueName string `json:"tenantUniqueName"`
// 	WorkerMode       string `json:"workerMode"`
// 	WorkerPoolId     string `json:"workerPoolId"`
// 	NfsStoragePath   string `json:"nfsStoragePath"`
// 	State            string `json:"state"`
// 	WorkerLabel      string `json:"workerLabel"`
// 	PrivateKey       string `json:"privateKey"`
// 	PublicKey        string `json:"publicKey"`
// 	Timestamp        string `json:"timestamp"`
// }

type listTenant struct {
	TenantId string      `json:"_id"`
	Source   interface{} `json:"_source"`
}

type workerRequest struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	NfsStoragePath   string `json:"nfsStoragePath"`
}

type deleteworkerRequest struct {
	WorkerId string `json:"workerId"`
}

type ApiResponse struct {
	Response interface{} `json:"Response"`
}
