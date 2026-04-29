package shellargs

import (
	"fmt"
	"strings"

	"shellargs/internal/spec"
)

type Spec = spec.Spec
type Field = spec.Field
type FieldKind = spec.FieldKind

const (
	FieldFlag   = spec.FieldFlag
	FieldOption = spec.FieldOption
	FieldArg    = spec.FieldArg
)

func ParseSpec(raw string) (Spec, error) {
	return spec.Parse(TrimSpecDoc(raw))
}

func TrimSpecDoc(raw string) string {
	lines := strings.Split(raw, "\n")
	first := -1
	last := -1

	for idx, line := range lines {
		if strings.TrimSpace(strings.TrimSuffix(line, "\r")) == "" {
			continue
		}
		if first < 0 {
			first = idx
		}
		last = idx
	}
	if first < 0 || first == last {
		return raw
	}
	if strings.TrimSpace(strings.TrimSuffix(lines[first], "\r")) != "@@@" {
		return raw
	}
	if strings.TrimSpace(strings.TrimSuffix(lines[last], "\r")) != "@@@" {
		return raw
	}

	return strings.Join(lines[first+1:last], "\n") + "\n"
}

func ZshCompletionScript(sp Spec, program string) (string, error) {
	if program == "" {
		program = sp.Name
	}
	if program == "" {
		return "", fmt.Errorf("program must not be empty")
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "#compdef %s\n\n", program)
	builder.WriteString(ZshCompletionBody(sp, ""))
	return builder.String(), nil
}

func ZshCompletionBody(sp Spec, indent string) string {
	specs := zshArgumentSpecs(sp.Fields)
	var builder strings.Builder
	if len(specs) == 0 {
		fmt.Fprintf(&builder, "%s_message 'no arguments'\n", indent)
	} else {
		fmt.Fprintf(&builder, "%s_arguments -s -S \\\n", indent)
		for idx, item := range specs {
			if idx == len(specs)-1 {
				fmt.Fprintf(&builder, "%s  %s\n", indent, zshQuote(item))
			} else {
				fmt.Fprintf(&builder, "%s  %s \\\n", indent, zshQuote(item))
			}
		}
	}
	return builder.String()
}

func zshArgumentSpecs(fields []Field) []string {
	var out []string
	position := 1

	for _, field := range fields {
		switch field.Kind {
		case FieldFlag:
			out = append(out, zshOptionSpecs(field, false)...)
		case FieldOption:
			out = append(out, zshOptionSpecs(field, true)...)
		case FieldArg:
			out = append(out, zshPositionalSpec(field, position))
			if !field.Repeatable {
				position++
			}
		}
	}

	return out
}

func zshOptionSpecs(field Field, takesValue bool) []string {
	var specs []string
	prefix := ""
	if field.Repeatable {
		prefix = "*"
	}

	for _, name := range zshOptionNames(field) {
		item := prefix + name + zshDescription(field.Description)
		if takesValue {
			item += ":" + zshPlaceholder(field) + ":" + zshAction(field.Type)
		}
		specs = append(specs, item)
	}

	return specs
}

func zshOptionNames(field Field) []string {
	var names []string
	if field.Short != "" {
		names = append(names, "-"+field.Short)
	}
	if field.Long != "" {
		names = append(names, "--"+field.Long)
	}
	return names
}

func zshPositionalSpec(field Field, position int) string {
	prefix := fmt.Sprintf("%d", position)
	if field.Repeatable {
		prefix = "*"
	}
	return prefix + ":" + zshPlaceholder(field) + ":" + zshAction(field.Type)
}

func zshDescription(value string) string {
	if value == "" {
		return ""
	}
	return "[" + zshSpecText(value) + "]"
}

func zshPlaceholder(field Field) string {
	if field.Placeholder != "" {
		return zshSpecText(field.Placeholder)
	}
	return zshSpecText(strings.ToUpper(field.Name))
}

func zshAction(fieldType string) string {
	if fieldType == "file" {
		return "_files"
	}
	return ""
}

func zshSpecText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`[`, `\[`,
		`]`, `\]`,
		`:`, `\:`,
	)
	return replacer.Replace(value)
}

func zshQuote(input string) string {
	return "'" + strings.ReplaceAll(input, "'", `'\''`) + "'"
}
