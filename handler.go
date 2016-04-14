package response

import (
	"net/http"
)

// 框架要求的逻辑处理单元:处理标准输入，返回标准输出
type LogicHandler func(req *Request) (res *Response)

// 框架要求的handler定义
type RestHandler interface {
	http.Handler                   //ServeHTTP(http.ResponseWriter, *http.Request)
	ServeLogic(*Request) *Response //
}
