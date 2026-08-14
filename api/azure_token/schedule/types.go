package main

type getSchedule struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	ScheduleId            string `json:"scheduleId"`
	User                  string `json:"User"`
}

type createSchedule struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotId                 string `json:"botId"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
	EndTime               string `json:"endTime"`
	Interval              string `json:"interval"`
	Frequency             string `json:"frequency"`
	Days                  string `json:"days"`
	BotMode               string `json:"botMode"`
	Priority              string `json:"priority"`
	MachineName           string `json:"machineName"`
	User                  string `json:"User"`
}

type updateSchedule struct {
	ScheduleId            string `json:"scheduleId"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotId                 string `json:"botId"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
	EndTime               string `json:"endTime"`
	Interval              string `json:"interval"`
	Frequency             string `json:"frequency"`
	Days                  string `json:"days"`
	BotMode               string `json:"botMode"`
	Priority              string `json:"priority"`
	MachineName           string `json:"machineName"`
	User                  string `json:"User"`
}

type listSchedule struct {
	ScheduleId string      `json:"_id"`
	Source     interface{} `json:"_source"`
}

type ApiResponse struct {
	StatusCode int         `json:"StatusCode"`
	Header     interface{} `json:"Header"`
	Body       interface{} `json:"Body"`
}

type tokenSchedule struct {
	Token string `json:"token"`
}
