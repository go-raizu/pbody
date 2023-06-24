// * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
// Copyright(c) 2022-2023 individual contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
// * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *

package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/elnormous/contenttype"
	"github.com/go-raizu/herr"
	"github.com/go-raizu/pbody"
)

func errf(format string, args ...any) error {
	return errors.Join(herr.ErrBadRequest, pbody.ErrBadContent, fmt.Errorf(format, args...))
}

func init() {
	pbody.Register(Codec)
}

func Detect(mtype contenttype.MediaType) bool {
	return mtype.Type == "application" && mtype.Subtype == "json"
}

func Decode(r io.Reader, _ contenttype.MediaType, out any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			return errf("malformed JSON (at position %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errf("malformed JSON")

		case errors.As(err, &unmarshalTypeError):
			return errf(
				"malformed JSON (at position %d), bad value for %q",
				unmarshalTypeError.Offset,
				unmarshalTypeError.Field,
			)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return errf("unknown field %s", fieldName)

		case errors.Is(err, io.EOF):
			return errf("body must not be empty")

		default:
			return errors.Join(herr.ErrBadRequest, err)
		}
	}

	// Call Decode again to make sure the request body only
	// contains a single JSON object. Decode will return io.EOF
	// if there was only one JSON object. So if we get anything else,
	// we know that there is additional data in the request body.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errf("multiple json objects")
	}

	return nil
}

var Codec = pbody.Codec{
	DetectFn: Detect,
	DecodeFn: Decode,
}
