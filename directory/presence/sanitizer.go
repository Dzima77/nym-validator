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
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"os"
	"reflect"
)

type Sanitizer interface {
	Sanitize(input interface{})
}

type sanitizer struct {
	policy *bluemonday.Policy
}

// NewSanitizer returns a new input sanitizer for all presence-related things
func NewSanitizer(policy *bluemonday.Policy) Sanitizer {
	return sanitizer{
		policy: policy,
	}
}


func (s sanitizer) Sanitize(input interface{}) {
	v := reflect.ValueOf(input)
	v = reflect.Indirect(v)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		kind := field.Kind()

		switch kind {
		case reflect.String:
			if !field.CanSet() {
				fmt.Printf("wtf can't set %v (type: %v)", field, kind)
				continue
			}
			field.SetString(s.policy.Sanitize(field.String()))
		case reflect.Struct:
			s.Sanitize(v.Field(i).Addr().Interface())
		case reflect.Int64:
		case reflect.Uint:
			continue
		default:
			// rather than ignoring everything like Uint above, let's do each single type
			// explicitly and separately for time being so that we wouldn't be confused
			// why, say, a map or slice doesn't work if we introduced them
			fmt.Fprintf(os.Stderr, "tried to sanitize unknown type %+v\n", kind)
		}
	}
}