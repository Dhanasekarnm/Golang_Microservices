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
		"Getworker",
		"POST",
		"/getworker",
		GetWorker,
	},
	Route{
		"Createworker",
		"POST",
		"/createworker",
		CreateWorker,
	},
	Route{
		"Updateworker",
		"POST",
		"/updateworker",
		UpdateWorker,
	},
	Route{
		"Deleteworker",
		"DELETE",
		"/deleteworker",
		DeleteWorker,
	},
	Route{
		"Listworker",
		"POST",
		"/listworker",
		ListWorker,
	},
	Route{
		"workercount",
		"POST",
		"/workercount",
		WorkerCount,
	},
}
