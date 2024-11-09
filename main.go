package main

import (
	"fmt"
	mw "home_api/src/middleware"
	"home_api/src/routes"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/cors"
)

type LogWriter struct {
	FileName string
}

func NewLogWriter(fileName string) *LogWriter {
	return &LogWriter{
		FileName: fileName,
	}
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	f, err := os.OpenFile(w.FileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(p)
}

type WebServer struct {
	Address   string
	UsingUDS  bool
	LogWriter *LogWriter
}

// NewWebServer - Create a new Webserver
func NewWebServer(address string, usingUDS bool) *WebServer {
	return &WebServer{
		Address:   address,
		UsingUDS:  usingUDS,
		LogWriter: NewLogWriter("./data/latest.log"),
	}
}

// ApplyRoutes - Apply the routes to the Webserver
func ApplyRoutes(mux *http.ServeMux) *http.ServeMux {
	routes.WoolCatalogue(mux)
	return mux
}

// Setup - Setup the Webserver
func (s *WebServer) Setup() http.Handler {
	routerStack := routes.CreateStack()

	middlewareStack := mw.CreateStack(
		mw.RequestLoggerMiddleware,
		cors.AllowAll().Handler,
	)

	router := routerStack(http.NewServeMux())
	router = ApplyRoutes(router)
	router.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	return middlewareStack(router)
}

// Run - Start the Webserver
func (s *WebServer) Run() error {
	server := http.Server{
		Addr:    s.Address,
		Handler: s.Setup(),
	}

	if s.UsingUDS {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			os.Remove(s.Address)
			os.Exit(1)
		}()

		if _, err := os.Stat(s.Address); err == nil {
			log.Printf("Removing existing socket file %s", s.Address)
			if err := os.Remove(s.Address); err != nil {
				return err
			}
		}

		socket, err := net.Listen("unix", s.Address)
		if err != nil {
			return err
		}
		log.Printf("WebServer listening on %s", s.Address)
		return server.Serve(socket)
	} else {
		log.Printf("WebServer listening on %s", s.Address)
		return server.ListenAndServe()
	}
}

func main() {
	server := NewWebServer("0.0.0.0:9080", false)
	log.SetOutput(server.LogWriter)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
