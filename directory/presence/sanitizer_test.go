// Copyright 2020 Nym Technologies SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package presence

import (
	"github.com/microcosm-cc/bluemonday"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("Sanitizer", func() {
	Describe("sanitizing inputs", func() {
		Context("when XSS is present", func() {
			It("sanitizes input for string", func() {
				input := xssString()
				policy := bluemonday.UGCPolicy()
				sanitizer := NewSanitizer(policy)
				sanitizer.Sanitize(&input)
				assert.Equal(GinkgoT(), goodString(), input)
			})
			It("sanitizes input for struct", func() {
				type foomp struct {
					Foomper string
					Foo     uint
				}
				xssInput := foomp{
					xssString(), 42,
				}
				goodInput := foomp{
					goodString(), 42,
				}
				policy := bluemonday.UGCPolicy()
				sanitizer := NewSanitizer(policy)
				sanitizer.Sanitize(&xssInput)
				assert.Equal(GinkgoT(), goodInput, xssInput)
			})
			It("sanitizes input for nested struct", func() {
				type foomp struct {
					Foomper string
					Foo     uint
				}
				type bar struct {
					Foomp foomp
					Baz   string
					Bar   uint
				}

				xssInput := bar{
					Foomp: foomp{
						Foomper: xssString(),
						Foo:     42,
					},
					Baz: xssString(),
					Bar: 9001,
				}
				goodInput := bar{
					Foomp: foomp{
						Foomper: goodString(),
						Foo:     42,
					},
					Baz: goodString(),
					Bar: 9001,
				}

				policy := bluemonday.UGCPolicy()
				sanitizer := NewSanitizer(policy)
				sanitizer.Sanitize(&xssInput)
				assert.Equal(GinkgoT(), goodInput, xssInput)
			})
		})
	})
})

func xssString() string {
	return "foomp<script>alert('gotcha')</script>"
}

func goodString() string {
	return "foomp"
}
