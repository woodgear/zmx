package engine

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	flags "github.com/jessevdk/go-flags"

	"shellargs/internal/spec"
)

var ErrHelpShown = fmt.Errorf("help shown")

type ParseOptions struct {
	Args     []string
	AutoHelp bool
	Stdout   io.Writer
}

type ParseResult struct {
	Values            map[string]any
	CompletionHandled bool
}

type Engine struct {
	spec             spec.Spec
	rootType         reflect.Type
	rootFields       []compiledField
	positionalField  *compiledField
	positionalFields []compiledField
}

type compiledField struct {
	specName string
	goName   string
}

type BashCompletionOptions struct {
	Runner     string
	Program    string
	SpecBase64 string
	Shell      string
}

func New(sp spec.Spec) (*Engine, error) {
	rootType, rootFields, positionalField, positionalFields, err := compile(sp)
	if err != nil {
		return nil, err
	}

	return &Engine{
		spec:             sp,
		rootType:         rootType,
		rootFields:       rootFields,
		positionalField:  positionalField,
		positionalFields: positionalFields,
	}, nil
}

func (e *Engine) Parse(opts ParseOptions) (ParseResult, error) {
	instance := reflect.New(e.rootType)
	parser := flags.NewParser(instance.Interface(), parserOptions(opts.AutoHelp))
	parser.Name = e.spec.Name
	parser.ShortDescription = e.spec.Summary
	parser.LongDescription = e.spec.Description
	parser.Usage = e.spec.Usage

	completionHandled := false
	parser.CompletionHandler = func(items []flags.Completion) {
		completionHandled = true
		for _, item := range items {
			fmt.Fprintln(opts.Stdout, item.Item)
		}
	}

	_, err := parser.ParseArgs(opts.Args)
	if err != nil {
		if ferr, ok := err.(*flags.Error); ok && ferr.Type == flags.ErrHelp {
			parser.WriteHelp(opts.Stdout)
			return ParseResult{}, ErrHelpShown
		}
		return ParseResult{}, err
	}
	if completionHandled {
		return ParseResult{CompletionHandled: true}, nil
	}

	return ParseResult{
		Values: flattenValues(instance.Elem(), e.rootFields, e.positionalField, e.positionalFields),
	}, nil
}

func (e *Engine) WriteHelp(w io.Writer) error {
	instance := reflect.New(e.rootType)
	parser := flags.NewParser(instance.Interface(), flags.HelpFlag|flags.PassDoubleDash)
	parser.Name = e.spec.Name
	parser.ShortDescription = e.spec.Summary
	parser.LongDescription = e.spec.Description
	parser.Usage = e.spec.Usage
	parser.WriteHelp(w)
	return nil
}

func BashCompletionScript(opts BashCompletionOptions) (string, error) {
	if opts.Shell != "bash" {
		return "", fmt.Errorf("unsupported shell %q; only bash is supported right now", opts.Shell)
	}
	if opts.Runner == "" {
		return "", fmt.Errorf("runner must not be empty")
	}
	if opts.Program == "" {
		return "", fmt.Errorf("program must not be empty")
	}
	if opts.SpecBase64 == "" {
		return "", fmt.Errorf("spec base64 must not be empty")
	}

	fn := "__shellargs_complete_" + shellIdentifier(opts.Program)
	runner := shellQuote(opts.Runner)
	program := shellQuote(opts.Program)
	specBase64 := shellQuote(opts.SpecBase64)

	return fmt.Sprintf(`%s() {
  local args=("${COMP_WORDS[@]:1:$COMP_CWORD}")
  local IFS=$'\n'
  COMPREPLY=($(GO_FLAGS_COMPLETION=1 %s parse --spec-base64 %s --prog %s -- "${args[@]}"))
  return 0
}

complete -F %s %s
`, fn, runner, specBase64, program, fn, program), nil
}

func parserOptions(autoHelp bool) flags.Options {
	var options flags.Options = flags.PassDoubleDash
	if autoHelp {
		options |= flags.HelpFlag
	}
	return options
}

