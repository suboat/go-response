package response

import (
	"github.com/suboat/go-response/log"
	"github.com/suboat/go-response/session"

	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const (
	// header中自定义方法名
	RequestCrudMethodTag = "Manual-Method"

	// CRUD operations (create, read, update, delete)
	RequestCrudCreate  = "POST"
	RequestCrudRead    = "GET"
	RequestCrudQuery   = "QUERY"
	RequestCrudUpdate  = "PUT"
	RequestCrudDelete  = "DELETE"
	RequestCrudOptions = "OPTIONS"
)

var (
	// allow CORS config
	AllowCors     = true
	AllowCorsHost = []string{}
)

// 格式化后的标准请求
type Request struct {
	Method    string `maxLength:"255"`  // GET, POST, PUT, DELETE ...
	Url       string `maxLength:"512"`  // for websocket parser
	RequestId string `maxLength:"255"`  // for websocket callback, like jsonp
	Token     string `maxLength:"1024"` // for webscoket header auth

	// 一次性信息
	Password   string // 支付密码
	VerifyCode string // 验证码

	// For Get,Query
	Key   map[string]interface{} // search
	Sort  []string               // meta
	Skip  int                    // meta
	Limit int                    // meta

	// For Post, Put, Delete
	Data interface{}

	Session  *session.Session // 会话信息,含用户uid及会话级别
	RemoteIp string           // 请求ip,ban计数
}

// 后台解析请求方法
func SerializeHttp(rw http.ResponseWriter, req *http.Request) (que *Request, err error) {
	var (
		category string           // 类型: json, other
		se       *session.Session // session
	)
	que = new(Request)

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
		que.Session = se
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
func SerializeHttpWs(conn *ConnWs, msgType int, msg []byte) (que *Request, err error) {
	var (
		se *session.Session
	)

	// TODO: check msgType

	// TODO: 解析session信息
	que = new(Request)
	que.Session.Uid = conn.Uid

	if err = json.NewDecoder(bytes.NewReader(msg)).Decode(&que); err != nil {
		return
	}

	log.Debug("conn uid: ", conn.Uid)

	// 解析session信息
	if len(que.Token) > 0 {
		if que.Session, err = session.TokenToUid(que.Token); err != nil {
			return
		} else {
			que.Session = se
		}
	} else {
		// 匿名会话
		que.Session = new(session.Session)
	}

	// TODO: change/update ws hub
	if que.Session != nil && (conn.Uid != que.Session.Uid) {
		// 以req的uid为准，更新当前conn的用户指向
		log.Debug("conn.Uid=", conn.Uid, " que.Uid=", que.Session.Uid)
		if err = conn.UidUpdate(que.Session.Uid); err != nil {
			return
		}
	}

	return
}
