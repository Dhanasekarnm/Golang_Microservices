package main

type getEnvironment struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	Owner                 string `json:"owner"`
}

type createEnvironment struct {
	TenantUniqueName string `json:"tenantUniqueName"`
	EnvironmentLabel string `json:"environmentLabel"`
	Owner            string `json:"owner"`
}

type updateEnvironment struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	EnvironmentId         string `json:"environmentId"`
	Bots                  string `json:"bots"`
	Owner                 string `json:"owner"`
	State                 string `json:"state"`
}

type deleteEnvironment struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	EnvironmentId         string `json:"environmentId"`
	DeletedBy             string `json:"deletedBy"`
}

type updateEnvironmentRoles struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	EnvironmentId         string `json:"environmentId"`
	Owner                 string `json:"owner"`
	Developer             string `json:"developer"`
	RunOnly               string `json:"runOnly"`
}

type environment struct {
	TotalEnvironment  int `json:"totalEnvironment"`
	ActiveEnvironment int `json:"activeEnvironment"`
}

type listEnvironment struct {
	EnvironmentId string      `json:"_id"`
	Source        interface{} `json:"_source"`
}

type ApiResponse struct {
	Response interface{} `json:"Response"`
}
