package handler

import (
	"os"

	"github.com/valyala/fasthttp"
)

//Alive returns to 200
func Alive(ctx *fasthttp.RequestCtx) {

	ctx.SetStatusCode(200)
}

func GracefullShutdown(ctx *fasthttp.RequestCtx, quit chan<- os.Signal) {

	ctx.SetStatusCode(200)
	quit <- os.Signal(os.Interrupt)

}

func ShutDown(ctx *fasthttp.RequestCtx, shutdown chan<- os.Signal) {
	ctx.SetStatusCode(200)
	ctx.WriteString("Server Shutdown")
	close(shutdown)

}
