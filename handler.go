package response

import (
	"github.com/suboat/go-response/log"
	"net/http"
)

// 框架要求的逻辑处理单元:处理标准输入，返回标准输出
type LogicHandler func(req *Request) (res *Response)

// 框架要求的handler定义:
// 1.能够处理普通http请求
// 2.能够处理标准逻辑处理单元,for websocket
type RestHandler interface {
	http.Handler                   //ServeHTTP(http.ResponseWriter, *http.Request)
	ServeLogic(*Request) *Response //
}

// simple
// 实现一个公共的符合要求的RestHandler,供快捷调用
type SimpleRestHandler struct {
	LogicHandler *LogicHandler
}

// 处理逻辑单元
func (h *SimpleRestHandler) ServeLogic(req *Request) *Response {
	return (*h.LogicHandler)(req)
}

// 处理http
func (h *SimpleRestHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var (
		res = new(Response)
		que *Request
	)
	// 返回
	defer func() {
		CreateResponse(rw, req, res)
	}()

	// 转换成标准请求
	if que, res.Error = SerializeHttp(rw, req); res.Error != nil {
		log.Error("SerializeHttp: ", res.Error)
		return
	}

	res = h.ServeLogic(que)
	return
}

// 将一个基础逻辑处理单元封装成SimpleRestHandler
func NewSimpleRestHandler(inf interface{}) (h *SimpleRestHandler) {
	// go特性,以断言来处理两种格式
	if lg, ok := inf.(*LogicHandler); ok == true {
		h = &SimpleRestHandler{LogicHandler: lg}
		return
	} else if lg, ok := inf.(LogicHandler); ok == true {
		h = &SimpleRestHandler{LogicHandler: &lg}
		return
	} else if _lg, ok := inf.(*func(*Request) *Response); ok == true {
		lg = LogicHandler((*_lg))
		h = &SimpleRestHandler{LogicHandler: &lg}
		return
	} else if _lg, ok := inf.(func(*Request) *Response); ok == true {
		lg = LogicHandler(_lg)
		h = &SimpleRestHandler{LogicHandler: &lg}
		return
	}
	// error
	log.Panic("unknown LogicHandler: ", inf)
	return
}
