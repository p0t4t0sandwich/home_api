package routes

import (
	"github.com/a-h/templ"
	"home_api/src/api/modules/woolcatalogue"
	"home_api/src/web/components"
	"net/http"
)

type Router func(*http.ServeMux) *http.ServeMux

func CreateStack(routers ...Router) Router {
	return func(mux *http.ServeMux) *http.ServeMux {
		for _, router := range routers {
			mux = router(mux)
		}
		return mux
	}
}

func WoolCatalogue(mux *http.ServeMux) *http.ServeMux {
	store, err := woolcatalogue.Load()
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /api/v1/wool-catalogue/wool", woolcatalogue.GetWool(store))
	mux.Handle("POST /api/v1/wool-catalogue/wool", woolcatalogue.CreateWool(store))
	mux.Handle("PUT /api/v1/wool-catalogue/wool", woolcatalogue.UpdateWool(store))
	mux.Handle("DELETE /api/v1/wool-catalogue/wool", woolcatalogue.DeleteWool(store))

	mux.Handle("GET /api/v1/wool-catalogue/wools", woolcatalogue.GetWools(store))
	mux.Handle("GET /api/v1/wool-catalogue/html/wools", woolcatalogue.GetWoolsHTML(store))

	mux.Handle("GET /wool-catalogue", templ.Handler(components.WoolRoot()))
	return mux
}
