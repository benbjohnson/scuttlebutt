// Copyright 2012 Arne Roomann-Kurrik
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

package json

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	STRING = iota
	NUMBER
	MAP
	ARRAY
	ENDARRAY
	ESCAPE
	BOOL
	NULL
)

func clamp(val int, min int, max int) int {
	if val > max {
		val = max
	}
	if val < min {
		val = min
	}
	return val
}

type Event struct {
	Type  int
	Index int
}

type State struct {
	data   []byte
	i      int
	v      interface{}
	events []Event
}

func (s *State) Read() (err error) {
	var t int = s.nextType()
	switch t {
	case STRING:
		err = s.readString()
	case NUMBER:
		err = s.readNumber()
	case MAP:
		err = s.readMap()
	case ARRAY:
		err = s.readArray()
	case ENDARRAY:
		s.i++
		err = EndArray{}
	case BOOL:
		err = s.readBool()
	case NULL:
		err = s.readNull()
	case ESCAPE:
		err = fmt.Errorf("JSON should not start with escape")
	default:
		length := len(s.data) - 1
		idx := clamp(s.i, 0, length)
		max := clamp(idx + 10, 0, length)
		min := clamp(idx - 10, 0, length)
		mid := clamp(s.i + 1, idx, length)
		b := string(s.data[min : idx])
		c := string(s.data[idx : mid])
		e := string(s.data[mid : max])
		err = fmt.Errorf("Unrecognized type in '%v -->%v<-- %v'", b, c, e)
	}
	return
}

func (s *State) nextType() int {
	for {
		if s.i >= len(s.data) {
			return -1
		}
		c := s.data[s.i]
		switch {
		case c == ' ':
			fallthrough
		case c == '\t':
			s.i++
			break
		case c == '"':
			return STRING
		case '0' <= c && c <= '9' || c == '-':
			return NUMBER
		case c == '[':
			return ARRAY
		case c == ']':
			return ENDARRAY
		case c == '{':
			return MAP
		case c == 't' || c == 'T' || c == 'f' || c == 'F':
			return BOOL
		case c == 'n':
			return NULL
		default:
			return -1
		}
	}
	return -1
}

func (s *State) readString() (err error) {
	var (
		c       byte
		start   int
		buf     *bytes.Buffer
		atstart bool = false
		more    bool = true
		escaped bool = false
		escape  bool = false
	)
	for atstart == false {
		c = s.data[s.i]
		switch {
		case c == ' ':
			fallthrough
		case c == '\t':
			s.i++
		case c == '"':
			atstart = true
			break
		case c == '}':
			s.i++
			return EndMap{}
		case c == ']':
			s.i++
			return EndArray{}
		default:
			return fmt.Errorf("Invalid string char: %v", c)
		}
	}
	s.i++
	start = s.i
	buf = new(bytes.Buffer)
	for more {
		c = s.data[s.i]
		switch {
		case escape == false && c == '\\':
			escape = true
			escaped = true
			break
		case escape == true && c == '\\':
			escape = false
			break
		case escape == true && c == '/':
			// Skip the backslash
			buf.Write(s.data[start:s.i-1])
			start = s.i
			escape = false
			break
		case c == '"':
			if escape == false {
				more = false
			} else {
				escape = false
			}
			break
		case s.i >= len(s.data)-1:
			return fmt.Errorf("No string terminator")
		default:
			escape = false
			break
		}
		s.i++
	}
	buf.Write(s.data[start : s.i-1])
	s.v = buf.String()
	if escaped == true {
		var utfstr = s.v.(string)
		utfstr = fmt.Sprintf("\"%v\"", utfstr)
		if s.v, err = strconv.Unquote(utfstr); err == nil {
			s.v = decodeSurrogates(s.v.(string))
		}
	}
	return
}

func (s *State) readNumber() (err error) {
	var c byte
	var val int64 = 0
	var valf float64 = 0
	var mult int64 = 1
	if s.data[s.i] == '-' {
		mult = -1
		s.i++
	}
	var more = true
	var places int = 0
	for more {
		c = s.data[s.i]
		switch {
		case '0' <= c && c <= '9':
			if places != 0 {
				places *= 10
			}
			val = val*10 + int64(c-'0')
		case '}' == c:
			err = EndMap{}
			more = false
		case ']' == c:
			err = EndArray{}
			more = false
		case ',' == c:
			s.i--
			more = false
		case ' ' == c || '\t' == c:
			more = false
		case '.' == c:
			valf = float64(val)
			val = 0
			places = 1
		default:
			return fmt.Errorf("Bad num char: %v", string([]byte{c}))
		}
		if s.i >= len(s.data)-1 {
			more = false
		}
		s.i++
	}
	if places > 0 {
		s.v = (valf + float64(val)/float64(places)) * float64(mult)
	} else {
		s.v = val * mult
	}
	return
}

