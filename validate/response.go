package validate

import (
	"regexp"
)

// Response format when running validations
type Response struct {
	Valid           bool              // state of validation
	HeaderMessage   string            // header message in json
	errorMessages   map[string]string // compiled error message
	failureMessages map[string]string // failure error message
	customMessages  map[string]string // custom messages
	failureKeys     []string
}

// GetError returns failure message by key provided as parameter,
func (res *Response) GetError(k string) string {
	return res.errorMessages[k]
}

// SetError set an failure message as key and value
func (res *Response) SetError(k string, e string) {
	if res.failureMessages == nil {
		res.failureMessages = make(map[string]string)
	}

	res.Valid = false
	res.failureMessages[k] = e

	res.compile()
}

// GetMessages return all error messages.
func (res *Response) GetMessages() map[string]string {
	return res.errorMessages
}

// GetFailures return all failure message from validations.
func (res *Response) GetFailures() map[string]string {
	return res.failureMessages
}

func (res *Response) compile() *Response {
	res.errorMessages = make(map[string]string)
	for k, v := range res.failureMessages {
		ke := removeLastDot(k)
		if _, ok := res.errorMessages[ke]; !ok {
			res.errorMessages[ke] = v
		}
	}

	return res
}

func (res *Response) applyCustomMessage() {
	for i := range res.failureMessages {
		if c := res.customMessages[i]; c != "" {
			res.SetError(i, c)
			continue
		}

		if IsMatches(i, "(\\.[0-9]+\\.[a-z]+\\.[a-z]*)$") {
			re := regexp.MustCompile("[^a-z.]")
			ix := re.ReplaceAllString(i, "*")
			if c := res.customMessages[ix]; c != "" {
				res.SetError(i, c)
			}
		}
	}
}

func (res *Response) Error() string {
	vr := &validatorResponse{
		Message: res.HeaderMessage,
		Error:   res.errorMessages,
	}

	if vr.Message == "" {
		vr.Message = "Your input is invalid"
	}

	return toJSON(vr)
}
