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
		"Getbot",
		"POST",
		"/getbot",
		tokenHandler,
	},
	Route{
		"Createbot",
		"POST",
		"/createbot",
		tokenHandler,
	},
	Route{
		"Updatebot",
		"POST",
		"/updatebot",
		tokenHandler,
	},
	Route{
		"Deletebot",
		"DELETE",
		"/deletebot",
		tokenHandler,
	},
	Route{
		"Deletebotpermanent",
		"DELETE",
		"/deletebotpermanent",
		tokenHandler,
	},
	Route{
		"Listbot",
		"POST",
		"/listbot",
		tokenHandler,
	},
	Route{
		"Enablebot",
		"POST",
		"/enablebot",
		tokenHandler,
	},
	Route{
		"Disablebot",
		"POST",
		"/disablebot",
		tokenHandler,
	},
}
