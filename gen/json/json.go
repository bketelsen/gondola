// Package json generates methods for encoding/decoding types to/from JSON.
//
// When used correctly, these methods can easily give a ~200-300% performance
// increase when serializing objects to JSON while also reducing memory usage
// by ~95-99%. For taking advantage of these gains, you must use
// gnd.la/app/serialize or Context.WriteJSON to encode to JSON, since
// json.Marshal won't use these methods correctly and might even have worse
// performance when these methods are implemented.
//
// This is a small benchmark comparing the performance of these JSON encoding
// methods. JSONDirect uses WriteJSON(), JSONSerialize uses
// gnd.la/app/serialize (which adds some overhead because it also sets the
// Content-Length and Content-Encoding headers and thus must encode into an
// intermediate buffer first), while JSON uses json.Marshal(). All three
// benchmarks write the result to ioutil.Discard.
//
//  BenchmarkJSONDirect	    1000000 1248 ns/op	117.73 MB/s 16 B/op	2 allocs/op
//  BenchmarkJSONSerialize  1000000 1587 ns/op	92.62 MB/s  16 B/op	2 allocs/op
//  BenchmarkJSON	    500000  4583 ns/op	32.07 MB/s  620 B/op	4 allocs/op
//
// Code generated by this package respects json related struct tags except
// omitempty and and encodes time.Time UTC as a Unix time (encoding/json uses
// time.Format).
//
// If you want to specify a different serialization when using encoding/json
// than when using this package, you can use the "genjson" field tag. Fields
// with a genjson tag will use it and ignore the "json" tag.
//
// The recommended way use to generate JSON methods for a given package is
// using the gondola command rather than using this package directly.
package json

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"gnd.la/gen/genutil"
	"gnd.la/log"
	"gnd.la/util/structs"
	"path/filepath"
	"regexp"
)

const (
	defaultBufSize = 8 * 1024
)

type Method struct {
	Key       string
	Name      string
	OmitEmpty bool
}

// Options specify the options used when generating JSON related
// methods.
type Options struct {
	// Wheter to generate a MarshalJSON method. This is false by default
	// because in most cases will result in lower performance when using
	// json.Marshal, since the encoder from encoding/json will revalidate
	// the returned JSON, resulting in a performance loss. Turn this on
	// only if you're using the Methods feature (otherwise you'll get
	// different results when serializing with json.Marshal).
	MarshalJSON bool
	// The size of the allocated buffers for serializing to JSON. If zero,
	// the default size of 8192 is used (8K).
	BufferSize int
	// The maximum buffer size. Buffers which grow past this size won't
	// be reused. If zero, it takes the same value os BufferSize.
	MaxBufferSize int
	// The number of buffers to be kept for reusing. If zero, it defaults
	// to GOMAXPROCS. Set it to a negative number to disable buffering.
	BufferCount int
	// If not zero, this takes precedence over BufferCount. The number of
	// maximum buffers will be GOMAXPROCS * BuffersPerProc.
	BuffersPerProc int
	// Methods indicates struct methods which should be included in the JSON
	// output. The key in the map is the type name in the package (e.g.
	// MyStruct not mypackage.MyStruct).
	Methods map[string][]*Method
	// If not nil, only types matching this regexp will be included.
	Include *regexp.Regexp
	// If not nil, types matching this regexp will be excluded.
	Exclude *regexp.Regexp
}

