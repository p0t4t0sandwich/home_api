package routes

import (
	"home_api/src/api/modules/photodump"
	"home_api/src/database"
	"home_api/src/web/components"
	"net/http"

	"github.com/a-h/templ"
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

// ApplyRoutes - Apply the routes to the Webserver
func ApplyRoutes(mux *http.ServeMux) *http.ServeMux {
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	Home(mux)
	PhotoDump(mux)
	WoolCatalogue(mux)
	return mux
}

func Home(mux *http.ServeMux) *http.ServeMux {
	mux.Handle("/", templ.Handler(components.HomeRoot()))
	return mux
}

func PhotoDump(mux *http.ServeMux) *http.ServeMux {
	db := database.GetDB("home")
	s3 := database.GetS3()
	ps := photodump.NewStore(db, s3)
	s := photodump.NewService(ps)

	mux.Handle("GET /photo-dump", templ.Handler(components.PhotoDumpRoot()))

	mux.Handle("GET /api/v1/photo-dump/photo", photodump.GetPhoto(s))
	mux.Handle("POST /api/v1/photo-dump/photo", photodump.UploadPhoto(s))
	mux.Handle("PUT /api/v1/photo-dump/photo", photodump.UpdatePhoto(s))
	mux.Handle("DELETE /api/v1/photo-dump/photo", photodump.DeletePhoto(s))
	mux.Handle("GET /api/v1/photo-dump/photos", photodump.GetPhotos(s))
	return mux
}

func WoolCatalogue(mux *http.ServeMux) *http.ServeMux {
	// store, err := woolcatalogue.Load()
	// if err != nil {
	// 	panic(err)
	// }
	// mux.Handle("GET /wool-catalogue", templ.Handler(components.WoolRoot()))
	//
	// mux.Handle("GET /api/v1/wool-catalogue/wool", woolcatalogue.GetWool(store))
	// mux.Handle("POST /api/v1/wool-catalogue/wool", woolcatalogue.CreateWool(store))
	// mux.Handle("PUT /api/v1/wool-catalogue/wool", woolcatalogue.UpdateWool(store))
	// mux.Handle("DELETE /api/v1/wool-catalogue/wool", woolcatalogue.DeleteWool(store))
	// mux.Handle("GET /api/v1/wool-catalogue/wools", woolcatalogue.GetWools(store))

	return mux
}
