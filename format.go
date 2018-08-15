// Package jsonaux provides opinionated formatting and other auxiliary
// functionality related to JSON I/O.
package jsonaux

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// Format transforms the input using a comma-prefix style. The particular
// formatting should be considered opinionated and subject to change.
func Format(w io.Writer, r io.Reader) error {
	bw := bufio.NewWriter(w)
	dec := json.NewDecoder(r)
	dec.UseNumber()

	state := &state{Writer: bw, Decoder: dec, stack: make(stack, 0, 64)}
	err := state.any()
	if err != nil {
		return err
	}
	bw.WriteByte('\n')
	return bw.Flush()
}

type state struct {
	*bufio.Writer
	*json.Decoder
	min bool
	stack
}

func (s *state) any() error {
	t, err := s.Token()
	if err != nil {
		return err
	}
	d, ok := t.(json.Delim)
	if !ok {
		s.scalar(t)
		return nil
	}
	return s.composite(d)
}

func (s *state) composite(d json.Delim) (err error) {
	switch d {
	case '{':
		err = s.object()
	case '[':
		err = s.array()
	default:
		return fmt.Errorf("impossible state: %q", d)
	}
	if err == nil {
		// this will be '}' or ']'
		_, err = s.Token()
	}
	return err
}

func (s *state) object() error {
	s.push(object)
	defer s.pop()
	if s.next() == object {
		s.indent()
	}
	s.WriteByte('{')
	s.space()

	first := true
	for s.More() {
		if !first {
			s.comma()
		}
		err := s.string()
		if err != nil {
			return err
		}

		s.colon()
		err = s.any()
		if err != nil {
			return err
		}

		s.indent()
		first = false
	}
	return s.WriteByte('}')
}

func (s *state) array() error {
	s.push(array)
	defer s.pop()
	if s.next() == object {
		s.indent()
	}
	s.WriteByte('[')
	s.space()
	first := true
	for s.More() {
		if !first {
			s.comma()
		}
		err := s.any()
		if err != nil {
			return err
		}
		s.indent()
		first = false
	}
	return s.WriteByte(']')
}

func (s *state) string() error {
	t, err := s.Token()
	if err != nil {
		return err
	}
	s.scalar(t)
	return nil
}

func (s *state) scalar(t json.Token) {
	out, ok := t.(string)
	if ok {
		buf, _ := json.Marshal(out)
		s.Write(buf)
		return
	}
	switch t {
	case nil:
		out = "null"
	case true:
		out = "true"
	case false:
		out = "false"
	default:
		out = string(t.(json.Number))
	}
	s.WriteString(out)
}

func (s *state) comma() { s.punc(',') }
func (s *state) colon() { s.punc(':') }

func (s *state) punc(b byte) {
	s.WriteByte(b)
	s.space()
}

func (s *state) space() {
	if !s.min {
		s.WriteByte(' ')
	}
}

func (s *state) indent() {
	if !s.min {
		s.WriteByte('\n')
		n := s.depth()
		for i := 1; i < n; i++ {
			s.WriteString("  ")
		}
	}
}

type doctype uint8

const (
	none doctype = iota
	array
	object
)

type stack []doctype

func (s stack) get(i int) doctype {
	n := len(s)
	if i >= n {
		return none
	}
	return s[n-i-1]
}

func (s stack) depth() int      { return len(s) }
func (s stack) top() doctype    { return s.get(0) }
func (s stack) next() doctype   { return s.get(1) }
func (s *stack) push(t doctype) { *s = append(*s, t) }
func (s *stack) pop()           { *s = (*s)[:len(*s)-1] }
