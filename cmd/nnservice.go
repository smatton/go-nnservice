package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strconv"

	//"github.com/pkg/profile"

	"net/http"
	_ "net/http/pprof"

	"github.com/smatton/go-nnservice/pkg/neighbors"
	"github.com/smatton/go-nnservice/pkg/network"
	"github.com/smatton/go-nnservice/pkg/server"
	"github.com/smatton/go-nnservice/pkg/server/http/handler"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/pprofhandler"
)

var (
	PORT string
	CPU  int
)

func main() {

	flag.StringVar(&PORT, "port", "9023", "port to start registry on")

	flag.Parse()
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	NTHREADS := os.Getenv("NTHREADS")
	HNSWLIB_DIMS := os.Getenv("HNSWLIB_DIMS")
	INDEX_FILE := os.Getenv("INDEX_FILE")
	HNSWLIB_MAX_ELEMENTS := os.Getenv("HNSWLIB_MAX_ELEMENTS")
	HNSWLIB_M := os.Getenv("HNSWLIB_M")
	HNSWLIB_EF_CONSTRUCTION := os.Getenv("HNSWLIB_EF_CONSTRUCTION")
	HNSWLIB_NORMALIZED := os.Getenv("HNSWLIB_NORMALIZED")
	nthreads := 4
	if NTHREADS != "" {
		nthreads, _ = strconv.Atoi(NTHREADS)
	}

	DIMS := 128
	if HNSWLIB_DIMS != "" {
		DIMS, _ = strconv.Atoi(HNSWLIB_DIMS)
	}

	MAX_ELEMENTS := 10000
	if HNSWLIB_MAX_ELEMENTS != "" {
		MAX_ELEMENTS, _ = strconv.Atoi(HNSWLIB_MAX_ELEMENTS)

	}

	M := 32
	if HNSWLIB_M != "" {
		M, _ = strconv.Atoi(HNSWLIB_M)
	}

	HNSWLIB_EFC := 400

	if HNSWLIB_EF_CONSTRUCTION != "" {
		HNSWLIB_EFC, _ = strconv.Atoi(HNSWLIB_EF_CONSTRUCTION)
	}
	normalizedPoints := false
	if HNSWLIB_NORMALIZED != "" {
		normalizedPoints = true
	}

	newindex := neighbors.NewIndex(M, HNSWLIB_EFC, MAX_ELEMENTS, DIMS, normalizedPoints)
	if INDEX_FILE != "" {
		newindex.Load(INDEX_FILE)
		newindex.Hnsw.Grow(MAX_ELEMENTS)
	}

	logger.Println(newindex.Hnsw.Stats())

	myIP, err := network.GetOutboundIP()
	if err != nil {
		logger.Println("Couldn't determine hostname, starting on loopback 127.0.0.1")
		myIP = "127.0.0.1"
	}

	//numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(nthreads)

	myserver := server.New(myIP + ":" + PORT)
	//var upgrader = websocket.FastHTTPUpgrader{}
	// Add custom routes
	myserver.Router.POST("/hnsw/api/v1/search", func(ctx *fasthttp.RequestCtx) {
		handler.KNNSearch(ctx, newindex)
	})
	myserver.Router.POST("/hnsw/api/v1/insert", func(ctx *fasthttp.RequestCtx) {
		handler.Insert(ctx, newindex)
	})
	myserver.Router.GET("/hnsw/api/v1/stats", func(ctx *fasthttp.RequestCtx) {
		handler.IndexStats(ctx, newindex)
	})
	myserver.Router.POST("/hnsw/api/v1/benchmark", func(ctx *fasthttp.RequestCtx) {
		handler.Benchmark(ctx, newindex)
	})
	myserver.Router.ANY("/ws/hnsw/api/v1/search", func(ctx *fasthttp.RequestCtx) {
		handler.WsKNNSearch(ctx, newindex)
	})

	// PROFILER HANDLES
	myserver.Router.GET("/debug/pprof", pprofhandler.PprofHandler)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Start server
	myserver.Start()
}
