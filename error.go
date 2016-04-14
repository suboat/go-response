package response

import (
	"errors"
)

var (
	ErrRequestSupport     error = errors.New("Request unsupport")                   // sometext
	ErrRequestDataType    error = errors.New("Request data field type error")       // sometext
	ErrUploadFileSize     error = errors.New("Upload File size error")              // sometext
	ErrRequestRestMethod  error = errors.New("RESTful method error")                // sometext
	ErrImageType          error = errors.New("Type of file is not image")           // sometext
	ErrPermission         error = errors.New("error Permission")                    // sometext
	ErrSocketConnHubEmpty error = errors.New("ws-hub in socket conn struct is nil") // sometext
)
