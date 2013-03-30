// This work is subject to the CC0 1.0 Universal (CC0 1.0) Public Domain Dedication
// license. Its contents can be found at:
// http://creativecommons.org/publicdomain/zero/1.0/

package main

import (
	"compress/gzip"
	"fmt"
	"io"
)

// translate translates the input file to go source code.
func translate(input io.Reader, output io.Writer, pkgname, build, funcname string, uncompressed, nomemcpy bool) {
	write_header(output, pkgname, build)
	if nomemcpy {
		if uncompressed {
			translate_nomemcpy_uncomp(input, output, funcname)
		} else {
			translate_nomemcpy_comp(input, output, funcname)
		}
	} else {
		if uncompressed {
			translate_memcpy_uncomp(input, output, funcname)
		} else {
			translate_memcpy_comp(input, output, funcname)
		}
	}
}

func write_header(output io.Writer, pkgname, build string) {
	if build != "" {
		fmt.Fprintf(output, "// +build %s\n\n", build)
	}

	fmt.Fprintf(output, "package %s\n", pkgname)
}

// input -> gzip -> gowriter -> output.
func translate_memcpy_comp(input io.Reader, output io.Writer, funcname string) {
	fmt.Fprintf(output, `
import (
	"bytes"
	"compress/gzip"
	"io"
)

// %s returns raw, uncompressed file data.
func %s() []byte {
	gz, err := gzip.NewReader(bytes.NewBuffer([]byte{`, funcname, funcname)

	gz := gzip.NewWriter(&ByteWriter{Writer: output})
	io.Copy(gz, input)
	gz.Close()

	fmt.Fprint(output, `
	}))

	if err != nil {
		panic("Decompression failed: " + err.Error())
	}

	var b bytes.Buffer
	io.Copy(&b, gz)
	gz.Close()

	return b.Bytes()
}`)
}

// input -> gzip -> gowriter -> output.
func translate_memcpy_uncomp(input io.Reader, output io.Writer, funcname string) {
	fmt.Fprintf(output, `
// %s returns raw file data.
func %s() []byte {
	return []byte{`, funcname, funcname)

	io.Copy(&ByteWriter{Writer: output}, input)

	fmt.Fprint(output, `
	}
}`)
}

// input -> gzip -> gowriter -> output.
func translate_nomemcpy_comp(input io.Reader, output io.Writer, funcname string) {
	fmt.Fprintf(output, `
import (
	"bytes"
	"compress/gzip"
	"io"
	"reflect"
	"unsafe"
)

var _%s = "`, funcname)

	gz := gzip.NewWriter(&StringWriter{Writer: output})
	io.Copy(gz, input)
	gz.Close()

	fmt.Fprintf(output, `"

// %s returns raw, uncompressed file data.
func %s() []byte {
	var empty [0]byte
	sx := (*reflect.StringHeader)(unsafe.Pointer(&_%s))
	b := empty[:]
	bx := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bx.Data = sx.Data
	bx.Len = len(_%s)
	bx.Cap = bx.Len

	gz, err := gzip.NewReader(bytes.NewBuffer(b))

	if err != nil {
		panic("Decompression failed: " + err.Error())
	}

	var buf bytes.Buffer
	io.Copy(&buf, gz)
	gz.Close()

	return buf.Bytes()
}
`, funcname, funcname, funcname, funcname)
}

// input -> gowriter -> output.
func translate_nomemcpy_uncomp(input io.Reader, output io.Writer, funcname string) {
	fmt.Fprintf(output, `
import (
	"reflect"
	"unsafe"
)

var _%s = "`, funcname)

	io.Copy(&StringWriter{Writer: output}, input)

	fmt.Fprintf(output, `"

// %s returns raw file data.
//
// WARNING: The returned byte slice is READ-ONLY.
// Attempting to alter the slice contents will yield a runtime panic.
func %s() []byte {
	var empty [0]byte
	sx := (*reflect.StringHeader)(unsafe.Pointer(&_%s))
	b := empty[:]
	bx := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bx.Data = sx.Data
	bx.Len = len(_%s)
	bx.Cap = bx.Len
	return b
}
`, funcname, funcname, funcname, funcname)
}
