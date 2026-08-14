package main

import (
	"net/http"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}
type Routes []Route

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Index,
	},
	Route{
		"Getbot",
		"POST",
		"/getbot",
		GetBot,
	},
	Route{
		"Createbot",
		"POST",
		"/createbot",
		CreateUpload,
	},
	Route{
		"Updatebot",
		"POST",
		"/updatebot",
		UpdateUpload,
	},
	Route{
		"Deletebot",
		"DELETE",
		"/deletebot",
		DeleteBot,
	},
	Route{
		"Deletebotpermanent",
		"DELETE",
		"/deletebotpermanent",
		DeleteBotPermenant,
	},
	Route{
		"Listbot",
		"POST",
		"/listbot",
		ListBot,
	},
	Route{
		"Enablebot",
		"POST",
		"/enablebot",
		EnableBot,
	},
	Route{
		"Disablebot",
		"POST",
		"/disablebot",
		DisableBot,
	},
	Route{
		"Executebot",
		"POST",
		"/executebot",
		ExecuteBot,
	},
	Route{
		"ExecutionResult",
		"POST",
		"/executionresult",
		ExecutionResult,
	},
	Route{
		"DownloadInventory",
		"POST",
		"/downloadinventory",
		DownloadInventory,
	},
	Route{
		"DownloadResult",
		"POST",
		"/downloadresult",
		DownloadResult,
	},
}
