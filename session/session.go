package session

import (
	"github.com/suboat/sorm"
	"github.com/suboat/sorm/log"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/sessions"
	"net/http"
	"strconv"
	"time"
)

var (
	// cookie
	SessionKey       = "jksijdnfhrwuxnfh"                          // 秘钥
	SessionStore     = sessions.NewCookieStore([]byte(SessionKey)) // session cookie
	SessionStoreName = "sessionid"                                 // session cookie的字段名称
	SessionTagUid    = "uid"                                       // uid
	SessionAuthTag   = "is_authenticated"                          // 判断是否登陆

	// jwt
	TokenTagHead    = "Authorization"    // http head中存放token
	TokenTagExp     = "exp"              //
	TokenTagKid     = "kid"              //
	TokenTagUid     = "uid"              //
	TokenTagLevel   = "level"            // 会话等级
	TokenKidUser    = "user"             //
	TokenKidAdmin   = "admin"            //
	TokenExpDefault = time.Hour * 72     // token默认有效期
	TokenSessionKey = []byte(SessionKey) // byte slice

	// config
	UserModel orm.Model // 用户数据库,由外部package赋值维护
)

const (
	SessionLevelNormal uint = 1 << iota // 用户授权
	SessionLevelSecure                  // 安全密码已输入
	SessionLevelPay                     // 支付级别,一次性使用
)

const (
	// user status
	UserStatusNormal    int = iota // 0 正常
	UserStatusUnActive             // 1 option: 未激活
	UserStatusFreeze               // 2 option: 冻结:可以登录,不能进入安全会话
	UserStatusBand                 // 3 option: 禁用:不能登录
	UserStatusLoginWarn            // 4 option: 登录异常
	UserStatusDeleting             // 5 option: 等待删除,清空数据
	UserStatusReLogin              // 6 option: 需要重新登录:如修改密码后
)

type Session struct {
	Uid    orm.Uid // uid
	Secure uint    // 安全级别
}

// 含有uid字段与某些字段的model: 只为对应数据库映射，取uid
// pg issue: missing destination name https://github.com/jmoiron/sqlx/issues/143
type userBase struct {
	Uid    orm.Uid // uid
	Status int     // 当前用户状态 正常,待激活,冻结,禁用
}

// 从http读Session
func HttpSessionUid(rw http.ResponseWriter, req *http.Request) (se *Session, err error) {
	se = new(Session)
	// 默认uid
	//uid = orm.GuestUid

	//var (
	//	ok   bool
	//	v    interface{}
	//	_uid orm.Uid
	//	s    *sessions.Session
	//)
	//// 从cookie取Uid github.com/gorilla/sessions
	//if s, err = SessionStore.Get(req, SessionStoreName); err != nil {
	//	println(err.Error())
	//	return
	//}
	////
	//if v, ok = s.Values[SessionTagUid]; ok == true {
	//	if _uid, ok = v.(orm.Uid); ok != true {
	//		return
	//	}
	//	if err = _uid.Valid(); err != nil {
	//		return
	//	}
	//	uid = _uid
	//}

	// 从headToken中取uid
	if tokenStr := req.Header.Get(TokenTagHead); len(tokenStr) > 0 {
		if se, err = TokenToUid(tokenStr); err != nil {
			log.Warn("tokenStr error: ", err, " ", tokenStr)
		}
	}

	return
}

// 解析token中的uid信息
// TODO:验证token的时效性,如密码已更改,用户被禁用
func TokenToUid(tokenStr string) (se *Session, err error) {
	se = new(Session)
	var (
		uid    orm.Uid
		ok     bool
		v      interface{}
		_uid   orm.Uid
		token  *jwt.Token
		_uid_s string
		//u      *userBase // 对应的用户
	)

	// 默认uid
	uid = orm.GuestUid

	if token, err = ParseTokenString(tokenStr); err != nil {
		return
	}

	// 取uid
	if v, ok = token.Claims[TokenTagUid]; ok == true {
		if _uid_s, ok = v.(string); ok != true {
			return
		}
		_uid = orm.Uid(_uid_s)
		if err = _uid.Valid(); err != nil {
			return
		}
		uid = _uid
	}

	// session level
	if v, ok = token.Claims[TokenTagLevel]; ok == true {
		if _v, _ok := v.(string); _ok == true {
			var i int
			if i, err = strconv.Atoi(_v); err != nil {
				return
			} else {
				se.Secure = uint(i)
			}
		}
	}

	//// 取用户验证
	//if uid != orm.GuestUid {
	//	if u, err = PubUserGetByUid(uid); err != nil {
	//		return
	//	}
	//	// 已禁用的用户
	//	if u.Status == UserStatusBand {
	//		err = ErrSessionBan
	//		return
	//	}
	//	// 需要重新登录
	//	if u.Status == UserStatusReLogin {
	//		err = ErrSessionReLogin
	//		return
	//	}
	//	// 已冻结
	//	if u.Status == UserStatusFreeze && (se.Secure&SessionLevelSecure > 0) {
	//		err = ErrSessionFrozen
	//		return
	//	}
	//}

	se.Uid = uid
	return
}

// map转tokenStr
func NewToken(kid string, m map[string]interface{}, tExp *time.Duration) (token string, err error) {
	if m == nil {
		err = ErrTokenArgs
		return
	}

	// exp time
	if tExp == nil {
		tExp = &TokenExpDefault
	}

	t := jwt.New(jwt.SigningMethodHS256)
	t.Header[TokenTagKid] = kid
	t.Claims = m
	t.Claims[TokenTagExp] = time.Now().Add(*tExp).Unix()

	token, err = t.SignedString(TokenSessionKey)
	return
}

// token解析
func ParseTokenString(tokenStr string) (token *jwt.Token, err error) {
	if token, err = jwt.Parse(tokenStr, lookupKey); err != nil {
		return
	} else if token == nil {
		err = ErrTokenParseUnknow
	}

	if token.Valid == false {
		err = ErrTokenParseInvalid
		return
	}

	return
}

// 解密钥匙
func lookupKey(token *jwt.Token) (inf interface{}, err error) {
	switch token.Header[TokenTagKid] {
	case TokenKidUser:
		inf = TokenSessionKey
		break
	case TokenKidAdmin:
		inf = TokenSessionKey
		break
	default:
		inf = TokenSessionKey
		break
	}
	return
}

func Init() {
	// TODO: read key file
}

func init() {
	Init()
	//println("sesseion level: ", SessionLevelNormal, SessionLevelSecure, SessionLevelPay)
}
