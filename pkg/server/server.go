package server

import (
	"log"
	//"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/buaazp/fasthttprouter"
	"github.com/smatton/go-nnservice/pkg/server/http/handler"
	"github.com/valyala/fasthttp"
)

type Config struct {
	Address string
	Exit    chan os.Signal
	Done    chan bool
	Logger  *log.Logger
	Server  *fasthttp.Server
	Router  *fasthttprouter.Router
}

func New(address string) *Config {
	var cfg Config
	// Initialize logger
	cfg.Logger = log.New(os.Stdout, "INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	done := make(chan bool, 1)
	exit := make(chan os.Signal, 1)
	cfg.Done = done
	cfg.Exit = exit
	cfg.Address = address

	// Create new simple server
	cfg.Server, cfg.Router = NewSimpleServer(cfg.Logger)

	// minimally add the alive handle

	cfg.Router.GET("/alive", func(ctx *fasthttp.RequestCtx) {
		handler.Alive(ctx)
	})
	cfg.Router.GET("/shutdown", func(ctx *fasthttp.RequestCtx) {
		handler.ShutDown(ctx, cfg.Exit)
	})

	return &cfg
}

func (cfg *Config) Start() error {

	// start Gracefull shutdown thread
	signal.Notify(cfg.Exit, os.Interrupt)
	go GracefullShutdown(cfg.Server, cfg.Logger, cfg.Exit, cfg.Done)
	cfg.Logger.Println("Listening on: ", cfg.Address)

	if err := cfg.Server.ListenAndServe(cfg.Address); err != nil {
		cfg.Logger.Fatalf("Could not listen on %s: %v\n", ":"+cfg.Address, err)
		return err
	}

	<-cfg.Done
	return nil
}

func GracefullShutdown(server *fasthttp.Server, logger *log.Logger, quit <-chan os.Signal, done chan<- bool) {
	<-quit
	logger.Println("Server is shutting down...")

	if err := server.Shutdown(); err != nil {
		logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	close(done)
}

func NewSimpleServer(logger *log.Logger) (*fasthttp.Server, *fasthttprouter.Router) {
	router := fasthttprouter.New()
	server := &fasthttp.Server{
		Handler:      router.Handler,
		Logger:       logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return server, router
}
