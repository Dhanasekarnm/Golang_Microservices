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
		"Gettenant",
		"POST",
		"/gettenant",
		GetTenant,
	},
	Route{
		"Createtenant",
		"POST",
		"/createtenant",
		CreateTenant,
	},
	Route{
		"Updatetenant",
		"POST",
		"/updatetenant",
		UpdateTenant,
	},
	Route{
		"Deletetenant",
		"DELETE",
		"/deletetenant",
		DeleteTenant,
	},
	Route{
		"Listtenant",
		"POST",
		"/listtenant",
		ListTenant,
	},
	Route{
		"Tenantcount",
		"POST",
		"/tenantcount",
		TenantCount,
	},
	Route{
		"Addtenantonwer",
		"POST",
		"/addtenantowner",
		AddTenantOwner,
	},
	Route{
		"Addenvironmentmaker",
		"POST",
		"/addenvironmentmaker",
		AddEnvironmentMaker,
	},
	Route{
		"Removetenantowner",
		"DELETE",
		"/removetenantowner",
		RemoveTenantOwner,
	},
	Route{
		"Removeenvironmentmaker",
		"DELETE",
		"/removeenvironmentmaker",
		RemoveEnvironmentMaker,
	},
}
