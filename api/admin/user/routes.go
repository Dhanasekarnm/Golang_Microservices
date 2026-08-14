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
		"Getuser",
		"POST",
		"/getuser",
		GetUser,
	},
	Route{
		"Createuser",
		"POST",
		"/createuser",
		CreateUser,
	},
	Route{
		"Updateuser",
		"POST",
		"/updateuser",
		UpdateUser,
	},
	Route{
		"Deleteuser",
		"DELETE",
		"/deleteuser",
		DeleteUser,
	},
	Route{
		"Listuser",
		"POST",
		"/listuser",
		ListUser,
	},
	Route{
		"Checkuserrole",
		"POST",
		"/checkuserrole",
		CheckUserRole,
	},
}
