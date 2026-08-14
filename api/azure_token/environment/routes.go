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
		"Getenvironment",
		"POST",
		"/getenvironment",
		tokenHandler,
	},
	Route{
		"Createenvironment",
		"POST",
		"/createenvironment",
		tokenHandler,
	},
	Route{
		"Updateenvironment",
		"POST",
		"/updateenvironment",
		tokenHandler,
	},
	Route{
		"Deletetenant",
		"DELETE",
		"/deleteenvironment",
		tokenHandler,
	},
	Route{
		"Listenvironment",
		"POST",
		"/listenvironment",
		tokenHandler,
	},
	Route{
		"Environmentcount",
		"POST",
		"/environmentcount",
		tokenHandler,
	},
	Route{
		"Addowner",
		"POST",
		"/addowner",
		tokenHandler,
	},
	Route{
		"Adddeveloper",
		"POST",
		"/adddeveloper",
		tokenHandler,
	},
	Route{
		"Addrunonly",
		"POST",
		"/addrunonly",
		tokenHandler,
	},
	Route{
		"Removeowner",
		"DELETE",
		"/removeowner",
		tokenHandler,
	},
	Route{
		"Removedeveloper",
		"DELETE",
		"/removedeveloper",
		tokenHandler,
	},
	Route{
		"Removerunonly",
		"DELETE",
		"/removerunonly",
		tokenHandler,
	},
}
