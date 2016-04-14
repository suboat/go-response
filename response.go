package response

import (
	"github.com/suboat/go-response/log"
	"github.com/suboat/go-response/session"

	//"github.com/gorilla/websocket"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	RequestCrudMethodTag = "Manual-Method" // header中自定义方法名
	// CRUD operations (create, read, update, delete)
	RequestCrudCreate  = "POST"
	RequestCrudRead    = "GET"
	RequestCrudQuery   = "QUERY"
	RequestCrudUpdate  = "PUT"
	RequestCrudDelete  = "DELETE"
	RequestCrudOptions = "OPTIONS"
)

var (
	AllowCors     = true
	AllowCorsHost = []string{}
)

// 摘要信息
type Meta struct {
	Skip   int `json:"skip"`
	Limit  int `json:"limit"`
	Total  int `json:"total"`
	Length int `json:"length"`
}

// 后台接收格式 todo: 解决rest问题
type RequestMode struct {
	Url       string `maxLength:"512"`  // for websocket
	RequestId string `maxLength:"255"`  // for websocket callback
	Token     string `maxLength:"1024"` // for webscoket header auth
	Method    string `maxLength:"255"`  // GET, POST, PUT, DELETE

	// secure
	Password   string // 支付密码
	VerifyCode string // 验证码

	//Key   map[string]interface{} // search
	Key   map[string]interface{} // search
	Sort  []string               // meta
	Skip  int                    // meta
	Limit int                    // meta
	Data  interface{}            // For Post, Put, Delete

	QueryLimit  uint   // 查询限制
	Uid         string // TODO: session, 识别用户信息
	SecureLevel uint   // 回话安全级别
	RemoteIp    string // 请求ip
}

// 后台返回格式
type Response struct {
	Meta          *Meta          `json:"meta"`                // 摘要
	Data          interface{}    `json:"data"`                // 数据
	Error         error          `json:"-"`                   // 错误信息
	ErrorStr      string         `json:"error,omitempty"`     // error格式无法输出, 需明确为字符串
	Success       bool           `json:"success"`             // 如果error为空, success为true
	RequestId     string         `json:"requestId,omitempty"` // for websocket callback
	MessageWsPack *MessageWsPack `json:"-"`                   // for ws: 如果是websocket接口，push消息
	Uid           string         `json:"-"`                   // for ws: Logic handler 处理完后要改变当前会话uid, 为空则不改变
}

func (r *Response) ToJson() (s string) {
	if b, err := json.Marshal(r); err == nil {
		s = string(b)
	}
	return
}

// 后台解析请求方法
func SerializeHttp(rw http.ResponseWriter, req *http.Request) (que *RequestMode, err error) {
	var (
		category string           // 类型: json, other
		se       *session.Session // session
	)
	que = new(RequestMode)

	//CORS
	if AllowCors == true {
		origin := req.Header.Get("Origin")

		if len(origin) > 0 {
			rw.Header().Add("Access-Control-Allow-Origin", origin)
			rw.Header().Set("Access-Control-Allow-Credentials", "true")
			rw.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		}

		// options
		if req.Method == "OPTIONS" {
			//rw.Header().Add("Access-Control-Allow-Methods", "ACL, CANCELUPLOAD, CHECKIN, CHECKOUT, COPY, DELETE, GET, HEAD, LOCK, MKCALENDAR, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PROPPATCH, PUT, REPORT, SEARCH, UNCHECKOUT, UNLOCK, UPDATE, VERSION-CONTROL")
			rw.Header().Add("Access-Control-Allow-Methods", "DELETE, GET, HEAD, OPTIONS, POST, PUT, QUERY, UNLOCK, UPDATE")
		}
	}

	que.Token = req.Header.Get(session.TokenTagHead)
	// TODO:识别UID
	if se, err = session.HttpSessionUid(rw, req); err != nil {
		return
	} else {
		que.Uid = se.Uid
		que.SecureLevel = se.Secure
	}

	// 现只支持json
	category = "json"
	log.Debug("http: ", req.URL.String(), " ", req.Method)

	switch category {
	case "json":
		if req.ContentLength > 0 {
			err = json.NewDecoder(req.Body).Decode(&que)
		}
		break
	default:
		err = ErrRequestSupport
		break
	}

	if err != nil {
		return
	}

	// url
	que.Url = req.URL.String()

	// 从url读参数
	if v := req.FormValue("skip"); len(v) > 0 {
		if que.Skip, err = strconv.Atoi(v); err != nil {
			return
		}
	}
	if v := req.FormValue("sort"); len(v) > 0 {
		que.Sort = strings.Split(v, ",")
	}
	if v := req.FormValue("limit"); len(v) > 0 {
		if que.Limit, err = strconv.Atoi(v); err != nil {
			return
		}
	}
	if que.Limit == 0 {
		que.Limit = 10
	}

	// method: header first
	que.Method = req.Header.Get(RequestCrudMethodTag)
	if len(que.Method) == 0 {
		que.Method = req.Method
	}
	que.Url = req.URL.String()

	return
}

// 后台解析请求方法: ws
func SerializeHttpWs(conn *ConnWs, msgType int, msg []byte) (que *RequestMode, err error) {
	var (
		se *session.Session
	)
	// TODO: check msgType

	que = new(RequestMode)
	que.Uid = conn.Uid

	if err = json.NewDecoder(bytes.NewReader(msg)).Decode(&que); err != nil {
		return
	}

	log.Debug("conn uid: ", conn.Uid)
	if len(que.Token) > 0 {
		if se, err = session.TokenToUid(que.Token); err != nil {
			return
		} else {
			que.Uid = se.Uid
			que.SecureLevel = se.Secure
		}
	} else {
		que.Uid = session.GuestUid
	}
	// TODO: change/update ws hub
	if conn.Uid != que.Uid {
		// 以req的uid为准，更新当前conn的用户指向
		log.Debug("conn.Uid=", conn.Uid, " que.Uid=", que.Uid)
		if err = conn.UidUpdate(que.Uid); err != nil {
			return
		}
	}
	log.Debug("debug conn guest: ", len(conn.Hub.ConnWss[session.GuestUid]),
		"now:", conn.Uid, len(conn.Hub.ConnWss[conn.Uid]))

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

func NewResponse(req *RequestMode) (res *Response) {
	res = new(Response)
	if req != nil {
		res.RequestId = req.RequestId
	}
	return
}

func init() {
	//go HubWsSet.run()

	// ws CORS
	if AllowCors && WsUpgrader.CheckOrigin == nil {
		WsUpgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}
}
