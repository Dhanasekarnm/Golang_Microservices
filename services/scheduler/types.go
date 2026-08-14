package main

type triggerBot struct {
	TenantUniqueName      string `json:"tenantUniqueName"`
	EnvironmentUniqueName string `json:"environmentUniqueName"`
	BotMode               string `json:"botMode"`
	BotId                 string `json:"botId"`
	Priority              int    `json:"priority"`
	TriggeredBy           string `json:"triggeredBy"`
	TriggerMode           string `json:"triggerMode"`
	PlannedExecutionTime  string `json:"plannedExecutionTime"`
}
