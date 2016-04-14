package main

import (
	"github.com/suboat/go-response"
	"github.com/suboat/go-response/log"
	"github.com/suboat/go-response/mux"

	"net/http"
)

// 逻辑处理单元1
func handleRoot(req *response.Request) (res *response.Response) {
	res = response.NewResponse(req)
	res.Data = []string{"h", "e", "l", "l", "o"}
	return
}

// 逻辑处理单元2
func handleItem(req *response.Request) (res *response.Response) {
	res = response.NewResponse(req)
	res.Data = req.Key
	return
}

func main() {
	var (
		r       = mux.NewRouter().PathPrefix("/v1").Subrouter()
		err     error
		address = "127.0.0.1:8080"
	)

	// Handle方法同时绑定了restful路由与websocket虚拟路由
	// 但目前websocket虚拟路由只支持基础的url参数设置,如/item/{accession}
	r.Handle("/", response.NewSimpleRestHandler(handleRoot)).Methods("GET", "POST", "OPTIONS")
	r.Handle("/item/{accession}", response.NewSimpleRestHandler(handleItem)).Methods("GET", "OPTIONS")

	// 启动路由
	// websocket虚拟路由入口
	if err = r.ListenAndServeWs("/wshub/v1/", nil); err != nil {
		log.Panic(err)
	}
	log.Info("ListenAndServe: ", address, " wshub: /wshub/v1/")
	http.ListenAndServe(address, r)

}
