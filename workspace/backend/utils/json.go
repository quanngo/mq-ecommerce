package utils

import (
	"bytes"
	"encoding/json"

	jsonI "github.com/json-iterator/go"
)

var _json = jsonI.ConfigCompatibleWithStandardLibrary

func MarshalToString(v interface{}) (string, error) {
	return _json.MarshalToString(v)
}

func Marshal(v interface{}) ([]byte, error) {
	return _json.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return _json.Unmarshal(data, v)
}

func UnmarshalFromString(str string, v interface{}) error {
	return _json.UnmarshalFromString(str, v)
}

func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return _json.MarshalIndent(v, prefix, indent)
}

func MarshalToJson(v interface{}) string {
	s, ok := v.(string)
	if ok {
		return s
	}
	str, err := MarshalToString(v)
	if err != nil {
		return ""
	}
	return str
}

func MarshalNoEscape(v any) string {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) // tắt escape HTML
	enc.Encode(v)
	return string(bytes.TrimSpace(buf.Bytes()))
}
