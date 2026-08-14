package main

type botBody struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotType               string `json:"botType"`
	BotLabel              string `json:"botLabel"`
	BotUniqueName         string `json:"botUniqueName"`
	Md5checksum           string `json:"md5Checksum"`
	S3InventoryPath       string `json:"s3InventoryPath"`
	S3InventoryVersionId  string `json:"s3InventoryVersionId"`
	State                 string `json:"state"`
	CreatedBy             string `json:"createdBy"`
	CreatedOn             string `json:"createdOn"`
	Timestamp             string `json:"timestamp"`
}

//	type createBot struct {
//		TenantUniqueName      string `json:"tenantUniqueName"`
//		EnvironmentUniqueName string `json:"environmentUniqueName"`
//		BotLabel              string `json:"botLabel"`
//		Email                 string `json:"email"`
//		UcmsId                string `json:"ucmsId"`
//		Account               string `json:"account"`
//		FilePath              string `json:"filePath"`
//	}
type getBot struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotUniqueName         string `json:"botUniqueName"`
}

type deleteBot struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotId                 string `json:"botId"`
	Email                 string `json:"email"`
}

type listBot struct {
	BotId  string      `json:"_id"`
	Source interface{} `json:"_source"`
}

type ApiResponse struct {
	StatusCode int         `json:"StatusCode"`
	Header     interface{} `json:"Header"`
	Body       interface{} `json:"Body"`
}

type tokenBot struct {
	Token string `json:"token"`
}
