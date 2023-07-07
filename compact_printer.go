package jl

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/emirpasic/gods/maps/treemap"
)

// CompactPrinter can print logs in a variety of compact formats, specified by FieldFormats.
type CompactPrinter struct {
	Out io.Writer
	// Disable colors disables adding color to fields.
	DisableColor bool
	// Disable truncate disables the Ellipsize and Truncate transforms.
	DisableTruncate bool
	// FieldFormats specifies the format the printer should use for logs. It defaults to DefaultCompactPrinterFieldFmt. Fields
	// are formatted in the order they are provided. If a FieldFmt produces a field that does not end with a whitespace,
	// a space character is automatically appended.
	FieldFormats []FieldFmt
}

// FieldFmt specifies a single field formatted by the CompactPrinter.
type FieldFmt struct {
	// Name of the field. This is used to find the field by key name if Finders is not set.
	Name string
	// List of FieldFinders to use to locate the field. Finders are executed in order until the first one that returns
	// non-nil.
	Finders []FieldFinder
	// Takes the output of the Finder and turns it into a string. If not set, DefaultStringer is used.
	Stringer Stringer
	// List of transformers to run on the field found to format the field.
	Transformers []Transformer
}

// DefaultCompactPrinterFieldFmt is a format for the CompactPrinter that tries to present logs in an easily skimmable manner
// for most types of logs.
var DefaultCompactPrinterFieldFmt = []FieldFmt{{
	Name:         "lvl",
	Finders:      []FieldFinder{ByNames("level", "lvl")},
	Transformers: []Transformer{Truncate(4), UpperCase, ColorMap(LevelColors)},
}, {
	Name:         "ts",
	Finders:      []FieldFinder{ByNames("time", "ts")},
	Transformers: []Transformer{UnixTimestamp, ColorTimestamp()},
	Stringer:     NumberStringer,
}, {
	Name:         "thread",
	Transformers: []Transformer{Ellipsize(16), Format("[%s]"), RightPad(18), ColorSequence(AllColors)},
}, {
	Name:         "logger",
	Finders:      []FieldFinder{ByNames("logger", "caller")},
	Transformers: []Transformer{Ellipsize(20), Format("%s|"), LeftPad(21), ColorSequence(AllColors)},
}, {
	Name:    "msg",
	Finders: []FieldFinder{ByNames("message", "msg")},
}}

// NewCompactPrinter allocates and returns a new compact printer.
func NewCompactPrinter(w io.Writer) *CompactPrinter {
	return &CompactPrinter{
		Out:          w,
		FieldFormats: DefaultCompactPrinterFieldFmt,
	}
}

func (p *CompactPrinter) Print(entry *Entry) {
	if entry.Partials == nil {
		fmt.Fprintln(p.Out, string(entry.Raw))
		return
	}

	for i, fieldFmt := range p.FieldFormats {
		ctx := Context{
			DisableColor:    p.DisableColor,
			DisableTruncate: p.DisableTruncate,
		}
		formattedField := fieldFmt.format(&ctx, entry)
		if formattedField != "" {
			if i != 0 && !strings.HasPrefix(formattedField, "\n") {
				p.Out.Write([]byte(" "))
			}
			p.Out.Write([]byte(formattedField))
		}
	}

	m := treemap.NewWithStringComparator()

	var skip bool
	for key, value := range entry.Partials {
		skip = false

		for _, fieldFmt := range p.FieldFormats {
			if fieldFmt.Name == key {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		m.Put(key, " "+key+":="+string(value))
	}

	for _, v := range m.Values() {
		p.Out.Write([]byte(v.(string)))
	}

	p.Out.Write([]byte("\n"))
}

func (f *FieldFmt) format(ctx *Context, entry *Entry) string {
	var v interface{}
	// Find the value
	if len(f.Finders) > 0 {
		for _, finder := range f.Finders {
			if v = finder(entry); v != nil {
				break
			} else {
			}
		}
	} else {
		v = entry.Partials[f.Name]
	}
	if v == nil {
		return ""
	}

	// Stringify the value
	var s string
	if f.Stringer != nil {
		s = f.Stringer(ctx, v)
	} else {
		s = DefaultStringer(ctx, v)
	}
	s = strings.TrimRightFunc(s, unicode.IsSpace)

	if s == "" {
		return ""
	}

	original := s
	ctx.Original = original
	// Apply transforms
	for _, transform := range f.Transformers {
		s = transform.Transform(ctx, s)
	}

	if s == "" {
		return ""
	}

	return s
}
