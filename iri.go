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

func (p *parser) ipchar() error {
	// iunreserved / pct-encoded / sub-delims / ":" / "@"
	preIndex := p.index
	if iErr := p.iunreserved(); iErr == nil {
		return nil
	}
	p.index = preIndex
	if pctErr := p.pctEncoded(); pctErr == nil {
		return nil
	}
	p.index = preIndex
	r := p.current()
	if isSubDelim(r) || r == ':' || r == '@' {
		p.next()
		return nil
	}
	return newIriError(p, "Invalid ipchar value")
}

func (p *parser) iquery() {
	for {
		preIndex := p.index
		if iErr := p.ipchar(); iErr == nil {
			continue
		}
		p.index = preIndex
		if iErr := p.iprivate(); iErr == nil {
			continue
		}
		p.index = preIndex
		r := p.current()
		if r == '/' || r == '?' {
			continue
		}
		return
	}
}

func (p *parser) ifragment() {
	for {
		preIndex := p.index
		if iErr := p.ipchar(); iErr == nil {
			continue
		}
		p.index = preIndex
		r := p.current()
		if r == '/' || r == '?' {
			continue
		}
		return
	}
}

func (p *parser) iunreserved() error {
	//ALPHA / DIGIT / "-" / "." / "_" / "~" / ucschar
	r := p.current()
	if isAlpha(r) || isDigit(r) || r == '-' || r == '.' || r == '_' || r == '~' {
		return nil
	}
	if uErr := p.ucschar(); uErr != nil {
		return uErr
	}
	return newIriError(p, "Invalid iunreserved value")
}

func (p *parser) ucschar() error {
	r := p.current()
	if (r >= 0xa0 && r <= 0xd7ff) ||
		(r >= 0x10000 && r <= 0x1fffd) ||
		(r >= 0x20000 && r <= 0x2fffd) ||
		(r >= 0x30000 && r <= 0x3fffd) ||
		(r >= 0x40000 && r <= 0x4fffd) ||
		(r >= 0x50000 && r <= 0x5fffd) ||
		(r >= 0x60000 && r <= 0x6fffd) ||
		(r >= 0x70000 && r <= 0x7fffd) ||
		(r >= 0x80000 && r <= 0x8fffd) ||
		(r >= 0x90000 && r <= 0x9fffd) ||
		(r >= 0xa0000 && r <= 0xafffd) ||
		(r >= 0xb0000 && r <= 0xbfffd) ||
		(r >= 0xc0000 && r <= 0xcfffd) ||
		(r >= 0xd0000 && r <= 0xdfffd) ||
		(r >= 0xe0000 && r <= 0xefffd) {
		return nil
	}
	return newIriError(p, fmt.Sprintf("Invalid ucschar value %c", p.current()))
}

func (p *parser) iprivate() error {
	r := p.current()
	if (r >= 0xe000 && r <= 0xf8ff) || (r >= 0xf0000 && r <= 0xffffd) || (r >= 0x100000 && r <= 0x10fff8) {
		return nil
	}
	return newIriError(p, fmt.Sprintf("Invalid iprivate value %c", p.current()))
}

func (p *parser) schema() error {
	if !isAlpha(p.current()) {
		return newIriError(p, "Schema must start with alpha")
	}
	p.next()
	for {
		c := p.current()
		if isAlpha(c) || isDigit(c) || c == '+' || c == '-' || c == '.' {
			if nErr := p.next(); nErr != nil {
				return nil
			}
			continue
		}
		break
	}
	return nil
}

func (p *parser) port() error {
	count := 0
	digits := make([]rune, 0)
	for {
		if isDigit(p.current()) {
			digits = append(digits, p.current())
			count++
			if pErr := p.next(); pErr == nil {
				continue
			}
		}
		break
	}
	if count == 0 {
		return newIriError(p, "No port")
	}
	if count > 5 {
		return newIriError(p, "Invalid port")
	}
	portStr := string(digits)
	port, _ := strconv.Atoi(portStr)
	if port > 65535 {
		return newIriError(p, "Invalid port value")
	}
	p.next()
	return nil
}

func (p *parser) ipLiteral() error {
	if p.current() != '[' {
		return newIriError(p, "Missing starting '[' ip literal")
	}
	p.next()
	preIndex := p.index
	if ipv6Err := p.ipv6Address(); ipv6Err == nil {
		return nil
	}
	p.index = preIndex
	if ipvfErr := p.ipvFuture(); ipvfErr != nil {
		return newIriError(p, "Invalid ipv6 or ipv future for ip literal")
	}
	p.next()
	if p.current() != ']' {
		return newIriError(p, "Missing ending ']' ip literal")
	}
	p.next()
	return nil
}

func (p *parser) ipvFuture() error {
	if p.current() != 'v' {
		return newIriError(p, "IpvFuture must start with 'v'")
	}
	h16Count := 0
	for {
		p.next()
		if h16Err := p.h16(); h16Err == nil {
			h16Count++
			continue
		}
		if h16Count < 1 {
			return newIriError(p, "Invalid IpvFuture")
		}
		break
	}
	if p.current() != '.' {
		return newIriError(p, "Invalid IpvFuture")
	}
	p.next()
	postCount := 0
	for {
		if isUnreserved(p.current()) || isSubDelim(p.current()) || p.current() == ':' {
			postCount++
			if pNext := p.next(); pNext == nil {
				continue
			}
		}
		if postCount < 1 {
			return newIriError(p, "Invalid IpvFuture")
		}
		break
	}
	return nil
}

func (p *parser) ipv6Address() error {
	groupCount := 0
	zeroCollaps := false
	prevColon := false

	for {
		if p.current() == ':' {
			if prevColon {
				groupCount++
				zeroCollaps = true
			} else if zeroCollaps {
				return newIriError(p, "Invalid zero collapse in ipv6")
			}
			prevColon = true
		} else {
			prevColon = false
			if pErr := p.h16(); pErr != nil {
				return newIriError(p, "Invalid hextet in ipv6")
			}
			groupCount++
		}
		if groupCount == 8 {
			break
		}
		if nErr := p.next(); nErr != nil {
			return nErr
		}
	}
	p.next()
	return nil
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
