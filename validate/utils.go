package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Basic regular expressions for validating strings
const (
	regexEmail             string = "^(((([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|((\\x22)((((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(([\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(\\([\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(\\x22)))@((([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|\\.|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|\\.|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"
	regexAlpha             string = "^[a-zA-Z]+$"
	regexAlphanumeric      string = "^[a-zA-Z0-9]+$"
	regexAlphanumericSpace string = "^[a-zA-Z0-9\\s]+$"
	regexAlphaSpace        string = "^[\\pL\\s]+$"
	regexIP                string = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	regexURLSchema         string = `((ftp|tcp|udp|wss?|https?):\/\/)`
	regexURLUsername       string = `(\S+(:\S*)?@)`
	regexURLPath           string = `((\/|\?|#)[^\s]*)`
	regexURLPort           string = `(:(\d{1,5}))`
	regexURLIP             string = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	regexURLSubdomain      string = `((www\.)|([a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*))`
	regexURL                      = `^` + regexURLSchema + `?` + regexURLUsername + `?` + `((` + regexURLIP + `|(\[` + regexIP + `\])|(([a-zA-Z0-9]([a-zA-Z0-9-]+)?[a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*)|(` + regexURLSubdomain + `?))?(([a-zA-Z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-zA-Z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-zA-Z\x{00a1}-\x{ffff}]{1,}))?))` + regexURLPort + `?` + regexURLPath + `?$`
	regexLatitude          string = "^[-+]?([1-8]?\\d(\\.\\d+)?|90(\\.0+)?)$"
	regexLongitude         string = "^[-+]?(180(\\.0+)?|((1[0-7]\\d)|([1-9]?\\d))(\\.\\d+)?)$"
)

var (
	patternEmail             = regexp.MustCompile(regexEmail)
	patternAlpha             = regexp.MustCompile(regexAlpha)
	patternAlphanumeric      = regexp.MustCompile(regexAlphanumeric)
	patternAlphanumericSpace = regexp.MustCompile(regexAlphanumericSpace)
	patternAlphaSpace        = regexp.MustCompile(regexAlphaSpace)
	patternURL               = regexp.MustCompile(regexURL)
	patternLatitude          = regexp.MustCompile(regexLatitude)
	patternLongitude         = regexp.MustCompile(regexLongitude)
)

// toJSON convert the input to a valid JSON string
func toJSON(value interface{}) string {
	res, err := json.Marshal(value)
	if err != nil {
		res = []byte("")
	}
	return string(res)
}

// toString convert the input to a string.
func toString(value interface{}) string {
	res := fmt.Sprintf("%v", value)
	return string(res)
}

// toInt convert the input string to an integer, or 0 if the input is not an integer.
func toInt(value interface{}) int {
	res, err := strconv.Atoi(strings.Trim(toString(value), ""))
	if err != nil {
		res = 0
	}
	return res
}

// toUnderscore converts from camel case form to underscore separated form.
// Ex.: MyFunc => my_func
func toUnderscore(value interface{}) string {
	s := toString(value)
	var output []rune
	var segment []rune
	for _, r := range s {
		if !unicode.IsLower(r) {
			output = addSegment(output, segment)
			segment = nil
		}
		segment = append(segment, unicode.ToLower(r))
	}
	output = addSegment(output, segment)
	return string(output)
}

func addSegment(inrune, segment []rune) []rune {
	if len(segment) == 0 {
		return inrune
	}
	if len(inrune) != 0 {
		inrune = append(inrune, '_')
	}
	inrune = append(inrune, segment...)
	return inrune
}

// dataLength convert type of data to float64
// if the val is not numeric, the result is can be
// length of string, length of slice
func dataLength(val interface{}) (x float64) {
	v := reflect.ValueOf(val)
	// check if type of data is pointer
	if v.Kind() == reflect.Ptr {
		//v is value of *val
		v = v.Elem()
	}
	//switch base on type
	switch v.Kind() {
	case reflect.String:
		//count string length and change it to float64
		str := utf8.RuneCountInString(toString(v))
		x = float64(str)
		break
	case reflect.Slice:
		//length of slice to float64 (by Len() from lib value
		slc := v.Len()
		x = float64(slc)
		break
	case reflect.Float32:
		fl32 := val.(float32)
		x = float64(fl32)
		break
	case reflect.Float64:
		x = val.(float64)
		break
	default:
		num := toInt(v)
		x = float64(num)
	}
	return x
}

// removeLastDot removing string from the last dot
// ex: username.required to username
func removeLastDot(str string) string {
	if idx := strings.LastIndex(str, "."); idx != -1 {
		return str[:idx]
	}
	return str
}

func ValidPhone(text string) (p string, e error) {
	reg, _ := regexp.Compile("[^0-9]+")
	p = reg.ReplaceAllString(text, "")

	if len(p) <= 6 {
		e = errors.New("invalid format phone")
		return
	}

	prefix := string(p[0:2])
	if prefix == "08" {
		p = strings.Replace(p, "08", "628", 1)
	} else {
		prefix2 := string(p[0:1])
		if prefix2 == "8" {
			p = strings.Replace(p, "8", "628", 1)
		}
	}

	fp := string(p[0:2])
	if fp != "62" {
		e = errors.New("invalid format phone")
	}

	return
}