// Gen generates a WriteJSON method and, optionally, MarshalJSON for every
// exported type in the given package. The package might be either an
// absolute path or an import path.
func Gen(pkgName string, opts *Options) error {
	pkg, err := genutil.NewPackage(pkgName)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkg.Name()))
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("\nimport (\n")
	imports := []string{"bytes", "io", "strconv", "unicode/utf8"}
	if opts == nil || opts.BufferCount == 0 || opts.BuffersPerProc != 0 {
		imports = append(imports, "runtime")
	}
	for _, v := range imports {
		buf.WriteString(fmt.Sprintf("%q\n", v))
	}
	buf.WriteString(")\n")
	buf.WriteString("var _ = strconv.FormatBool\n")
	buf.WriteString("var _ = io.ReadFull\n")
	var include *regexp.Regexp
	var exclude *regexp.Regexp
	if opts != nil {
		include = opts.Include
		exclude = opts.Exclude
	}
	var methods bytes.Buffer
	for _, v := range pkg.ExportedTypes(include, exclude) {
		methods.Reset()
		if err := jsonMarshal(v, opts, &methods); err != nil {
			log.Warningf("Skipping type %s: %s", v.Obj().Name(), err)
			continue
		}
		buf.WriteString(methods.String())
	}
	buf.WriteString(encode_go)
	bufSize := defaultBufSize
	maxBufSize := bufSize
	bufferCount := 0
	buffersPerProc := 0
	if opts != nil {
		if opts.BufferSize > 0 {
			bufSize = opts.BufferSize
			maxBufSize = bufSize
		}
		if opts.MaxBufferSize >= maxBufSize {
			maxBufSize = opts.MaxBufferSize
		}
		bufferCount = opts.BufferCount
		buffersPerProc = opts.BuffersPerProc
	}
	buf.WriteString(fmt.Sprintf("const jsonBufSize = %d\n", bufSize))
	buf.WriteString(fmt.Sprintf("const jsonMaxBufSize = %d\n", maxBufSize))
	if buffersPerProc > 0 {
		buf.WriteString(fmt.Sprintf("var jsonBufferCount = runtime.GOMAXPROCS(0) * %d\n", buffersPerProc))
	} else if bufferCount > 0 {
		buf.WriteString(fmt.Sprintf("const jsonBufferCount = %d\n", bufferCount))
	} else {
		buf.WriteString("var jsonBufferCount = runtime.GOMAXPROCS(0)\n")
	}
	buf.WriteString(buffer_go)
	out := filepath.Join(pkg.Dir(), "gen_json.go")
	log.Debugf("Writing autogenerated JSON methods to %s", out)
	return genutil.WriteAutogen(out, buf.Bytes())
}

func jsonMarshal(typ *types.Named, opts *Options, buf *bytes.Buffer) error {
	tname := typ.Obj().Name()
	if _, ok := typ.Underlying().(*types.Struct); ok {
		tname = "*" + tname
	}
	if opts != nil && opts.MarshalJSON {
		buf.WriteString(fmt.Sprintf("func(o %s) MarshalJSON() ([]byte, error) {\n", tname))
		buf.WriteString("var buf bytes.Buffer\n")
		buf.WriteString("_, err := o.WriteJSON(&buf)\n")
		buf.WriteString("return buf.Bytes(), err\n")
		buf.WriteString("}\n\n")
	}
	buf.WriteString(fmt.Sprintf("func(o %s) WriteJSON(w io.Writer) (int, error) {\n", tname))
	buf.WriteString("buf := jsonGetBuffer()\n")
	if err := jsonValue(typ, nil, "o", opts, buf); err != nil {
		return err
	}
	buf.WriteString("n, err := w.Write(buf.Bytes())\n")
	buf.WriteString("jsonPutBuffer(buf)\n")
	buf.WriteString("return n, err\n")
	buf.WriteString("}\n\n")
	return nil
}

func fieldTag(tag string) *structs.Tag {
	if gtag := structs.NewStringTagNamed(tag, "genjson"); gtag != nil && !gtag.IsEmpty() {
		return gtag
	}
	return structs.NewStringTagNamed(tag, "json")
}

func jsonStruct(st *types.Struct, p types.Type, name string, opts *Options, buf *bytes.Buffer) error {
	buf.WriteString("buf.WriteByte('{')\n")
	count := st.NumFields()
	hasFields := false
	for ii := 0; ii < count; ii++ {
		field := st.Field(ii)
		if field.IsExported() {
			key := field.Name()
			omitEmpty := false
			tag := st.Tag(ii)
			if ftag := fieldTag(tag); ftag != nil {
				if n := ftag.Name(); n != "" {
					key = n
				}
				omitEmpty = ftag.Has("omitempty")
			}
			if key != "-" {
				if hasFields {
					buf.WriteString("buf.WriteByte(',')\n")
				}
				hasFields = true
				if err := jsonField(field, key, name+"."+field.Name(), omitEmpty, opts, buf); err != nil {
					return err
				}
			}
		}
	}
	if opts != nil {
		if named, ok := p.(*types.Named); ok {
			methods := opts.Methods[named.Obj().Name()]
			count := named.NumMethods()
			for _, v := range methods {
				found := false
				for ii := 0; ii < count; ii++ {
					fn := named.Method(ii)
					if fn.Name() == v.Name {
						found = true
						signature := fn.Type().(*types.Signature)
						if p := signature.Params(); p != nil || p.Len() > 0 {
							return fmt.Errorf("method %s on type %s requires arguments", v.Name, named.Obj().Name())
						}
						res := signature.Results()
						if res == nil || res.Len() != 1 {
							return fmt.Errorf("method %s on type %s must return exactly one value", v.Name, named.Obj().Name())
						}
						if hasFields {
							buf.WriteString("buf.WriteByte(',')\n")
						}
						hasFields = true
						if err := jsonField(res.At(0), v.Key, name+"."+v.Name+"()", v.OmitEmpty, opts, buf); err != nil {
							return err
						}
						break
					}
				}
				if !found {
					return fmt.Errorf("type %s does not have method %s", named.Obj().Name(), v.Name)
				}
			}
		}
	}
	buf.WriteString("buf.WriteByte('}')\n")
	return nil
}

