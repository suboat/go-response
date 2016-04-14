package response

import (
	"net/http"
)

// 处理标准输入，返回标准输出
type LogicHandler func(req *RequestMode) (res *Response)

type RestHandler interface {
	//ServeHTTP(http.ResponseWriter, *http.Request)
	http.Handler
	ServeLogic(*RequestMode) *Response
}
