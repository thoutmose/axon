package images

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
)

// Drawings for flashcards (EP-52) are SVG an AI agent writes. SanitizeSVG
// rebuilds one from an allowlist of shapes, text, and gradients: no
// scripts, no links, no external resources, no styles, no animation. What
// is not on the list is dropped with everything inside it.

// ErrNotSVG is returned for drawings that are not usable SVG.
var ErrNotSVG = errors.New("not a usable SVG drawing")

// MaxSVG is the largest drawing accepted.
const MaxSVG = 64 << 10

var svgElements = map[string]bool{
	"svg": true, "g": true, "path": true, "circle": true, "ellipse": true, "rect": true, "line": true,
	"polyline": true, "polygon": true, "text": true, "tspan": true, "defs": true, "linearGradient": true,
	"radialGradient": true, "stop": true, "title": true, "desc": true,
}

var svgAttributes = map[string]bool{
	"viewBox": true, "width": true, "height": true, "fill": true, "stroke": true, "stroke-width": true,
	"stroke-linecap": true, "stroke-linejoin": true, "stroke-dasharray": true, "opacity": true,
	"fill-opacity": true, "stroke-opacity": true, "fill-rule": true, "d": true, "cx": true, "cy": true, "r": true,
	"rx": true, "ry": true, "x": true, "y": true, "x1": true, "y1": true, "x2": true, "y2": true, "dx": true, "dy": true,
	"points": true, "transform": true, "font-size": true, "font-family": true, "font-weight": true,
	"font-style": true, "text-anchor": true, "dominant-baseline": true, "offset": true, "stop-color": true,
	"stop-opacity": true, "id": true, "gradientUnits": true, "gradientTransform": true, "fx": true, "fy": true,
	"role": true, "aria-label": true, "preserveAspectRatio": true,
}

var (
	safeValue = regexp.MustCompile(`^[\w\s#.,%()+\-:/'"]*$`)
	localRef  = regexp.MustCompile(`^url\(#[\w-]+\)$`)
)

func safeAttr(a xml.Attr) bool {
	if a.Name.Space != "" || !svgAttributes[a.Name.Local] || len(a.Value) > 20000 || !safeValue.MatchString(a.Value) {
		return false
	}
	v := strings.ToLower(a.Value)
	if strings.Contains(v, "url(") && !localRef.MatchString(strings.TrimSpace(a.Value)) {
		return false
	}
	return !strings.Contains(v, "javascript") && !strings.Contains(v, "expression")
}

// SanitizeSVG returns a safe copy of an SVG drawing.
func SanitizeSVG(src string) ([]byte, error) {
	if len(src) > MaxSVG {
		return nil, ErrNotSVG
	}
	dec := xml.NewDecoder(strings.NewReader(src))
	dec.Strict = true
	var out bytes.Buffer
	enc := xml.NewEncoder(&out)
	depth, skip := 0, 0 // skip > 0 while inside a dropped element
	root := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrNotSVG
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if skip > 0 || !svgElements[t.Name.Local] {
				skip++
				continue
			}
			if depth == 1 {
				if t.Name.Local != "svg" {
					return nil, ErrNotSVG
				}
				root = true
			}
			el := xml.StartElement{Name: xml.Name{Local: t.Name.Local}}
			if depth == 1 {
				el.Attr = append(el.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: "http://www.w3.org/2000/svg"})
			}
			for _, a := range t.Attr {
				if safeAttr(a) {
					el.Attr = append(el.Attr, xml.Attr{Name: xml.Name{Local: a.Name.Local}, Value: a.Value})
				}
			}
			if err := enc.EncodeToken(el); err != nil {
				return nil, ErrNotSVG
			}
		case xml.EndElement:
			depth--
			if skip > 0 {
				skip--
				continue
			}
			if err := enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: t.Name.Local}}); err != nil {
				return nil, ErrNotSVG
			}
		case xml.CharData:
			if skip == 0 && depth > 0 {
				if err := enc.EncodeToken(t.Copy()); err != nil {
					return nil, ErrNotSVG
				}
			}
		}
	}
	if err := enc.Flush(); err != nil || !root {
		return nil, ErrNotSVG
	}
	return out.Bytes(), nil
}
