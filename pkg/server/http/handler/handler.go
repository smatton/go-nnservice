package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fasthttp/websocket"
	"github.com/francoispqt/gojay"
	"github.com/smatton/go-nnservice/pkg/neighbors"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{}

//Alive returns to 200
func Alive(ctx *fasthttp.RequestCtx) {

	ctx.SetStatusCode(200)
}

//GracefullShutdown handler, sents interrupt signal on channel which triggers server
// shutdown
func GracefullShutdown(ctx *fasthttp.RequestCtx, quit chan<- os.Signal) {

	ctx.SetStatusCode(200)
	quit <- os.Signal(os.Interrupt)

}

//IndexStats handler returns the statistcs for hnsw index. Such as number of levels,
// elements and memory usage
func IndexStats(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	ctx.SetStatusCode(200)
	ctx.WriteString(index.Hnsw.Stats())

}

// ShutDown provides an endpoint to shutdown the server
func ShutDown(ctx *fasthttp.RequestCtx, shutdown chan<- os.Signal) {
	ctx.SetStatusCode(200)
	ctx.WriteString("Server Shutdown")
	close(shutdown)

}

// KNNSave calls the save method on the index attempting to save the index to file
// specified in the json request
func KNNSave(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var sp savePayload
	var rp saveResponse

	err := json.Unmarshal(ctx.PostBody(), &sp)
	if err != nil {
		ctx.Error("Save Json payload error", 500)
		// return failure to decode
		return
	}

	if sp.FileName == "" || sp.FileName == "/" {
		ctx.Error("file_name must be specified", 500)
		return
	}

	err = index.Save(sp.FileName)
	if err != nil {
		rp.Error = err.Error()
	} else {
		wd, _ := os.Getwd()

		rp.Message = fmt.Sprintf("Succesfully saved file to: %s", filepath.Join(wd, sp.FileName))
	}

	jsonBody, err := json.Marshal(rp)
	if err != nil {
		ctx.Error("Json Marhsall error", 500)
	}

	ctx.SetContentType("application/json; charset=utf-8")
	ctx.SetStatusCode(200)

	ctx.Response.SetBody(jsonBody)
	return
}

//KNNSearch handles POST request containing a json body. The client specifies
// number of neighbors "k", "efSearch" which controls the precision and a "query"
// point. Searching can be safely done conncurrently even while inserting.
func KNNSearch(ctx *fasthttp.RequestCtx, index *neighbors.Index) {

	var sp searchPayload
	var dist point
	var b bytes.Buffer

	dec := gojay.BorrowDecoder(bytes.NewReader(ctx.PostBody()))
	defer dec.Release()
	//err := json.Unmarshal(ctx.PostBody(), &sp)
	err := dec.Decode(&sp)
	if err != nil {
		ctx.Error("Search Json payload error", 500)
		// return failure to decode
		return
	}
	if sp.EfSearch != 0 {
		index.SetEf(sp.EfSearch)
	}

	stringLabels := make([]string, sp.K)
	labels, distances := index.Search(*sp.Query, sp.K)
	for i, lab := range labels {
		stringLabels[i] = string(lab)
	}
	//fmt.Println(distances)

	dist = distances

	resp := searchResponse{Labels: stringLabels, Dists: &dist}

	enc := gojay.BorrowEncoder(&b)
	err = enc.Encode(resp)
	defer enc.Release()
	//b, err := gojay.MarshalJSONObject(resp)
	if err != nil {
		ctx.Error("Response Json payload marshal error", 500)
	}

	//jsonBody, err := json.Marshal(resp)
	//if err != nil {
	//	ctx.Error("Json Marhsall error", 500)
	//}

	ctx.SetContentType("application/json; charset=utf-8")
	ctx.SetStatusCode(200)

	//ctx.Response.SetBody(jsonBody)
	ctx.Response.SetBody(b.Bytes())
	return

}

