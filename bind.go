package gosip

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var errNotStructPtr = errors.New("v harus berupa non-nil pointer ke struct")

// BindJSON mendekode request body JSON ke dalam v.
// v harus berupa non-nil pointer ke struct.
// Body otomatis ditutup setelah dibaca.
func (c *Ctx) BindJSON(v any) error {
	defer c.Request.Body.Close()
	if err := json.NewDecoder(c.Request.Body).Decode(v); err != nil {
		return fmt.Errorf("gosip: bind json: %w", err)
	}
	return nil
}

// BindQuery mendekode URL query parameters ke dalam v.
// Field struct harus diberi tag `query:"nama"`.
//
// Contoh:
//
//	type Filter struct {
//	    Page  int    `query:"page"`
//	    Limit int    `query:"limit"`
//	    Search string `query:"q"`
//	}
func (c *Ctx) BindQuery(v any) error {
	if err := decodeValues(c.Request.URL.Query(), v, "query"); err != nil {
		return fmt.Errorf("gosip: bind query: %w", err)
	}
	return nil
}

// BindParams mendekode URL path parameters ke dalam v.
// Field struct harus diberi tag `param:"nama"`.
//
// Contoh:
//
//	type PathParams struct {
//	    ID   int    `param:"id"`
//	    Slug string `param:"slug"`
//	}
func (c *Ctx) BindParams(v any) error {
	rv, err := structElem(v)
	if err != nil {
		return fmt.Errorf("gosip: bind params: %w", err)
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("param"), ",")
		if name == "" || name == "-" {
			continue
		}
		val := c.Request.PathValue(name)
		if val == "" {
			continue
		}
		if err := setField(rv.Field(i), field.Type, []string{val}); err != nil {
			return fmt.Errorf("gosip: bind params: field %s: %w", field.Name, err)
		}
	}
	return nil
}

// decodeValues mendekode url.Values ke dalam struct menggunakan struct tag yang ditentukan.
func decodeValues(vals map[string][]string, v any, tag string) error {
	rv, err := structElem(v)
	if err != nil {
		return err
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get(tag), ",")
		if name == "" || name == "-" {
			continue
		}
		values, ok := vals[name]
		if !ok || len(values) == 0 {
			continue
		}
		if err := setField(rv.Field(i), field.Type, values); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

// structElem memvalidasi bahwa v adalah non-nil pointer ke struct,
// dan mengembalikan reflect.Value dari elem-nya.
func structElem(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
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
	if ft.Kind() == reflect.Ptr {
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
