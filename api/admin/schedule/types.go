package main

type getSchedule struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	ScheduleId            string `json:"scheduleId"`
	Email                 string `json:"email"`
	BotId                 string `json:"botId"`
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
	User                  string `json:"user"`
}

type updateSchedule struct {
	ScheduleId            string `json:"ScheduleId"`
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
	EndTime               string `json:"endTime"`
	Interval              string `json:"interval"`
	Frequency             string `json:"frequency"`
	Days                  string `json:"days"`
	BotMode               string `json:"botMode"`
	Priority              string `json:"priority"`
	MachineName           string `json:"machineName"`
	User                  string `json:"user"`
}

type listSchedule struct {
	ScheduleId string      `json:"_id"`
	Source     interface{} `json:"_source"`
}

type ApiResponse struct {
	Response interface{} `json:"Response"`
}
