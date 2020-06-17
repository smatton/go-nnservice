package main

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/smatton/go-nnservice/pkg/network"
	"github.com/smatton/go-nnservice/pkg/server"
)

var (
	PORT string
	CPU  int
)

func main() {

	flag.StringVar(&PORT, "port", "9023", "port to start registry on")
	flag.Parse()
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	myIP, err := network.GetOutboundIP()
	if err != nil {
		logger.Println("Couldn't determine hostname, starting on loopback 127.0.0.1")
		myIP = "127.0.0.1"
	}

	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	myserver := server.New(myIP + ":" + PORT)

	myserver.Start()
}