func jsonSlice(sl *types.Slice, p types.Type, name string, opts *Options, buf *bytes.Buffer) error {
	buf.WriteString("buf.WriteByte('[')\n")
	buf.WriteString(fmt.Sprintf("for ii, v := range %s {\n", name))
	buf.WriteString("if ii > 0 {\n")
	buf.WriteString("buf.WriteByte(',')\n")
	buf.WriteString("}\n")
	if err := jsonValue(sl.Elem(), nil, "v", opts, buf); err != nil {
		return err
	}
	buf.WriteString("}\n")
	buf.WriteString("buf.WriteByte(']')\n")
	return nil
}

func jsonField(field *types.Var, key string, name string, omitEmpty bool, opts *Options, buf *bytes.Buffer) error {
	// TODO: omitEmpty
	buf.WriteString(fmt.Sprintf("buf.WriteString(%q)\n", fmt.Sprintf("%q", key)))
	buf.WriteString("buf.WriteByte(':')\n")
	if err := jsonValue(field.Type(), nil, name, opts, buf); err != nil {
		return err
	}
	return nil
}

func jsonValue(vtype types.Type, ptype types.Type, name string, opts *Options, buf *bytes.Buffer) error {
	switch typ := vtype.(type) {
	case *types.Basic:
		k := typ.Kind()
		_, isPointer := ptype.(*types.Pointer)
		if isPointer {
			name = "*" + name
		}
		switch k {
		case types.Bool:
			buf.WriteString(fmt.Sprintf("buf.WriteString(strconv.FormatBool(%s))\n", name))
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			buf.WriteString(fmt.Sprintf("buf.WriteString(strconv.FormatInt(int64(%s), 10))\n", name))
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			buf.WriteString(fmt.Sprintf("buf.WriteString(strconv.FormatUint(uint64(%s), 10))\n", name))
		case types.Float32, types.Float64:
			bitSize := 64
			if k == types.Float32 {
				bitSize = 32
			}
			buf.WriteString(fmt.Sprintf("buf.WriteString(strconv.FormatFloat(float64(%s), 'g', -1, %d))\n", name, bitSize))
		case types.String:
			buf.WriteString(fmt.Sprintf("jsonEncodeString(buf, string(%s))\n", name))
		default:
			return fmt.Errorf("can't encode basic kind %v", typ.Kind())
		}
	case *types.Named:
		if typ.Obj().Pkg().Name() == "time" && typ.Obj().Name() == "Time" {
			buf.WriteString(fmt.Sprintf("buf.WriteString(strconv.FormatInt(%s.UTC().Unix(), 10))\n", name))
		} else {
			if err := jsonValue(typ.Underlying(), typ, name, opts, buf); err != nil {
				return err
			}
		}
	case *types.Slice:
		if err := jsonSlice(typ, ptype, name, opts, buf); err != nil {
			return err
		}
	case *types.Struct:
		if err := jsonStruct(typ, ptype, name, opts, buf); err != nil {
			return err
		}
	case *types.Pointer:
		buf.WriteString(fmt.Sprintf("if %s == nil {\n", name))
		buf.WriteString("buf.WriteString(\"null\")\n")
		buf.WriteString("} else {\n")
		if err := jsonValue(typ.Elem(), typ, name, opts, buf); err != nil {
			return err
		}
		buf.WriteString("}\n")
	default:
		return fmt.Errorf("can't encode type %T %v (%T)", typ, typ, typ.Underlying())
	}
	return nil
}
