package network

import (
	"encoding/base64"
	"encoding/json"
)

// Encode encodes the input in base64
// It can optionally zip the input before encoding
func Encode(obj interface{}) (result string, err error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return
	}
	result = base64.StdEncoding.EncodeToString(b)
	return
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) error {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}

	return nil
}
