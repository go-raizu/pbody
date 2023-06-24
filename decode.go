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

package pbody

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/elnormous/contenttype"

	"github.com/go-raizu/herr"
)

var (
	ErrMissingContentType   = errors.New("missing content-type")
	ErrUnsupportedMediaType = errors.New("unsupported media type")
)

type Codec struct {
	DetectFn func(mtype contenttype.MediaType) bool
	DecodeFn func(r io.Reader, mtype contenttype.MediaType, out any) error
}

type Decoder struct {
	codecsPtr atomic.Pointer[[]Codec]
}

func (d *Decoder) findCodec(mtype contenttype.MediaType) (out Codec, ok bool) {
	registry := d.codecsPtr.Load()
	if registry == nil {
		return
	}

	for i := range *registry {
		if !(*registry)[i].DetectFn(mtype) {
			continue
		}
		return (*registry)[i], true
	}
	return
}

// Register adds a given Codec to the Decoder.
func (d *Decoder) Register(codec Codec) {
	for {
		oldRegistry := d.codecsPtr.Load()

		var newRegistry []Codec
		if oldRegistry == nil {
			newRegistry = []Codec{codec}
		} else {
			newRegistry = append(*oldRegistry, codec)
		}

		if d.codecsPtr.CompareAndSwap(oldRegistry, &newRegistry) {
			break
		}
	}
}

// Decode detects the content from a given [http.Request] instance and
// decodes the stream presented in an [io.Reader] instance to the given
// out value.
//
// The length of the body can optionally be limited by passing a
// [http.MaxBytesReader] instance to the body argument.
//
// May return a [herr.HTTPError] if no content type was specified in
// the [http.Request] instance, the content type was not understood or
// the payload was too large.
// May return a [herr.HTTPError] or any other error if the decoder
// failed for other reasons.
func (d *Decoder) Decode(r *http.Request, body io.Reader, out any) error {
	ctHeaders := r.Header.Values("Content-Type")
	if len(ctHeaders) == 0 || ctHeaders[0] == "" {
		return errors.Join(herr.ErrBadRequest, ErrMissingContentType)
	}
	ctHeader := ctHeaders[0]

	mtype, err := contenttype.ParseMediaType(ctHeader)
	if err != nil {
		err := fmt.Errorf("%q: %w", ctHeader, err)
		return errors.Join(herr.ErrUnsupportedMediaType, err)
	}

	codec, ok := d.findCodec(mtype)
	if !ok {
		err := fmt.Errorf("%q: %w", mtype.String(), ErrUnsupportedMediaType)
		return errors.Join(herr.ErrUnsupportedMediaType, err)
	}

	if err := codec.DecodeFn(body, mtype, out); err != nil {
		if errors.Is(err, &http.MaxBytesError{}) {
			err = errors.Join(herr.ErrTooLarge, err)
		}
		return err
	}

	return nil
}

var Default Decoder

func Register(codec Codec) {
	Default.Register(codec)
}

func Decode(r *http.Request, body io.Reader, out any) error {
	return Default.Decode(r, body, out)
}
