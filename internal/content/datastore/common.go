package datastore

import (
	"context"
	"database/sql/driver"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PGXPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

func PrepareSetterMap[T any](ctx context.Context, updates T) ([]string, []any) {
	ks := []string{}
	vs := []any{}
	m := GetMappings(updates)

	for k, v := range m {
		ks = append(ks, k)
		vs = append(vs, v)
	}

	return ks, vs
}

func GetMappings(x any) map[string]any {
	unsettableTyp := reflect.TypeOf((*interface{ IsUnset() bool })(nil)).Elem()
	driverValuerTyp := reflect.TypeOf((*interface{ Value() (driver.Value, error) })(nil)).Elem()
	var v map[string]any
	typ := reflect.TypeOf(x)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return v
	}

	refVal := reflect.ValueOf(x)
	if refVal.Kind() == reflect.Pointer {
		if refVal.IsNil() {
			return v
		}

		refVal = refVal.Elem()
	}

	v = make(map[string]any, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := fieldTag(field)
		if tag == "-" {
			continue
		}

		parts := strings.Split(tag, ",")
		name := parts[0]
		val := refVal.Field(i)
		shouldSet := true

		if val.Type().Implements(unsettableTyp) {
			shouldSet = !val.MethodByName("IsUnset").Call(nil)[0].Interface().(bool)
		}

		if !shouldSet {
			continue
		}

		if val.Type().Implements(driverValuerTyp) {
			v[name] = val.MethodByName("Value").Call(nil)[0].Interface()
			continue
		}

		v[name] = val.Interface()
	}

	return v
}

func fieldTag(f reflect.StructField) string {
	tag, ok := f.Tag.Lookup("db")
	if ok {
		return tag
	}

	return f.Tag.Get("json")
}
