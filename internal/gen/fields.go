package gen

import (
	"fmt"
	"strconv"
	"strings"
)

// Field is one column from a spec like "title:string", "bio:text:nullable",
// "price:float:default=0", "email:string:unique" or "user:references".
type Field struct {
	Name     string // as typed, e.g. "user", "publishedAt"
	Type     string // normalised type
	Nullable bool
	Unique   bool
	Default  string
	HasDef   bool
	Ref      string // referenced model for references, e.g. "User"
}

var typeAliases = map[string]string{
	"string": "string", "str": "string", "varchar": "string", "email": "string", "url": "string",
	"text": "text", "longtext": "text",
	"integer": "integer", "int": "integer", "bigint": "integer",
	"float": "float", "double": "float", "decimal": "float", "number": "float", "real": "float",
	"boolean": "boolean", "bool": "boolean",
	"timestamp": "timestamp", "datetime": "timestamp", "date": "timestamp",
	"json": "json", "jsonb": "json",
	"references": "references", "ref": "references", "belongsto": "references", "foreign": "references",
}

// FieldTypes are the accepted type names, for help output.
const FieldTypes = "string, text, integer, float, boolean, timestamp, json, references"

func ParseFields(specs []string) ([]Field, error) {
	var out []Field
	seen := map[string]bool{}
	for _, spec := range specs {
		f, err := ParseField(spec)
		if err != nil {
			return nil, err
		}
		if seen[f.Key()] {
			return nil, fmt.Errorf("duplicate field %q", f.Name)
		}
		seen[f.Key()] = true
		out = append(out, f)
	}
	return out, nil
}

func ParseField(spec string) (Field, error) {
	parts := strings.Split(strings.TrimSpace(spec), ":")
	if len(parts) < 2 || parts[0] == "" {
		return Field{}, fmt.Errorf("invalid field %q — use name:type[:modifier], types: %s", spec, FieldTypes)
	}
	f := Field{Name: parts[0]}
	typ := strings.ToLower(parts[1])
	if strings.HasSuffix(typ, "?") {
		f.Nullable = true
		typ = strings.TrimSuffix(typ, "?")
	}
	norm, ok := typeAliases[typ]
	if !ok {
		return Field{}, fmt.Errorf("unknown type %q in %q — types: %s", parts[1], spec, FieldTypes)
	}
	f.Type = norm
	for _, mod := range parts[2:] {
		switch m := strings.TrimSpace(mod); {
		case m == "nullable" || m == "optional":
			f.Nullable = true
		case m == "unique":
			f.Unique = true
		case strings.HasPrefix(m, "default="):
			f.Default, f.HasDef = strings.TrimPrefix(m, "default="), true
		case f.Type == "references":
			f.Ref = Pascal(m)
		default:
			return Field{}, fmt.Errorf("unknown modifier %q in %q (nullable, unique, default=value)", m, spec)
		}
	}
	if f.Type == "references" {
		f.Name = strings.TrimSuffix(strings.TrimSuffix(f.Name, "_id"), "Id")
		if f.Ref == "" {
			f.Ref = Pascal(f.Name)
		}
	}
	return f, nil
}

// Key is the TypeScript property name.
func (f Field) Key() string {
	if f.Type == "references" {
		return Camel(f.Name) + "Id"
	}
	return Camel(f.Name)
}

// Column is the SQL column name.
func (f Field) Column() string { return Snake(f.Key()) }

// Owner reports whether this field links a record to the user who owns it.
func (f Field) Owner() bool { return f.Type == "references" && f.Ref == "User" }

// ColumnCode renders the model column, e.g. `title: t.string("title").notNull()`.
func (f Field) ColumnCode() string {
	var expr string
	switch f.Type {
	case "references":
		ref := NamesFor(f.Ref)
		expr = fmt.Sprintf("t.foreignId(%q, () => %s.id)", f.Column(), ref.Var)
	case "json":
		expr = fmt.Sprintf("t.json(%q)", f.Column())
	default:
		expr = fmt.Sprintf("t.%s(%q)", f.Type, f.Column())
	}
	if !f.Nullable {
		expr += ".notNull()"
	}
	if f.Unique {
		expr += ".unique()"
	}
	if f.HasDef {
		expr += f.defaultCode()
	}
	return fmt.Sprintf("%s: %s", f.Key(), expr)
}

func (f Field) defaultCode() string {
	switch f.Type {
	case "boolean":
		return fmt.Sprintf(".default(%t)", f.Default == "true" || f.Default == "1")
	case "integer", "float":
		if _, err := strconv.ParseFloat(f.Default, 64); err == nil {
			return ".default(" + f.Default + ")"
		}
	case "timestamp":
		if f.Default == "now" {
			return ".$defaultFn(() => new Date())"
		}
	case "json":
		return ".default(" + f.Default + ")"
	}
	return fmt.Sprintf(".default(%q)", f.Default)
}

// RuleCode renders the Zod rule, e.g. `title: z.string().min(1).max(255)`.
func (f Field) RuleCode() string {
	var rule string
	switch f.Type {
	case "string":
		rule = "z.string().trim().min(1).max(255)"
	case "text":
		rule = "z.string()"
	case "integer":
		rule = "z.coerce.number().int()"
	case "float":
		rule = "z.coerce.number()"
	case "boolean":
		rule = "z.boolean()"
	case "timestamp":
		rule = "z.coerce.date()"
	case "json":
		rule = "z.unknown()"
	case "references":
		rule = "z.string().min(1)"
	}
	if f.Nullable {
		rule += ".nullish()"
	} else if f.HasDef {
		rule += ".optional()"
	}
	return fmt.Sprintf("%s: %s", f.Key(), rule)
}
