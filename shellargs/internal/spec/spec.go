package spec

import (
	"fmt"
	"strings"
)

type Spec struct {
	Name        string
	Summary     string
	Description string
	Usage       string
	Fields      []Field
}

type FieldKind string

const (
	FieldFlag   FieldKind = "flag"
	FieldOption FieldKind = "option"
	FieldArg    FieldKind = "arg"
)

type Field struct {
	Kind        FieldKind
	Name        string
	Short       string
	Long        string
	Type        string
	Default     string
	Description string
	Required    string
	Repeatable  bool
	Placeholder string
}

func Parse(raw string) (Spec, error) {
	var out Spec
	lines := strings.Split(raw, "\n")
	seen := map[string]struct{}{}

	for idx, line := range lines {
		lineNo := idx + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if key, value, ok := parseMetadata(trimmed); ok {
			switch key {
			case "name":
				out.Name = value
			case "summary":
				out.Summary = value
			case "description":
				out.Description = value
			case "usage":
				out.Usage = value
			}
			continue
		}

		field, err := parseFieldLine(trimmed)
		if err != nil {
			return Spec{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if _, exists := seen[field.Name]; exists {
			return Spec{}, fmt.Errorf("line %d: duplicate field name %q", lineNo, field.Name)
		}
		seen[field.Name] = struct{}{}
		out.Fields = append(out.Fields, field)
	}

	if err := validate(out); err != nil {
		return Spec{}, err
	}

	return out, nil
}

func parseMetadata(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}

	key := strings.TrimSpace(line[:idx])
	switch key {
	case "name", "summary", "description", "usage":
	default:
		return "", "", false
	}

	return key, strings.TrimSpace(line[idx+1:]), true
}

func parseFieldLine(line string) (Field, error) {
	segments := strings.Split(line, "|")
	if len(segments) == 0 {
		return Field{}, fmt.Errorf("empty field line")
	}

	head := strings.Fields(strings.TrimSpace(segments[0]))
	if len(head) != 2 {
		return Field{}, fmt.Errorf("field line must start with `<kind> <name>`")
	}

	field := Field{
		Kind: FieldKind(head[0]),
		Name: head[1],
		Type: "string",
	}

	switch field.Kind {
	case FieldFlag, FieldOption, FieldArg:
	default:
		return Field{}, fmt.Errorf("unsupported field kind %q", head[0])
	}

	if field.Kind != FieldArg {
		field.Long = kebabCase(field.Name)
	}
	if field.Kind == FieldFlag {
		field.Type = "bool"
	}
	field.Placeholder = defaultPlaceholder(field.Name)

	for _, rawSeg := range segments[1:] {
		seg := strings.TrimSpace(rawSeg)
		if seg == "" {
			continue
		}

		if !strings.Contains(seg, "=") {
			switch seg {
			case "required":
				field.Required = "yes"
			case "repeatable":
				field.Repeatable = true
			default:
				return Field{}, fmt.Errorf("unsupported attribute %q", seg)
			}
			continue
		}

		parts := strings.SplitN(seg, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "short":
			field.Short = value
		case "long":
			field.Long = value
		case "type":
			field.Type = value
		case "default":
			field.Default = value
		case "desc":
			field.Description = value
		case "required":
			field.Required = value
		case "placeholder":
			field.Placeholder = value
		default:
			return Field{}, fmt.Errorf("unsupported attribute key %q", key)
		}
	}

	return field, nil
}

func validate(spec Spec) error {
	seenShort := map[string]string{}
	seenLong := map[string]string{}
	repeatableSeen := false

	for _, field := range spec.Fields {
		if field.Name == "" {
			return fmt.Errorf("field name must not be empty")
		}

		if field.Short != "" {
			if len([]rune(field.Short)) != 1 {
				return fmt.Errorf("field %q short option must be one rune", field.Name)
			}
			if prev, ok := seenShort[field.Short]; ok {
				return fmt.Errorf("short option %q reused by %q and %q", field.Short, prev, field.Name)
			}
			seenShort[field.Short] = field.Name
		}

		if field.Kind != FieldArg {
			if field.Long == "" {
				return fmt.Errorf("field %q long option must not be empty", field.Name)
			}
			if prev, ok := seenLong[field.Long]; ok {
				return fmt.Errorf("long option %q reused by %q and %q", field.Long, prev, field.Name)
			}
			seenLong[field.Long] = field.Name
		}

		if field.Kind == FieldFlag && field.Repeatable {
			return fmt.Errorf("field %q repeatable flags are not supported", field.Name)
		}

		if field.Kind == FieldFlag && field.Type != "bool" {
			return fmt.Errorf("field %q flags must use type bool", field.Name)
		}

		switch field.Type {
		case "string", "bool", "int", "int64", "uint", "float64", "duration", "file":
		default:
			return fmt.Errorf("field %q has unsupported type %q", field.Name, field.Type)
		}

		if field.Kind == FieldArg && repeatableSeen {
			return fmt.Errorf("arg %q appears after a repeatable positional arg; repeatable positional args must be last", field.Name)
		}
		if field.Kind == FieldArg && field.Repeatable {
			repeatableSeen = true
		}
	}

	return nil
}

func kebabCase(name string) string {
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return strings.ToLower(name)
}

func defaultPlaceholder(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return strings.ToUpper(name)
}
