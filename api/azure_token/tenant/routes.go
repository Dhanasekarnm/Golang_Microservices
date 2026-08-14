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
		"Gettenant",
		"POST",
		"/gettenant",
		tokenHandler,
	},
	Route{
		"Createtenant",
		"POST",
		"/createtenant",
		tokenHandler,
	},
	Route{
		"Updatetenant",
		"POST",
		"/updatetenant",
		tokenHandler,
	},
	Route{
		"Deletetenant",
		"DELETE",
		"/deletetenant",
		tokenHandler,
	},
	Route{
		"Listtenant",
		"POST",
		"/listtenant",
		tokenHandler,
	},
	Route{
		"Tenantcount",
		"POST",
		"/tenantcount",
		tokenHandler,
	},
	Route{
		"Addtenantonwer",
		"POST",
		"/addtenantowner",
		tokenHandler,
	},
	Route{
		"Addenvironmentmaker",
		"POST",
		"/addenvironmentmaker",
		tokenHandler,
	},
	Route{
		"Removetenantowner",
		"DELETE",
		"/removetenantowner",
		tokenHandler,
	},
	Route{
		"Removeenvironmentmaker",
		"DELETE",
		"/removeenvironmentmaker",
		tokenHandler,
	},
}
