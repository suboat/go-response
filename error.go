package response

import (
	"errors"
)

var (
	ErrRequestSupport     error = errors.New("Request unsupport")
	ErrRequestDataType    error = errors.New("Request data field type error")
	ErrUploadFileSize     error = errors.New("Upload File size error")
	ErrRequestRestMethod  error = errors.New("RESTful method error")
	ErrImageType          error = errors.New("Type of file is not image")
	ErrPermission         error = errors.New("error Permission")
	ErrSocketConnHubEmpty error = errors.New("ws-hub in socket conn struct is nil")
)
