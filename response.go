package response

import (
	"github.com/suboat/go-response/log"

	"encoding/json"
	"fmt"
	"net/http"
)

// 摘要信息
type Meta struct {
	Skip   int `json:"skip"`
	Limit  int `json:"limit"`
	Total  int `json:"total"`
	Length int `json:"length"`
}

// 后台返回格式
type Response struct {
	// status
	Success       bool           `json:"success"`             // 如果error为空, success为true
	RequestId     string         `json:"requestId,omitempty"` // for websocket callback
	MessageWsPack *MessageWsPack `json:"-"`                   // for ws: 如果是websocket接口，push消息

	// search meta and data list
	Meta *Meta       `json:"meta"` // 摘要
	Data interface{} `json:"data"` // 数据

	// error
	Error    error  `json:"-"`               // 错误信息
	ErrorStr string `json:"error,omitempty"` // error格式无法输出, 需明确为字符串

	// websocket
	Uid string `json:"-"` // for ws: Logic handler 处理完后要改变当前会话uid, 为空则不改变
}

func (r *Response) ToJson() (s string) {
	if b, err := json.Marshal(r); err == nil {
		s = string(b)
	}
	return
}

// 后台处理返回
func CreateResponse(rw http.ResponseWriter, req *http.Request, d *Response) {
	var category string

	// TODO: 自动检测要返回的类型
	category = "json"

	switch category {
	case "json":
		if d.Error != nil {
			d.ErrorStr = d.Error.Error() // bug?, error 无法输出
		} else {
			d.Success = true
		}
		fmt.Fprint(rw, d.ToJson())
	default:
		fmt.Fprint(rw, d.ToJson())
	}

	// TODO: 更详细的log
	if d.Error != nil {
		//log.Println("error:", req.RequestURI, d.Error.Error())
		log.Error(req.RequestURI, d.Error.Error())
	}
	return
}

// 后台处理返回: websocket
func CreateResponseWs(conn *ConnWs, d *Response) {
	var category string

	// TODO: 自动检测要返回的类型
	category = "json"

	switch category {
	case "json":
		if d.Error != nil {
			d.ErrorStr = d.Error.Error() // bug?, error 无法输出
		} else {
			d.Success = true
		}
		conn.SendText <- d.ToJson()
	default:
		conn.SendText <- d.ToJson()
	}

	// TODO: 更详细的log
	if d.Error != nil {
		log.Error(d.Error.Error())
	}

}

// 初始化一个response(带回调id)
func NewResponse(req *Request) (res *Response) {
	res = new(Response)
	if req != nil {
		res.RequestId = req.RequestId
	}
	return
}

func init() {
	//// debug
	//go HubWsSet.run()

	// ws CORS
	if AllowCors && WsUpgrader.CheckOrigin == nil {
		WsUpgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}
}
