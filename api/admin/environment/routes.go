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
		"Getenvironment",
		"POST",
		"/getenvironment",
		GetEnvironment,
	},
	Route{
		"Createenvironment",
		"POST",
		"/createenvironment",
		CreateEnvironment,
	},
	Route{
		"Updateenvironment",
		"POST",
		"/updateenvironment",
		UpdateEnvironment,
	},
	Route{
		"Deletetenant",
		"DELETE",
		"/deleteenvironment",
		DeleteEnvironment,
	},
	Route{
		"Listenvironment",
		"POST",
		"/listenvironment",
		ListEnvironment,
	},
	Route{
		"Environmentcount",
		"POST",
		"/environmentcount",
		EnvironmentCount,
	},
	Route{
		"Addowner",
		"POST",
		"/addowner",
		AddEnvironmentOwner,
	},
	Route{
		"Adddeveloper",
		"POST",
		"/adddeveloper",
		AddDeveloper,
	},
	Route{
		"Addrunonly",
		"POST",
		"/addrunonly",
		AddRunOnly,
	},
	Route{
		"Removeowner",
		"DELETE",
		"/removeowner",
		RemoveEnvironmentOwner,
	},
	Route{
		"Removedeveloper",
		"DELETE",
		"/removedeveloper",
		RemoveDeveloper,
	},
	Route{
		"Removerunonly",
		"DELETE",
		"/removerunonly",
		RemoveRunOnly,
	},
}
