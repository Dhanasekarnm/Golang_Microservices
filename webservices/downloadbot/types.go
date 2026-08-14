package main

type downloadbotRequest struct {
	ExecutionId           string `json:"executionId"`
	RandomText            string `json:"randomText"`
	ChipherText           string `json:"chipherText"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
}
