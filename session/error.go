package session

import (
	"errors"
)

var (
	ErrTokenArgs         error = errors.New("token args error")
	ErrTokenParseUnknow  error = errors.New("token parse unknown error")
	ErrTokenParseInvalid error = errors.New("token parse invalid")
	ErrSessionBan        error = errors.New("user was ban")
	ErrSessionFrozen     error = errors.New("user was frozen")
	ErrSessionReLogin    error = errors.New("user need relogin") // 用户需要重新登录一次
)
