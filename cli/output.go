package cli

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/term"
)

type OutputFormat string

const (
	TextOutputFormat OutputFormat = "text"
	JSONOutputFormat OutputFormat = "json"

	DefaultOutputFormat = TextOutputFormat
)

var (
	AllOutputFormats = map[OutputFormat]struct{}{
		TextOutputFormat: struct{}{},
		JSONOutputFormat: struct{}{},
	}
)

func (f OutputFormat) MarshalText() ([]byte, error) {
	return []byte(string(f)), nil
}

func (f *OutputFormat) UnmarshalText(d []byte) (err error) {
	s := string(d)
	if _, ok := AllOutputFormats[OutputFormat(s)]; ok {
		*f = OutputFormat(s)
	} else {
		allFormats := ""
		for format, _ := range AllOutputFormats {
			if allFormats != "" {
				allFormats += ", "
			}
			allFormats += string(format)
		}
		err = fmt.Errorf("Unknown output format: %s (known formats: %s)", s, allFormats)
	}

	return
}

type Encoder interface {
	Encode(any) error
}

func (f OutputFormat) CreateEncoder(writer io.Writer) Encoder {
	switch f {
	case TextOutputFormat:
		return TextEncoder{writer}
	case JSONOutputFormat:
		encoder := json.NewEncoder(writer)
		if file, ok := writer.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
			encoder.SetIndent("", "  ")
		}
		return encoder
	default:
		return nil
	}
}

type TextEncoder struct {
	writer io.Writer
}

func (e TextEncoder) Encode(v any) error {
	var result *multierror.Error
	if v == nil {
		return nil
	}

	value := reflect.ValueOf(v)
	marshaler, ok := v.(encoding.TextMarshaler)

	switch {
	case ok:
		d, err := marshaler.MarshalText()
		result = multierror.Append(result, err)
		_, err = e.writer.Write(d)
		result = multierror.Append(result, err)
	case value.Kind() == reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			result = multierror.Append(result, e.Encode(value.Index(i).Interface()))
			_, err := e.writer.Write([]byte("\n"))
			result = multierror.Append(result, err)
		}
	default:
		fmt.Fprintf(e.writer, "%v", v)
	}

	return result.ErrorOrNil()
}
