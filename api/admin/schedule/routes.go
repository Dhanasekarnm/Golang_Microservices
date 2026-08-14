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
		"Getschedule",
		"POST",
		"/getschedule",
		GetSchedule,
	},
	Route{
		"Createschedule",
		"POST",
		"/createschedule",
		CreateSchedule,
	},
	Route{
		"Updateschedule",
		"POST",
		"/updateschedule",
		UpdateSchedule,
	},
	Route{
		"Deleteschedule",
		"DELETE",
		"/deleteschedule",
		DeleteSchedule,
	},
	Route{
		"Listschedule",
		"POST",
		"/listschedule",
		ListSchedule,
	},
	Route{
		"Disableschedule",
		"POST",
		"/disableschedule",
		DisableSchedule,
	},
	Route{
		"Enableschedule",
		"POST",
		"/enableschedule",
		EnableSchedule,
	},
}
