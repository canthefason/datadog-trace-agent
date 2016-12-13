package model

import (
	"encoding/json"
	"io"

	"github.com/ugorji/go/codec"
)

// Decoder is the common interface that all decoders should honor
type ClientDecoder interface {
	Decode(v interface{}) error
}

func DecoderFromContentType(contentType string, bodyBuffer io.Reader) ClientDecoder {
	// select the right Decoder based on the given content-type header
	switch contentType {
	case "application/msgpack":
		return codec.NewDecoder(bodyBuffer, &codec.MsgpackHandle{})
	default:
		// if the client doesn't use a specific decoder, fallback to JSON
		return json.NewDecoder(bodyBuffer)
	}
}