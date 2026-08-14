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
		"Getschedule",
		"POST",
		"/getschedule",
		tokenHandler,
	},
	Route{
		"Createschedule",
		"POST",
		"/createschedule",
		tokenHandler,
	},
	Route{
		"Updateschedule",
		"POST",
		"/updateschedule",
		tokenHandler,
	},
	Route{
		"Deleteschedule",
		"DELETE",
		"/deleteschedule",
		tokenHandler,
	},
	Route{
		"Listschedule",
		"POST",
		"/listschedule",
		tokenHandler,
	},
	Route{
		"Disableschedule",
		"POST",
		"/disableschedule",
		tokenHandler,
	},
	Route{
		"Enableschedule",
		"POST",
		"/enableschedule",
		tokenHandler,
	},
}
