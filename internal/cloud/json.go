package cloud

import (
	"encoding/json"
	"errors"
	"strconv"
	"unicode/utf8"
)

// encoding/json otherwise silently replaces invalid UTF-8 and unpaired UTF-16
// escapes. Reject both before decoding so source bytes and locations stay honest.
func decodeResponse(body []byte, out any) error {
	invalid := errors.New("malformed JSON or Unicode in response")
	if !utf8.Valid(body) || !json.Valid(body) {
		return invalid
	}
	inString := false
	for i := 0; i < len(body); i++ {
		if body[i] == '"' {
			inString = !inString
		} else if inString && body[i] == '\\' {
			if body[i+1] != 'u' {
				i++ // Includes escaped quotes and backslashes.
				continue
			}
			value, _ := strconv.ParseUint(string(body[i+2:i+6]), 16, 16)
			if value >= 0xDC00 && value <= 0xDFFF {
				return invalid
			}
			if value >= 0xD800 && value <= 0xDBFF {
				if i+12 > len(body) || body[i+6] != '\\' || body[i+7] != 'u' {
					return invalid
				}
				low, err := strconv.ParseUint(string(body[i+8:i+12]), 16, 16)
				if err != nil || low < 0xDC00 || low > 0xDFFF {
					return invalid
				}
				i += 6 // Consume the low surrogate along with its high surrogate.
			}
			i += 5
		}
	}
	if json.Unmarshal(body, out) != nil {
		return invalid
	}
	return nil
}
