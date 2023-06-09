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

var EOIError = errors.New("end of input")

// IriError specifies an error that occurred during the parsing of the input value.
type IriError struct {
	message string
	index   int
	char    rune
}

func (i IriError) Error() string {
	return fmt.Sprintf("index: %d, char: %c, message: %s", i.index, i.char, i.message)
}

// ParseIri attempts to parse a value into the IRI struct.
func ParseIri(value string) (*IRI, error) {
	p := newParser(value)
	p.next()
	return p.parse()
}

// IRI struct containing the parsed and validated value and each of its individual parts.
type IRI struct {
	// The raw value of this IRI
	Value     string
	Scheme    string
	Authority string
	Path      string
	Query     string
	Fragment  string
}

type parser struct {
	runes    []rune
	index    int
	length   int
	instance IRI
}

func newParser(value string) *parser {
	return &parser{
		runes:    bytes.Runes([]byte(value)),
		index:    -1,
		length:   len(value),
		instance: IRI{},
	}
}

func (p *parser) next() bool {
	if p.index-1 < p.length {
		p.index++
		return true
	}
	return false
}

func (p *parser) current() (rune, error) {
	if p.index >= p.length {
		return 0, EOIError
	}
	return p.runes[p.index], nil
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

func (p *parser) parse() (*IRI, error) {
	if iErr := p.iri(); iErr != nil {
		return nil, iErr
	}
	p.instance.Value = string(p.runes)
	return &p.instance, nil
}

func (p *parser) iri() error {
	if err := p.schema(); err != nil {
		return err
	}
	r, _ := p.current()
	if r != ':' {
		return newIriError(p, "iri missing ':' after schema")
	}
	if !p.next() {
		return EOIError
	}
	if err := p.ihierPart(); err != nil {
		if err == EOIError {
			return nil
		}
		return err
	}
	r, _ = p.current()
	if r == '?' {
		if !p.next() {
			return EOIError
		}
		p.iquery()
		return nil
	} else if r == '#' {
		if !p.next() {
			return EOIError
		}
		p.ifragment()
		return nil
	}
	return nil
}

func (p *parser) ihierPart() error {
	preIndex := p.index
	r, _ := p.current()
	pr, prErr := p.peek()
	if prErr == nil {
		if r == '/' && pr == '/' {
			p.next()
			if !p.next() {
				return EOIError
			}
			if authErr := p.iauthority(); authErr == nil {
				if eErr := p.ipathAbEmpty(); eErr == nil {
					return nil
				}
			}
		}
	}
	p.index = preIndex
	if err := p.ipathAbsolute(); err == nil {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	p.index = preIndex
	if err := p.ipathRootless(); err == nil {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	p.index = preIndex
	if err := p.ipathEmpty(); err != nil {
		return newIriError(p, "Invalid irelative-part value")
	}
	return nil
}

func (p *parser) iriReference() error {
	preIndex := p.index
	if err := p.iri(); err == nil {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	p.index = preIndex
	if err := p.irelativePart(); err == nil {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	return newIriError(p, "Invalid iri-reference value")
}

func (p *parser) absoluteIri() error {
	if err := p.schema(); err != nil {
		return err
	}
	r, _ := p.current()
	if r != ':' {
		return newIriError(p, "absolute-iri missing ':'")
	}
	if !p.next() {
		return EOIError
	}
	if err := p.ihierPart(); err != nil {
		return err
	}
	r, _ = p.current()
	if r == '?' {
		if p.next() {
			p.iquery()
		}
	}
	r, _ = p.current()
	if r == '#' {
		if !p.next() {
			return EOIError
		}
		p.ifragment()
	}
	return nil
}

func (p *parser) irelativePart() error {
	preIndex := p.index
	r, _ := p.current()
	pr, prErr := p.peek()
	if prErr == nil {
		if r == '/' && pr == '/' {
			p.next()
			if !p.next() {
				return EOIError
			}
			if authErr := p.iauthority(); authErr == nil {
				if eErr := p.ipathAbEmpty(); eErr == nil {
					if !p.next() {
						return EOIError
					}
					return nil
				}
			}
		}
	}
	p.index = preIndex
	if err := p.ipathAbsolute(); err == nil {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	p.index = preIndex
	if err := p.ipathNoSchema(); err == nil {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	p.index = preIndex
	if err := p.ipathEmpty(); err != nil {
		return newIriError(p, "Invalid irelative-part value")
	}
	return nil
}

func (p *parser) iauthority() error {
	authStart := p.index
	preIndex := p.index
	if err := p.iuserInfo(); err == nil {
		r, _ := p.current()
		if r == '@' {
			if !p.next() {
				return EOIError
			}
		} else {
			p.index = preIndex
		}
	} else {
		p.index = preIndex
	}
	p.ihost()
	r, _ := p.current()
	preIndex = p.index
	if r == ':' {
		if err := p.port(); err != nil {
			p.index = preIndex
		}
	}
	authRunes := p.runes[authStart:p.index]
	p.instance.Authority = string(authRunes)
	return nil
}

func (p *parser) iuserInfo() error {
	for {
		preIndex := p.index
		if err := p.iunreserved(); err == nil {
			if !p.next() {
				return EOIError
			}
			continue
		}
		p.index = preIndex
		if err := p.pctEncoded(); err == nil {
			if !p.next() {
				return EOIError
			}
			continue
		}
		p.index = preIndex
		r, _ := p.current()
		if isSubDelim(r) {
			if !p.next() {
				return EOIError
			}
			continue
		}
		if r == ':' {
			if !p.next() {
				return EOIError
			}
			continue
		}
		return nil
	}
}

func (p *parser) ihost() {
	preIndex := p.index
	if err := p.ipLiteral(); err == nil {
		return
	}
	p.index = preIndex
	if err := p.ipv4Address(); err == nil {
		return
	}
	p.index = preIndex
	p.iregName()
}

func (p *parser) iregName() {
	for {
		preIndex := p.index
		if err := p.iunreserved(); err != nil {
			p.index = preIndex
			if err = p.pctEncoded(); err != nil {
				p.index = preIndex
				r, _ := p.current()
				if !isSubDelim(r) {
					return
				}
			}
		}
		if !p.next() {
			return
		}
	}
}

func (p *parser) ipath() error {
	pathStart := p.index
	preIndex := p.index
	if err := p.ipathAbEmpty(); err != nil {
		p.index = preIndex
		if err := p.ipathAbsolute(); err != nil {
			p.index = preIndex
			if err := p.ipathNoSchema(); err != nil {
				p.index = preIndex
				if err := p.ipathRootless(); err != nil {
					p.index = preIndex
					if err := p.ipathEmpty(); err != nil {
						// Technically, the grammar shouldn't allow for this...
						return newIriError(p, "Invalid ipath")
					}
				}
			}
		}
	}
	p.instance.Path = string(p.runes[pathStart:p.index])
	return nil
}

func (p *parser) ipathAbEmpty() error {
	startIndex := p.index
	for {
		r, cErr := p.current()
		if cErr != nil {
			break
		}
		if r != '/' {
			break
		}
		if !p.next() {
			return EOIError
		}
		i := p.index
		p.isegment()
		if i == p.index {
			break
		}
	}
	p.instance.Path = string(p.runes[startIndex:p.index])
	return nil
}

func (p *parser) ipathAbsolute() error {
	startIndex := p.index
	r, _ := p.current()
	if r != '/' {
		return newIriError(p, "ipath-absolute must start with '/'")
	}
	if !p.next() {
		return EOIError
	}
	preIndex := p.index
	if err := p.isegmentNz(); err != nil {
		p.index = preIndex
	} else {
		for {
			r, _ = p.current()
			if r != '/' {
				break
			}
			if !p.next() {
				return EOIError
			}
			i := p.index
			p.isegment()
			if i == p.index {
				break
			}
		}
	}
	p.instance.Path = string(p.runes[startIndex:p.index])
	if !p.next() {
		return EOIError
	}
	return nil
}

func (p *parser) ipathNoSchema() error {
	startIndex := p.index
	if err := p.isegmentNzNc(); err != nil {
		return err
	}
	for {
		r, _ := p.current()
		if r != '/' {
			break
		}
		if !p.next() {
			return EOIError
		}
		i := p.index
		p.isegment()
		if i == p.index {
			break
		}
	}
	p.instance.Path = string(p.runes[startIndex:p.index])
	if !p.next() {
		return EOIError
	}
	return nil
}

func (p *parser) ipathRootless() error {
	preIndex := p.index
	if nzErr := p.isegmentNz(); nzErr != nil {
		p.index = preIndex
		return nzErr
	}
	if !p.next() {
		return EOIError
	}
	preIndex = p.index
	for {
		r, rErr := p.current()
		if rErr != nil {
			p.index = preIndex
			return rErr
		}
		if r != '/' {
			p.index = preIndex
			return newIriError(p, "Invalid ipath-rootless value")
		}
		p.isegment()
		if !p.next() {
			return nil
		}
	}
}

func (p *parser) ipathEmpty() error {
	if _, cErr := p.current(); cErr != nil {
		return EOIError
	}
	return nil
}

func (p *parser) isegment() {
	for {
		preIndex := p.index
		if iErr := p.ipchar(); iErr == nil {
			continue
		}
		p.index = preIndex
		return
	}
}

func (p *parser) isegmentNz() error {
	i := 0
	preIndex := p.index
	for {
		if iErr := p.ipchar(); iErr == nil {
			i++
			preIndex = p.index
			continue
		}
		break
	}
	p.index = preIndex
	if i < 1 {
		return newIriError(p, "Invalid isegment-nz value")
	}
	return nil
}

func (p *parser) isegmentNzNc() error {
	i := 0
	for {
		preIndex := p.index
		if iuErr := p.iunreserved(); iuErr == nil {
			i++
			continue
		}
		p.index = preIndex
		if pctErr := p.pctEncoded(); pctErr == nil {
			i++
			continue
		}
		p.index = preIndex
		r, _ := p.current()
		if isSubDelim(r) || r == '@' {
			i++
			continue
		}
		if i < 1 {
			return newIriError(p, "Invalid isegment-nz-nc")
		}
		return nil
	}
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
	r, _ := p.current()
	if isSubDelim(r) || r == ':' || r == '@' {
		p.next()
		return nil
	}
	return newIriError(p, "Invalid ipchar value")
}

func (p *parser) iquery() {
	startIndex := p.index
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
		r, _ := p.current()
		if r == '/' || r == '?' {
			if !p.next() {
				break
			}
			continue
		}
		p.instance.Query = string(p.runes[startIndex:p.index])
		return
	}
}

func (p *parser) ifragment() {
	startIndex := p.index
	for {
		preIndex := p.index
		if iErr := p.ipchar(); iErr == nil {
			continue
		}
		p.index = preIndex
		r, _ := p.current()
		if r == '/' || r == '?' {
			continue
		}
		p.instance.Fragment = string(p.runes[startIndex:p.index])
		return
	}
}

func (p *parser) iunreserved() error {
	//ALPHA / DIGIT / "-" / "." / "_" / "~" / ucschar
	r, _ := p.current()
	if isAlpha(r) || isDigit(r) || r == '-' || r == '.' || r == '_' || r == '~' {
		if !p.next() {
			return EOIError
		}
		return nil
	}
	if uErr := p.ucschar(); uErr != nil {
		return uErr
	}
	return newIriError(p, "Invalid iunreserved value")
}

func (p *parser) ucschar() error {
	r, _ := p.current()
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
	return newIriError(p, fmt.Sprintf("Invalid ucschar value %c", r))
}

func (p *parser) iprivate() error {
	r, _ := p.current()
	if (r >= 0xe000 && r <= 0xf8ff) || (r >= 0xf0000 && r <= 0xffffd) || (r >= 0x100000 && r <= 0x10fff8) {
		return nil
	}
	return newIriError(p, fmt.Sprintf("Invalid iprivate value %c", r))
}

func (p *parser) schema() error {
	schemaRunes := make([]rune, 0)
	r, _ := p.current()
	if !isAlpha(r) {
		return newIriError(p, "Scheme must start with alpha")
	}
	schemaRunes = append(schemaRunes, r)
	if !p.next() {
		return EOIError
	}
	for {
		r, _ = p.current()
		if isAlpha(r) || isDigit(r) || r == '+' || r == '-' || r == '.' {
			schemaRunes = append(schemaRunes, r)
			if !p.next() {
				p.instance.Scheme = string(schemaRunes)
				return nil
			}
			continue
		}
		break
	}
	p.instance.Scheme = string(schemaRunes)
	return nil
}

func (p *parser) port() error {
	count := 0
	digits := make([]rune, 0)
	for {
		r, _ := p.current()
		if isDigit(r) {
			digits = append(digits, r)
			count++
			if p.next() {
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
	r, _ := p.current()
	if r != '[' {
		return newIriError(p, "Missing starting '[' ip literal")
	}
	if !p.next() {
		return EOIError
	}
	preIndex := p.index
	if ipv6Err := p.ipv6Address(); ipv6Err != nil {
		p.index = preIndex
		if ipvfErr := p.ipvFuture(); ipvfErr != nil {
			return newIriError(p, "Invalid ipv6 or ipv future for ip literal")
		}
	}
	r, _ = p.current()
	if r != ']' {
		return newIriError(p, "Missing ending ']' ip literal")
	}
	p.next()
	return nil
}

func (p *parser) ipvFuture() error {
	r, _ := p.current()
	if r != 'v' {
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
	r, _ = p.current()
	if r != '.' {
		return newIriError(p, "Invalid IpvFuture")
	}
	p.next()
	postCount := 0
	for {
		r, _ = p.current()
		if isUnreserved(r) || isSubDelim(r) || r == ':' {
			postCount++
			if p.next() {
				continue
			} else {
				return EOIError
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
	zeroCollapse := false
	prevColon := false

	for {
		r, _ := p.current()
		if r == ':' {
			if prevColon {
				if zeroCollapse {
					return newIriError(p, "Ambiguous ':'")
				}
				zeroCollapse = true
			} else if zeroCollapse {
				return newIriError(p, "Invalid zero collapse in ipv6")
			}
			prevColon = true
		} else {
			prevColon = false
			if pErr := p.h16(); pErr != nil {
				if groupCount < 8 {
					if zeroCollapse {
						return nil
					}
					return newIriError(p, "Invalid group count in ipv6")
				}
				break
			}
			groupCount++
		}
		if groupCount == 8 {
			break
		}
		if !p.next() {
			return EOIError
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
	if !p.next() {
		return EOIError
	}
	r, _ := p.current()
	if r != ':' {
		return newIriError(p, "invalid ls32 value")
	}
	if !p.next() {
		return EOIError
	}
	if h16Err := p.h16(); h16Err != nil {
		return h16Err
	}
	return nil
}

func (p *parser) h16() error {
	r, _ := p.current()
	if !isHexDigit(r) {
		return newIriError(p, "invalid h16 value")
	}
	hexCount := 1
	for hexCount < 4 {
		pv, pErr := p.peek()
		if pErr != nil {
			return nil
		}
		if isHexDigit(pv) {
			if !p.next() {
				return EOIError
			}
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
		oErr := p.decOctet()
		if oErr != nil && oErr != EOIError {
			return oErr
		}
		octCount++
		if octCount == 4 {
			break
		}
		r, _ := p.current()
		if r != '.' {
			return newIriError(p, "invalid ipv4 address")
		}
		if !p.next() {
			return EOIError
		}
	}
	return nil
}

func (p *parser) decOctet() error {
	octetRunes := make([]rune, 0)
	r, _ := p.current()
	if !isDigit(r) {
		return newIriError(p, "invalid decimal octet")
	}
	octetRunes = append(octetRunes, r)
	peek, peekErr := p.peek()
	if peekErr != nil || !isDigit(peek) {
		if len(octetRunes) == 1 {
			if !p.next() {
				return EOIError
			}
		}
		return nil
	}
	if !p.next() {
		return EOIError
	}
	r, _ = p.current()
	octetRunes = append(octetRunes, r)
	d, _ := strconv.Atoi(string(octetRunes))
	if d < 10 {
		return newIriError(p, "invalid octet value")
	}
	peek, peekErr = p.peek()
	if peekErr != nil || !isDigit(peek) {
		if len(octetRunes) == 2 {
			if !p.next() {
				return EOIError
			}
		}
		return nil
	}
	if !p.next() {
		return EOIError
	}
	r, _ = p.current()
	octetRunes = append(octetRunes, r)
	d, _ = strconv.Atoi(string(octetRunes))
	if d < 100 || d > 255 {
		return newIriError(p, "invalid octet value")
	}
	peek, peekErr = p.peek()
	if peekErr != nil {
		if len(octetRunes) == 1 {
			if !p.next() {
				return EOIError
			}
		}
		return nil
	}
	if isDigit(peek) {
		return newIriError(p, "invalid octet value")
	}
	if !p.next() {
		return EOIError
	}
	return nil
}

func (p *parser) pctEncoded() error {
	if r, _ := p.current(); r != '%' {
		return newIriError(p, "invalid pct encoding")
	}
	if !p.next() {
		return EOIError
	}
	if r, _ := p.current(); !isHexDigit(r) {
		return newIriError(p, "invalid pct encoding")
	}
	if !p.next() {
		return EOIError
	}
	if r, _ := p.current(); !isHexDigit(r) {
		return newIriError(p, "invalid pct encoding")
	}
	if !p.next() {
		return EOIError
	}
	return nil
}

func newIriError(p *parser, message string) IriError {
	r, _ := p.current()
	return IriError{
		index:   p.index,
		char:    r,
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
