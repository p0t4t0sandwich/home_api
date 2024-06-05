package routes

import "net/http"

type Router func(*http.ServeMux) *http.ServeMux

func CreateStack(routers ...Router) Router {
	return func(mux *http.ServeMux) *http.ServeMux {
		for _, router := range routers {
			mux = router(mux)
		}
		return mux
	}
}