//Benchmark hanlder compares brute force search with ann search for a query point.
// this is used to gauge an appropriate efSearch specification for queries.
func Benchmark(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var sp searchPayload
	err := json.Unmarshal(ctx.PostBody(), &sp)
	if err != nil {
		ctx.Error("Search Json payload error", 500)
		// return failure to decode
		return
	}

	var q []float32
	q = *sp.Query
	precision := index.Hnsw.Benchmark(q, sp.EfSearch, sp.K)

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

//WsKNNSearch implements websocket connection for better performance while making
// consecutive knn searches.
func WsKNNSearch(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var sp searchPayload
	var resp searchResponse
	// Pre-allocate the slice to be large
	MaxLabels := 100
	stringLabels := make([]string, MaxLabels)
	var dist point
	var b bytes.Buffer

	err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				ctx.Error("read error", 500)
				break
			}

			dec := gojay.BorrowDecoder(bytes.NewReader(message))

			//err := json.Unmarshal(ctx.PostBody(), &sp)
			err = dec.Decode(&sp)
			dec.Release()
			if err != nil {
				ctx.Error("Search Json payload error", 500)
				// return failure to decode
				return
			}

			if sp.EfSearch != 0 {
				index.SetEf(sp.EfSearch)
			}

			stringLabels = stringLabels[:0]

			labels, distances := index.Search(*sp.Query, sp.K)
			for _, lab := range labels {
				stringLabels = append(stringLabels, string(lab))
			}

			dist = distances
			resp.Labels = stringLabels
			resp.Dists = &dist

			b.Reset()

			enc := gojay.BorrowEncoder(&b)
			err = enc.Encode(resp)
			enc.Release()
			if err != nil {
				log.Println("Websocket Error:", err.Error())
				log.Printf("Searched: %v, returned labels %v, distances %v", *sp.Query, stringLabels, dist)
				continue
			}
			// jsonBody, err := json.Marshal(resp)
			// if err != nil {

			// 	log.Println("write:", err)
			// 	log.Printf("Searched: %v, returned labels %v, distances %v", *sp.Query, stringLabels, dist)
			// 	break
			// }
			//fmt.Println(jsonBody)
			err = ws.WriteMessage(mt, b.Bytes())
		}
	})

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			log.Println(err)
		}
		return
	}

}

//Insert inserts a point and label into the index
func Insert(ctx *fasthttp.RequestCtx, index *neighbors.Index) {
	var ip insertPayload
	err := json.Unmarshal(ctx.PostBody(), &ip)
	if err != nil {

		ctx.Error("Insert Json payload error", 500)
		// return failure to decode
		return
	}

	index.Insert(*ip.Point, []byte(ip.Label))

	ctx.SetStatusCode(200)
	return

}

type saveResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type savePayload struct {
	FileName string `json:"file_name"`
}

type insertPayload struct {
	Point *point `json:"point"`
	Label string `json:"label"`
}

type searchPayload struct {
	Query    *point `json:"query"`
	EfSearch int    `json:"ef_search"`
	K        int    `json:"k"`
}

type searchResponse struct {
	Labels []string `json:"labels"`
	Dists  *point   `json:"distances"`
}

type benchResponse struct {
	Precision float64 `json:"precision"`
}

// The following are necessary to implement the gojay decoder interface for
// pooled decoder

func (sr searchResponse) MarshalJSONObject(enc *gojay.Encoder) {
	enc.SliceStringKey("labels", sr.Labels)
	enc.ArrayKey("distances", sr.Dists)
}

func (sr searchResponse) IsNil() bool {
	return &sr == nil
}

func (sp *searchPayload) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "query":
		query := make(point, 0)
		sp.Query = &query
		return dec.Array(&query)
	case "ef_search":
		return dec.Int(&sp.EfSearch)
	case "k":
		return dec.Int(&sp.K)
	}
	return nil
}

func (sp *searchPayload) NKeys() int {
	return 3
}

type point []float32

func (v *point) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var i float32
	if err := dec.Float32(&i); err != nil {
		return err
	}
	*v = append(*v, i)
	return nil
}

func (v *point) MarshalJSONArray(enc *gojay.Encoder) {
	for _, i := range *v {
		enc.Float32(i)
	}
}
func (v *point) IsNil() bool {
	return v == nil || len(*v) == 0
}
