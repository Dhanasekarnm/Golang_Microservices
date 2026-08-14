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
		tokenHandler,
	},
	Route{
		"Getworker",
		"POST",
		"/getworker",
		tokenHandler,
	},
	Route{
		"Createworker",
		"POST",
		"/createworker",
		tokenHandler,
	},
	Route{
		"Updateworker",
		"POST",
		"/updateworker",
		tokenHandler,
	},
	Route{
		"Deleteworker",
		"DELETE",
		"/deleteworker",
		tokenHandler,
	},
	Route{
		"Listworker",
		"POST",
		"/listworker",
		tokenHandler,
	},
	Route{
		"workercount",
		"POST",
		"/workercount",
		tokenHandler,
	},
}
