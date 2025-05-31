package rest

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
)

// Response defines the standard structure for all HTTP responses
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}
