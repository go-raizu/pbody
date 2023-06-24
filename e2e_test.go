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

package pbody_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/elnormous/contenttype"
	assertPkg "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	requirePkg "github.com/stretchr/testify/require"

	"github.com/go-raizu/herr"
	"github.com/go-raizu/pbody"
)

type mockCodec struct {
	mock.Mock
}

func (m *mockCodec) detect(mtype contenttype.MediaType) bool {
	return m.MethodCalled("Detect", mtype.Type, mtype.Subtype).Bool(0)
}

func (m *mockCodec) decode(r io.Reader, mtype contenttype.MediaType, out any) error {
	return m.MethodCalled("Decode", r, mtype, out).Error(0)
}

func Test(t *testing.T) {
	var d pbody.Decoder

	var m1 mockCodec
	m1.On("Detect", "application", "json").Return(true)
	m1.On("Detect", "application", "x-www-form-urlencoded").Return(false)
	m1.On("Detect", "application", "x-tar").Return(false)
	m1.On("Decode", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	d.Register(pbody.Codec{
		DetectFn: m1.detect,
		DecodeFn: m1.decode,
	})

	var m2 mockCodec
	m2.On("Detect", "application", "x-www-form-urlencoded").Return(true)
	m2.On("Decode", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m2.On("Detect", "application", "x-tar").Return(false)

	d.Register(pbody.Codec{
		DetectFn: m2.detect,
		DecodeFn: m2.decode,
	})

	type args struct {
		ctype string
	}
	tt := []struct {
		name    string
		args    args
		wantErr assertPkg.ErrorAssertionFunc
	}{
		{
			"accepted-json",
			args{"application/json"},
			assertPkg.NoError,
		},
		{
			"accepted-form-urlencoded",
			args{"application/x-www-form-urlencoded"},
			assertPkg.NoError,
		},

		{
			"missing-content-type",
			args{""},
			func(t assertPkg.TestingT, err error, i ...interface{}) bool {
				if !assertPkg.ErrorIs(t, err, herr.ErrBadRequest) {
					return false
				}

				return assertPkg.ErrorIs(t, err, pbody.ErrMissingContentType)
			},
		},
		{
			"unsupported-content-type",
			args{"application/x-tar"},
			func(t assertPkg.TestingT, err error, i ...interface{}) bool {
				if !assertPkg.ErrorIs(t, err, herr.ErrUnsupportedMediaType) {
					return false
				}

				return assertPkg.ErrorIs(t, err, pbody.ErrUnsupportedMediaType)
			},
		},
		{
			"invalid-input",
			args{"asd"},
			func(t assertPkg.TestingT, err error, i ...interface{}) bool {
				if !assertPkg.ErrorIs(t, err, herr.ErrUnsupportedMediaType) {
					return false
				}

				return assertPkg.ErrorIs(t, err, contenttype.ErrInvalidMediaType)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			requirePkg.NoError(t, err)
			if tc.args.ctype != "" {
				req.Header.Set("Content-Type", tc.args.ctype)
			}

			tc.wantErr(t, d.Decode(req, nil, nil), "Decode()")
		})
	}

	mock.AssertExpectationsForObjects(t, &m1, &m2)
}
