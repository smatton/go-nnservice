package handler

import (
	"os"

	"encoding/json"

	"github.com/smatton/go-nnservice/pkg/neighbors"
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

func IndexStats(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	ctx.SetStatusCode(200)
	ctx.WriteString(index.Hnsw.Stats())

}

func ShutDown(ctx *fasthttp.RequestCtx, shutdown chan<- os.Signal) {
	ctx.SetStatusCode(200)
	ctx.WriteString("Server Shutdown")
	close(shutdown)

}

func KNNSearch(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var sp searchPayload
	err := json.Unmarshal(ctx.PostBody(), &sp)
	if err != nil {
		ctx.Error("Search Json payload error", 500)
		// return failure to decode
		return
	}
	if sp.EfSearch != 0 {
		index.SetEf(sp.EfSearch)
	}

	stringLabels := make([]string, sp.K)
	labels, distances := index.Search(sp.Query, sp.K)
	for i, lab := range labels {
		stringLabels[i] = string(lab)
	}

	resp := searchResponse{Labels: stringLabels, Dists: distances}
	jsonBody, err := json.Marshal(resp)
	if err != nil {
		ctx.Error("Json Marhsall error", 500)
	}
	ctx.SetContentType("application/json; charset=utf-8")
	ctx.SetStatusCode(200)
	ctx.Response.SetBody(jsonBody)
	return

}

func Benchmark(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var sp searchPayload
	err := json.Unmarshal(ctx.PostBody(), &sp)
	if err != nil {
		ctx.Error("Search Json payload error", 500)
		// return failure to decode
		return
	}

	precision := index.Hnsw.Benchmark(sp.Query, sp.EfSearch, sp.K)

	resp := benchResponse{Precision: precision}
	jsonBody, err := json.Marshal(resp)
	if err != nil {
		ctx.Error("Json Marhsall error", 500)
	}
	ctx.SetContentType("application/json; charset=utf-8")
	ctx.SetStatusCode(200)
	ctx.Response.SetBody(jsonBody)
	return

}

func Insert(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var ip insertPayload
	err := json.Unmarshal(ctx.PostBody(), &ip)
	if err != nil {

		ctx.Error("Insert Json payload error", 500)
		// return failure to decode
		return
	}

	index.Insert(ip.Point, []byte(ip.Label))

	ctx.SetStatusCode(200)
	return

}

type insertPayload struct {
	Point []float32 `json:"point"`
	Label string    `json:"label"`
}

type searchPayload struct {
	Query    []float32 `json:"query"`
	EfSearch int       `json:"ef_search"`
	K        int       `json:"k"`
}

type searchResponse struct {
	Labels []string  `json:"labels"`
	Dists  []float32 `json:"distances"`
}

type benchResponse struct {
	Precision float64 `json:"precision"`
}
