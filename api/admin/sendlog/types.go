package main

type Sendlogreq struct {
	Message               string `json:"message"`
	User                  string `json:"user"`
	Entity                string `json:"entity"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	Level                 string `json:"level"`
}
type Sendlogresp struct {
	Message string `json:"message"`
}
