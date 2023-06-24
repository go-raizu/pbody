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

package json_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/elnormous/contenttype"
	"github.com/go-raizu/herr"
	"github.com/go-raizu/pbody/json"
	assertPkg "github.com/stretchr/testify/assert"
)

func errorAssert(msg string) assertPkg.ErrorAssertionFunc {
	return func(t assertPkg.TestingT, err error, msgAndArgs ...any) bool {
		if !assertPkg.ErrorIs(t, err, herr.ErrBadRequest, msgAndArgs...) {
			return false
		}

		return assertPkg.ErrorContains(t, err, msg, msgAndArgs...)
	}
}

func Test(t *testing.T) {
	var (
		empty  struct{}
		single struct{ A string }
	)

	type args struct {
		inp string
		out any
	}
	tt := []struct {
		name    string
		args    args
		wantErr assertPkg.ErrorAssertionFunc
	}{
		{"ok-emtpy", args{`{}`, &empty}, assertPkg.NoError},
		{"err-empty", args{``, &single}, errorAssert(`body must not be empty`)},
		{"err-multiple-bodies", args{`{}{}`, &empty}, errorAssert("multiple json objects")},
		{"err-bad-quote-marks", args{`{”A”: 1}`, &single}, errorAssert("malformed JSON (at position 2)")},
		{"err-bad-type", args{`{"A": 1}`, &single}, errorAssert(`malformed JSON (at position 7), bad value for "A"`)},
		{"err-unknown-field", args{`{"B": 23}`, &single}, errorAssert(`unknown field "B"`)},
		{"err-bad-content", args{`{"hello wor`, &empty}, errorAssert("malformed JSON")},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mt := contenttype.MediaType{Type: "application", Subtype: "json"}

			buf := strings.NewReader(tc.args.inp)

			tc.wantErr(t, json.Decode(buf, mt, tc.args.out), fmt.Sprintf("Decode(%v, _, _)", tc.args.inp))
		})
	}
}