func compile(sp spec.Spec) (reflect.Type, []compiledField, *compiledField, []compiledField, error) {
	var rootStructFields []reflect.StructField
	var rootFields []compiledField
	var positionalStructFields []reflect.StructField
	var positionalFields []compiledField

	usedNames := map[string]int{}

	for _, field := range sp.Fields {
		goName := uniqueFieldName(field.Name, usedNames)
		tag := buildFieldTag(field)
		rtype, err := fieldType(field)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		structField := reflect.StructField{
			Name: goName,
			Type: rtype,
			Tag:  reflect.StructTag(tag),
		}
		compiled := compiledField{
			specName: field.Name,
			goName:   goName,
		}

		if field.Kind == spec.FieldArg {
			positionalStructFields = append(positionalStructFields, structField)
			positionalFields = append(positionalFields, compiled)
			continue
		}

		rootStructFields = append(rootStructFields, structField)
		rootFields = append(rootFields, compiled)
	}

	var positionalField *compiledField
	if len(positionalStructFields) > 0 {
		argsType := reflect.StructOf(positionalStructFields)
		goName := uniqueFieldName("args", usedNames)
		rootStructFields = append(rootStructFields, reflect.StructField{
			Name: goName,
			Type: argsType,
			Tag:  reflect.StructTag(`positional-args:"yes"`),
		})
		positionalField = &compiledField{
			specName: "args",
			goName:   goName,
		}
	}

	return reflect.StructOf(rootStructFields), rootFields, positionalField, positionalFields, nil
}

func buildFieldTag(field spec.Field) string {
	parts := make([]string, 0, 8)

	if field.Short != "" {
		parts = append(parts, fmt.Sprintf(`short:%s`, strconv.Quote(field.Short)))
	}
	if field.Kind != spec.FieldArg && field.Long != "" {
		parts = append(parts, fmt.Sprintf(`long:%s`, strconv.Quote(field.Long)))
	}
	if field.Description != "" {
		parts = append(parts, fmt.Sprintf(`description:%s`, strconv.Quote(field.Description)))
	}
	if field.Default != "" {
		parts = append(parts, fmt.Sprintf(`default:%s`, strconv.Quote(field.Default)))
	}
	if field.Required != "" {
		parts = append(parts, fmt.Sprintf(`required:%s`, strconv.Quote(field.Required)))
	}

	if field.Kind == spec.FieldArg {
		name := field.Placeholder
		if name == "" {
			name = field.Name
		}
		parts = append(parts, fmt.Sprintf(`positional-arg-name:%s`, strconv.Quote(name)))
	} else if field.Kind == spec.FieldOption {
		name := field.Placeholder
		if name == "" {
			name = strings.ToUpper(field.Name)
		}
		parts = append(parts, fmt.Sprintf(`value-name:%s`, strconv.Quote(name)))
	}

	return strings.Join(parts, " ")
}

func fieldType(field spec.Field) (reflect.Type, error) {
	var base reflect.Type

	switch field.Type {
	case "string":
		base = reflect.TypeOf("")
	case "bool":
		base = reflect.TypeOf(false)
	case "int":
		base = reflect.TypeOf(int(0))
	case "int64":
		base = reflect.TypeOf(int64(0))
	case "uint":
		base = reflect.TypeOf(uint(0))
	case "float64":
		base = reflect.TypeOf(float64(0))
	case "duration":
		base = reflect.TypeOf(time.Duration(0))
	case "file":
		base = reflect.TypeOf(flags.Filename(""))
	default:
		return nil, fmt.Errorf("field %q uses unsupported type %q", field.Name, field.Type)
	}

	if field.Repeatable {
		return reflect.SliceOf(base), nil
	}

	return base, nil
}

func flattenValues(root reflect.Value, rootFields []compiledField, positionalField *compiledField, positionalFields []compiledField) map[string]any {
	out := make(map[string]any, len(rootFields)+len(positionalFields))

	for _, field := range rootFields {
		out[field.specName] = exportValue(root.FieldByName(field.goName))
	}

	if positionalField != nil {
		argsVal := root.FieldByName(positionalField.goName)
		for _, field := range positionalFields {
			out[field.specName] = exportValue(argsVal.FieldByName(field.goName))
		}
	}

	return out
}

func exportValue(value reflect.Value) any {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return reflect.MakeSlice(value.Type(), 0, 0).Interface()
	}
	return value.Interface()
}

func uniqueFieldName(name string, used map[string]int) string {
	candidate := exportName(name)
	if used[candidate] == 0 {
		used[candidate] = 1
		return candidate
	}

	index := used[candidate]
	used[candidate]++
	return fmt.Sprintf("%s%d", candidate, index)
}

func exportName(name string) string {
	var parts []string
	split := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})

	for _, part := range split {
		if part == "" {
			continue
		}
		part = strings.ToLower(part)
		parts = append(parts, strings.ToUpper(part[:1])+part[1:])
	}

	if len(parts) == 0 {
		return "Field"
	}

	out := strings.Join(parts, "")
	if unicode.IsDigit(rune(out[0])) {
		return "X" + out
	}
	return out
}

func shellIdentifier(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "shellargs"
	}
	if unicode.IsDigit(rune(out[0])) {
		return "_" + out
	}
	return out
}

func shellQuote(input string) string {
	return "'" + strings.ReplaceAll(input, "'", `'"'"'`) + "'"
}