// Decodes UTF-16 surrogate pairs (such as \uD834\uDD1E).
func decodeSurrogates(s string) string {
	var (
		r1  rune = 0
		buf      = new(bytes.Buffer)
	)
	for _, r := range s {
		if utf16.IsSurrogate(r) {
			if r1 == 0 {
				r1 = r
			} else {
				buf.WriteRune(utf16.DecodeRune(r1, r))
				r1 = 0
			}
		} else {
			if r1 != 0 {
				buf.WriteRune(r1)
				r1 = 0
			}
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

type EndMap struct{}

func (e EndMap) Error() string {
	return "End of map structure encountered."
}

type EndArray struct{}

func (e EndArray) Error() string {
	return "End of array structure encountered."
}

func (s *State) readComma() (err error) {
	var more = true
	for more {
		switch {
		case s.data[s.i] == ',':
			more = false
		case s.data[s.i] == '}':
			s.i++
			return EndMap{}
		case s.data[s.i] == ']':
			s.i++
			return EndArray{}
		case s.i >= len(s.data)-1:
			return fmt.Errorf("No comma")
		}
		s.i++
	}
	return nil
}

func (s *State) readColon() (err error) {
	var more = true
	for more {
		switch {
		case s.data[s.i] == ':':
			more = false
		case s.i >= len(s.data)-1:
			return fmt.Errorf("No colon")
		}
		s.i++
	}
	return nil
}

func (s *State) readMap() (err error) {
	s.i++
	var (
		m   map[string]interface{}
		key string
	)
	m = make(map[string]interface{})
	for {
		if err = s.readString(); err != nil {
			if _, ok := err.(EndMap); !ok {
				return
			}
			break
		}
		key = s.v.(string)
		if err = s.readColon(); err != nil {
			return
		}
		if err = s.Read(); err != nil {
			if _, ok := err.(EndMap); !ok {
				return
			}
		}
		m[key] = s.v
		if _, ok := err.(EndMap); ok {
			break
		}
		if err = s.readComma(); err != nil {
			if _, ok := err.(EndMap); ok {
				break
			}
			return
		}
	}
	s.v = m
	return nil
}

func (s *State) readArray() (err error) {
	s.i++
	var (
		a []interface{}
		c uint = 0
	)
	a = make([]interface{}, 0, 10)
	for {
		if err = s.Read(); err != nil {
			if _, ok := err.(EndArray); !ok {
				return
			}
			if c == 0 {
				break
			}
		}
		a = append(a, s.v)
		c++
		if _, ok := err.(EndArray); ok {
			break
		}
		if err = s.readComma(); err != nil {
			if _, ok := err.(EndArray); ok {
				break
			}
			return
		}
	}
	s.v = a
	return nil
}

func (s *State) readBool() (err error) {
	if strings.ToLower(string(s.data[s.i:s.i+4])) == "true" {
		s.i += 4
		s.v = true
	} else if strings.ToLower(string(s.data[s.i:s.i+5])) == "false" {
		s.i += 5
		s.v = false
	} else {
		err = fmt.Errorf("Could not parse boolean")
	}
	return
}

func (s *State) readNull() (err error) {
	if strings.ToLower(string(s.data[s.i:s.i+4])) == "null" {
		s.i += 4
		s.v = nil
	} else {
		err = fmt.Errorf("Could not parse null")
	}
	return
}

func Unmarshal(data []byte, v interface{}) error {
	state := &State{data, 0, v, make([]Event, 0, 10)}
	if err := state.Read(); err != nil {
		return err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("Need a pointer, got %v", reflect.TypeOf(v))
	}
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	sv := reflect.ValueOf(state.v)
	for sv.Kind() == reflect.Ptr {
		sv = sv.Elem()
	}
	var (
		rvt = rv.Type()
		svt = sv.Type()
	)
	if !svt.AssignableTo(rvt) {
		if rv.Kind() != reflect.Slice && sv.Kind() != reflect.Slice {
			return fmt.Errorf("Cannot assign %v to %v", svt, rvt)
		}
		if sv.Len() == 0 {
			return nil
		}
		var (
			mapi  map[string]interface{}
			mapt  = reflect.TypeOf(mapi)
			svte  = svt.Elem()
			rvte  = rvt.Elem()
			ismap bool
		)
		_, ismap = sv.Index(0).Interface().(map[string]interface{})
		if !(ismap && mapt.AssignableTo(rvte)) {
			return fmt.Errorf("Cannot assign %v to %v", svte, rvte)
		}
		var (
			ssv = reflect.MakeSlice(rvt, sv.Len(), sv.Cap())
		)
		for i := 0; i < sv.Len(); i++ {
			v := sv.Index(i).Interface().(map[string]interface{})
			ssv.Index(i).Set(reflect.ValueOf(v))
		}
		sv = ssv
	}
	rv.Set(sv)
	return nil
}
