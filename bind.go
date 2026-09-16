package gosip

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var errNotStructPtr = errors.New("v harus berupa non-nil pointer ke struct")

type fieldInfo struct {
	Idx  int
	Name string
	Type reflect.Type
}

type structInfo struct {
	QueryFields []fieldInfo
	ParamFields []fieldInfo
}

var structCache sync.Map // map[reflect.Type]*structInfo

func getStructInfo(rt reflect.Type) *structInfo {
	if info, ok := structCache.Load(rt); ok {
		return info.(*structInfo)
	}

	info := &structInfo{}
	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		// parse query tag
		qName, _, _ := strings.Cut(field.Tag.Get("query"), ",")
		if qName != "" && qName != "-" {
			info.QueryFields = append(info.QueryFields, fieldInfo{
				Idx:  i,
				Name: qName,
				Type: field.Type,
			})
		}

		// parse param tag
		pName, _, _ := strings.Cut(field.Tag.Get("param"), ",")
		if pName != "" && pName != "-" {
			info.ParamFields = append(info.ParamFields, fieldInfo{
				Idx:  i,
				Name: pName,
				Type: field.Type,
			})
		}
	}

	structCache.Store(rt, info)
	return info
}

// BindJSON ini buat nge-parse body request format JSON langsung masuk ke struct `v`.
// Pastiin `v` itu pointer ke struct ya, dan jangan nil.
// Nggak usah repot tutup body request, udah ditutupin otomatis kok.
func (c *Ctx) BindJSON(v any) error {
	defer func() { _ = c.Request.Body.Close }()
	if err := json.NewDecoder(c.Request.Body).Decode(v); err != nil {
		return fmt.Errorf("gosip: bind json: %w", err)
	}
	return nil
}

// BindQuery ini dipakai buat ngambil data dari URL query parameters (yang ada di URL setelah tanda tanya `?`).
// Tinggal kasih tag `query:"nama_parameternya"` di struct kamu, nanti datanya otomatis masuk.
// Asyiknya, dia udah pakai sistem cache, jadi kenceng banget buat request-request selanjutnya!
func (c *Ctx) BindQuery(v any) error {
	rv, err := structElem(v)
	if err != nil {
		return fmt.Errorf("gosip: bind query: %w", err)
	}
	info := getStructInfo(rv.Type())
	vals := c.Request.URL.Query()

	for _, f := range info.QueryFields {
		values, ok := vals[f.Name]
		if !ok || len(values) == 0 {
			continue
		}
		if err := setField(rv.Field(f.Idx), f.Type, values); err != nil {
			return fmt.Errorf("gosip: bind query: field %s: %w", rv.Type().Field(f.Idx).Name, err)
		}
	}
	return nil
}

// BindParams mirip kayak BindQuery, tapi ini khusus buat ngambil path parameter di URL.
// Kasih tag `param:"nama_parameternya"` di field struct kamu biar datanya bisa di-bind.
// Sama kayak BindQuery, dia juga udah dilengkapin cache biar makin ngebut!
func (c *Ctx) BindParams(v any) error {
	rv, err := structElem(v)
	if err != nil {
		return fmt.Errorf("gosip: bind params: %w", err)
	}
	info := getStructInfo(rv.Type())

	for _, f := range info.ParamFields {
		val := c.Request.PathValue(f.Name)
		if val == "" {
			continue
		}
		if err := setField(rv.Field(f.Idx), f.Type, []string{val}); err != nil {
			return fmt.Errorf("gosip: bind params: field %s: %w", rv.Type().Field(f.Idx).Name, err)
		}
	}
	return nil
}

// structElem memvalidasi bahwa v adalah non-nil pointer ke struct,
// dan mengembalikan reflect.Value dari elem-nya.
func structElem(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return reflect.Value{}, errNotStructPtr
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, errNotStructPtr
	}
	return rv, nil
}

// setField mengisi reflect.Value berdasarkan slice string dari input.
// Mendukung: tipe primitif, pointer ke primitif, dan slice dari primitif.
func setField(fv reflect.Value, ft reflect.Type, values []string) error {
	// Pointer: alokasikan dan isi elem-nya
	if ft.Kind() == reflect.Pointer {
		ptr := reflect.New(ft.Elem())
		if err := setField(ptr.Elem(), ft.Elem(), values); err != nil {
			return err
		}
		fv.Set(ptr)
		return nil
	}

	// Slice: isi tiap elemen dari tiap nilai
	if ft.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(ft, len(values), len(values))
		for i, v := range values {
			if err := setField(slice.Index(i), ft.Elem(), []string{v}); err != nil {
				return fmt.Errorf("index %d: %w", i, err)
			}
		}
		fv.Set(slice)
		return nil
	}

	return setPrimitive(fv, ft, values[0])
}

// setPrimitive mengisi field primitif berdasarkan string val.
func setPrimitive(fv reflect.Value, ft reflect.Type, val string) error {
	switch ft.Kind() {
	case reflect.String:
		fv.SetString(val)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(val, 10, ft.Bits())
		if err != nil {
			return err
		}
		fv.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(val, 10, ft.Bits())
		if err != nil {
			return err
		}
		fv.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(val, ft.Bits())
		if err != nil {
			return err
		}
		fv.SetFloat(n)

	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		fv.SetBool(b)

	default:
		return fmt.Errorf("tipe tidak didukung: %s", ft.Kind())
	}
	return nil
}
