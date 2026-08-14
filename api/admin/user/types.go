package main

type getUser struct {
	Email string `json:"email"`
}

type createUser struct {
	Email string `json:"email"`
	Type  string `json:"type"`
}

type updateUser struct {
	UserId string `json:"userId"`
	Email  string `json:"email"`
	Type   string `json:"type"`
	State  string `json:"state"`
}

type deleteUser struct {
	UserId string `json:"userId"`
	Email  string `json:"email"`
}

type checkUserRole struct {
	Email                 string `json:"email"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
}

type userRoleResponse struct {
	TenantOwner      bool `json:"tenantOwner"`
	EnvironmentMaker bool `json:"environmentMaker"`
	EnvironmentOwner bool `json:"environmentOwner"`
	Developer        bool `json:"developer"`
	RunOnly          bool `json:"runOnly"`
}

type user struct {
	TotalUser  int `json:"totalUser"`
	ActiveUser int `json:"activeUser"`
}

type listUser struct {
	UserId string      `json:"_id"`
	Source interface{} `json:"_source"`
}

type ApiResponse struct {
	Response interface{} `json:"Response"`
}
