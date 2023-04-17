package odin_iri

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const digit = "0123456789"
const unreservedSymbols = "-._~"
const genDelims = ":/?#[]@"
const subDelims = "!$&'()*+,;="

type IriError struct {
	message string
	index   int
	char    rune
}

func (i IriError) Error() string {
	return fmt.Sprintf("index: %d, char: %c, message: %s", i.index, i.char, i.message)
}

func ParseIri(value string) (IRI, error) {
	p := newParser(value)
	return p.parse()
}

type IRI interface {
}

type iri struct {
}

type parser struct {
	runes  []rune
	index  int
	length int
}

func newParser(value string) *parser {
	return &parser{
		runes:  bytes.Runes([]byte(value)),
		index:  -1,
		length: len(value),
	}
}

func (p *parser) next() error {
	if p.index+1 == p.length {
		return newIriError(p, "incomplete IRI")
	}
	p.index++
	return nil
}

func (p *parser) current() rune {
	return p.runes[p.index]
}

func (p *parser) peek() (rune, error) {
	if p.index+1 == p.length {
		return rune(0), errors.New("end of rune set reached")
	}
	return p.runes[p.index+1], nil
}

func (p *parser) setIndex(index int) {
	p.index = index
}

func (p *parser) parse() (IRI, error) {
	panic("not implemented")
}

func (p *parser) ls32() error {
	preH16Index := p.index
	if ipv4Err := p.ipv4Address(); ipv4Err == nil {
		return nil
	}
	p.setIndex(preH16Index)
	if h16Err := p.h16(); h16Err != nil {
		return h16Err
	}
	if nErr := p.next(); nErr != nil {
		return nErr
	}
	if p.current() != ':' {
		return newIriError(p, "invalid ls32 value")
	}
	if nErr := p.next(); nErr != nil {
		return nErr
	}
	if h16Err := p.h16(); h16Err != nil {
		return h16Err
	}
	return nil
}

func (p *parser) h16() error {
	if !isHexDigit(p.current()) {
		return newIriError(p, "invalid h16 value")
	}
	hexCount := 1
	for hexCount < 4 {
		pv, pErr := p.peek()
		if pErr != nil {
			return nil
		}
		if isHexDigit(pv) {
			p.next()
			hexCount++
		} else {
			break
		}
	}
	return nil
}

func (p *parser) ipv4Address() error {
	octCount := 0
	for octCount < 4 {
		if oErr := p.decOctet(); oErr != nil {
			return oErr
		}
		octCount++
		if octCount == 4 {
			break
		}
		if nErr := p.next(); nErr != nil {
			return nErr
		}
		if p.current() != '.' {
			return newIriError(p, "invalid ipv4 address")
		}
		if nErr := p.next(); nErr != nil {
			return nErr
		}
	}
	return nil
}

func (p *parser) decOctet() error {
	octetRunes := make([]rune, 0)
	if !isDigit(p.current()) {
		return newIriError(p, "invalid decimal octet")
	}
	octetRunes = append(octetRunes, p.current())
	peek, peekErr := p.peek()
	if peekErr != nil || !isDigit(peek) {
		return nil
	}
	p.next()
	octetRunes = append(octetRunes, p.current())
	d, _ := strconv.Atoi(string(octetRunes))
	if d < 10 {
		return newIriError(p, "invalid octet value")
	}
	peek, peekErr = p.peek()
	if peekErr != nil || !isDigit(peek) {
		return nil
	}
	p.next()
	octetRunes = append(octetRunes, p.current())
	d, _ = strconv.Atoi(string(octetRunes))
	if d < 100 || d > 255 {
		return newIriError(p, "invalid octet value")
	}
	peek, peekErr = p.peek()
	if peekErr != nil {
		return nil
	}
	if isDigit(peek) {
		return newIriError(p, "invalid octet value")
	}
	return nil
}

func (p *parser) pctEncoded() error {
	if p.current() != '%' {
		return newIriError(p, "invalid pct encoding")
	}
	if nErr := p.next(); nErr != nil {
		return nErr
	}
	if !isHexDigit(p.current()) {
		return newIriError(p, "invalid pct encoding")
	}
	if nErr := p.next(); nErr != nil {
		return nErr
	}
	if !isHexDigit(p.current()) {
		return newIriError(p, "invalid pct encoding")
	}
	return nil
}

func newIriError(p *parser, message string) IriError {
	return IriError{
		index:   p.index,
		char:    p.current(),
		message: message,
	}
}

func isAlpha(r rune) bool {
	return strings.ContainsRune(alpha, r)
}

func isDigit(r rune) bool {
	return strings.ContainsRune(digit, r)
}

func isSubDelim(r rune) bool {
	return strings.ContainsRune(subDelims, r)
}

func isGenDelim(r rune) bool {
	return strings.ContainsRune(genDelims, r)
}

func isReserved(r rune) bool {
	return isSubDelim(r) || isGenDelim(r)
}

func isUnreserved(r rune) bool {
	return isAlpha(r) || isDigit(r) || strings.ContainsRune(unreservedSymbols, r)
}

func isHexDigit(r rune) bool {
	return strings.ContainsRune("abcdefABCDEF", r) || isDigit(r)
}
