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
		"Getuser",
		"POST",
		"/getuser",
		tokenHandler,
	},
	Route{
		"Createuser",
		"POST",
		"/createuser",
		tokenHandler,
	},
	Route{
		"Updateuser",
		"POST",
		"/updateuser",
		tokenHandler,
	},
	Route{
		"Deleteuser",
		"DELETE",
		"/deleteuser",
		tokenHandler,
	},
	Route{
		"Listuser",
		"POST",
		"/listuser",
		tokenHandler,
	},
	Route{
		"Checkuserrole",
		"POST",
		"/checkuserrole",
		tokenHandler,
	},
}
