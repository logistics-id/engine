package rest

import "net/http"

// Message defines the allowed standard message values
type Message string

const (
	MsgSuccess Message = "success"
	MsgCreated Message = "resource created"
	MsgUpdated Message = "resource updated"
	MsgDeleted Message = "resource deleted"

	MsgInvalidJSON     Message = "invalid request body"
	MsgMissingField    Message = "missing required fields"
	MsgInvalidField    Message = "invalid field value"
	MsgValidationError Message = "validation failed"

	MsgUnauthorized       Message = "unauthorized"
	MsgForbidden          Message = "forbidden"
	MsgNotFound           Message = "resource not found"
	MsgConflict           Message = "conflict"
	MsgInternalError      Message = "internal server error"
	MsgServiceUnavailable Message = "service unavailable"
	MsgBadRequest         Message = "invalid request body. please check your input format"
	MsgNotAllowed         Message = "method not allowed"
)

type HTTPError struct {
	Code    int
	Message Message
}

func (e HTTPError) Error() string {
	return string(e.Message)
}

func BadRequest() HTTPError {
	return HTTPError{Code: http.StatusBadRequest, Message: MsgBadRequest}
}

func Unauthorized() HTTPError {
	return HTTPError{Code: http.StatusUnauthorized, Message: MsgUnauthorized}
}

func Forbidden() HTTPError {
	return HTTPError{Code: http.StatusForbidden, Message: MsgForbidden}
}

func InternalServer() HTTPError {
	return HTTPError{Code: http.StatusInternalServerError, Message: MsgInternalError}
}

func NotFound() HTTPError {
	return HTTPError{Code: http.StatusNotFound, Message: MsgNotFound}
}

func NotAllowed() HTTPError {
	return HTTPError{Code: http.StatusMethodNotAllowed, Message: MsgNotAllowed}
}
