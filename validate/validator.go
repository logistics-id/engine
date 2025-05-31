package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type (
	// Validator holding the tag name and taglists available
	Validator struct {
		TagName      string
		ValidatorFns map[string]validatorFn
	}

	validatorTag struct {
		Name  string
		Param string
		Fn    validatorFn
	}

	validatorResponse struct {
		Message string            `json:"message"`
		Error   map[string]string `json:"error"`
	}

	// Request interface validation requests
	Request interface {
		Validate() *Response
		Messages() map[string]string
	}
)

func (v *Validator) fetchTag(tag string) (vt []validatorTag, e error) {
	if tag == "-" {
		e = errors.New("tag skipped")
		return
	}

	tl := strings.Split(tag, "|")
	vt = make([]validatorTag, 0, len(tl))

	for _, i := range tl {
		t := validatorTag{}
		p := strings.SplitN(i, ":", 2)

		if t.Name = strings.Trim(p[0], " "); t.Name == "" {
			e = errors.New("tag validation cannot be empty")
			break
		}

		if len(p) > 1 {
			t.Param = strings.Trim(p[1], " ")
		}

		var found bool
		if t.Fn, found = v.ValidatorFns[t.Name]; !found {
			e = fmt.Errorf("cannot find any tag function with name %s", t.Name)
			break
		}

		vt = append(vt, t)
	}

	return

}

// Field validates a value based on the provided
// tags and returns validator response
func (v *Validator) Field(value interface{}, tag string) (res *Response) {
	res = NewResponse()

	tags, err := v.fetchTag(tag)
	if err != nil {
		return
	}

	var e string
	for _, t := range tags {
		if res.Valid, e = t.Fn(value, t.Param); !res.Valid {
			res.SetError(t.Name, e)
			break
		}
	}

	return
}

// Struct validates the object of a struct based
// on 'valid' tags and returns errors found indexed
// by the field name.
func (v *Validator) Struct(object interface{}) (res *Response) {
	iVal := reflect.ValueOf(object)
	iType := reflect.TypeOf(object)

	// when object is pointer,
	// we should run validation for the real struct
	if iVal.Kind() == reflect.Ptr && !iVal.IsNil() {
		return v.Struct(iVal.Elem().Interface())
	}

	// the interface is not struct
	if iVal.Kind() != reflect.Struct && iVal.Kind() != reflect.Interface {
		return &Response{}
	}

	res = NewResponse()

	nf := iVal.NumField()
	for i := 0; i < nf; i++ {
		field := iVal.Field(i)
		fType := iType.Field(i)

		fname := fType.Tag.Get("json")
		if fname == "" {
			fname = toUnderscore(fType.Name)
		}

		fTag := fType.Tag.Get(v.TagName)
		if fTag == "" || fTag == "-" {
			continue
		}

		if field.Type() != reflect.TypeOf(time.Time{}) {
			if isPointer(field) || isStruct(field) {
				if r, ok := v.validRequest(field.Interface()); ok && !r.Valid {
					mergeResponse(fname, r, res)

					continue
				}

				if r := v.Struct(field.Interface()); !r.Valid {
					mergeResponse(fname, r, res)
				}

				continue
			}

			if isSlice(field) {
				for i := 0; i < field.Len(); i++ {
					if isPointer(field.Index(i)) || isStruct(field.Index(i)) {
						if r, ok := v.validRequest(field.Interface()); ok && !r.Valid {
							mergeResponse(fmt.Sprintf("%s.%d", fname, i), r, res)

							continue
						}
						if r := v.Struct(field.Index(i).Interface()); !r.Valid {
							mergeResponse(fmt.Sprintf("%s.%d", fname, i), r, res)
						}
					}
				}

				continue
			}
		}

		// run the validation for struct field
		if r := v.Field(field.Interface(), fTag); !r.Valid {
			mergeResponse(fname, r, res)
		}
	}
	return
}

// Request same as Validation.Struct but, this
// should be implement an ValidationRequest interfaces
// so we can do some custom validation and custome error messages.
func (v *Validator) Request(object Request) (res *Response) {
	res = &Response{
		Valid:          true,
		customMessages: object.Messages(),
	}

	// run as struct validation
	if os := v.Struct(object); !os.Valid {
		for k, e := range os.GetFailures() {
			res.SetError(k, e)
		}
	}

	// run custom validation
	if or := object.Validate(); or != nil && !or.Valid {
		for x, y := range or.GetFailures() {
			res.SetError(x, y)
		}
	}

	res.applyCustomMessage()
	res.compile()

	return
}

func (v *Validator) validRequest(object interface{}) (r *Response, valid bool) {
	if oq, ok := object.(Request); ok {
		valid = true
		r = v.Request(oq)
	}

	return
}

func mergeResponse(name string, cr *Response, pr *Response) {
	cr.compile()

	for k, e := range cr.GetFailures() {
		if IsContains(e, "%s") {
			e = fmt.Sprintf(e, strings.Replace(name, "_", " ", -1))
		}

		pr.SetError(name+"."+k, e)
	}
}

func isPointer(f reflect.Value) bool {
	return f.Kind() == reflect.Ptr && !f.IsNil()
}

func isStruct(f reflect.Value) bool {
	return (f.Kind() == reflect.Struct || f.Kind() == reflect.Interface) && f.Type() != reflect.TypeOf(time.Time{})
}

func isSlice(f reflect.Value) bool {
	if f.Kind() == reflect.Slice && f.Len() > 0 {
		return isStruct(f.Index(0)) || isPointer(f.Index(0))
	}

	return false
}

var tagsFn = map[string]validatorFn{
	"required":        validRequired,
	"numeric":         validNumeric,
	"alpha":           validAlpha,
	"alpha_num":       validAlphaNum,
	"alpha_num_space": validAlphaNumSpace,
	"alpha_space":     validAlphaSpace,
	"email":           validEmail,
	"latitude":        validLatitude,
	"longitude":       validLongitude,
	"url":             validURL,
	"json":            validJSON,
	"lte":             validLte,
	"gte":             validGte,
	"lt":              validLt,
	"gt":              validGt,
	"range":           validRange,
	"contains":        validContains,
	"match":           validMatch,
	"same":            validSame,
	"in":              validIn,
	"not_in":          validNotIn,
}

// New creates a new Validation instances.
func New() *Validator {
	return &Validator{
		TagName:      "valid",
		ValidatorFns: tagsFn,
	}
}

// NewResponse create new instance responses
func NewResponse() *Response {
	return &Response{Valid: true}
}
