package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	yaml "gopkg.in/yaml.v2"
)

type outputRow map[string]any

type outputField struct {
	Name  string
	Value any
}

func renderListOutput(stdout io.Writer, opts *Options, columns []string, rows []outputRow) error {
	if rows == nil {
		rows = []outputRow{}
	}
	columns = selectColumns(columns, opts.Columns)
	sortRows(rows, columns, opts)

	switch opts.Format {
	case "json":
		return writeJSONRows(stdout, columns, rows, opts.NoIndent)
	case "yaml":
		bytes, err := yaml.Marshal(rows)
		if err != nil {
			return err
		}
		_, err = stdout.Write(bytes)
		return err
	case "csv":
		writer := csv.NewWriter(stdout)
		if err := writer.Write(columns); err != nil {
			return err
		}
		for _, row := range rows {
			if err := writer.Write(rowStrings(columns, row)); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case "value":
		for _, row := range rows {
			if _, err := fmt.Fprintln(stdout, strings.Join(rowStrings(columns, row), " ")); err != nil {
				return err
			}
		}
		return nil
	case "pretty":
		return renderPrettyList(stdout, columns, rows)
	default:
		tableRows := make([][]string, 0, len(rows))
		for _, row := range rows {
			tableRows = append(tableRows, rowStrings(columns, row))
		}
		return renderTable(stdout, opts, columns, tableRows, 8, opts.PrintEmpty)
	}
}

func renderShowOutput(stdout io.Writer, opts *Options, fields []outputField) error {
	fields = selectFields(fields, opts.Columns)

	switch opts.Format {
	case "json":
		return writeJSONFields(stdout, fields, opts.NoIndent)
	case "yaml":
		bytes, err := yaml.Marshal(fieldsToMap(fields))
		if err != nil {
			return err
		}
		_, err = stdout.Write(bytes)
		return err
	case "shell":
		for _, field := range fields {
			if _, err := fmt.Fprintf(stdout, "%s%s=\"%s\"\n", opts.Prefix, shellName(field.Name), shellEscape(valueString(field.Value))); err != nil {
				return err
			}
		}
		return nil
	case "value":
		for _, field := range fields {
			if _, err := fmt.Fprintln(stdout, valueString(field.Value)); err != nil {
				return err
			}
		}
		return nil
	case "pretty":
		for _, field := range fields {
			if _, err := fmt.Fprintf(stdout, "%s: %s\n", field.Name, valueString(field.Value)); err != nil {
				return err
			}
		}
		return nil
	default:
		rows := make([][]string, 0, len(fields))
		for _, field := range fields {
			rows = append(rows, []string{field.Name, valueString(field.Value)})
		}
		return renderTable(stdout, opts, []string{"Field", "Value"}, rows, 16, opts.PrintEmpty)
	}
}

func renderPrettyList(stdout io.Writer, columns []string, rows []outputRow) error {
	for i, row := range rows {
		if i > 0 {
			if _, err := fmt.Fprintln(stdout); err != nil {
				return err
			}
		}
		for _, column := range columns {
			if _, err := fmt.Fprintf(stdout, "%s: %s\n", column, valueString(row[column])); err != nil {
				return err
			}
		}
	}
	return nil
}

func selectColumns(columns []string, requested []string) []string {
	if len(requested) == 0 {
		return columns
	}
	selected := make([]string, 0, len(requested))
	for _, want := range requested {
		normalizedWant := normalizeColumnName(want)
		for _, column := range columns {
			if normalizeColumnName(column) == normalizedWant {
				selected = append(selected, column)
				break
			}
		}
	}
	if len(selected) == 0 {
		return columns
	}
	return selected
}

func selectFields(fields []outputField, requested []string) []outputField {
	if len(requested) == 0 {
		return fields
	}
	var selected []outputField
	for _, want := range requested {
		normalizedWant := normalizeColumnName(want)
		for _, field := range fields {
			if normalizeColumnName(field.Name) == normalizedWant {
				selected = append(selected, field)
				break
			}
		}
	}
	if len(selected) == 0 {
		return fields
	}
	return selected
}

func sortRows(rows []outputRow, columns []string, opts *Options) {
	sortColumns := opts.SortColumns
	if len(sortColumns) == 0 {
		return
	}
	selected := selectColumns(columns, sortColumns)
	descending := opts.SortDescending && !opts.SortAscending
	sort.SliceStable(rows, func(i int, j int) bool {
		for _, column := range selected {
			left := valueString(rows[i][column])
			right := valueString(rows[j][column])
			if left == right {
				continue
			}
			if descending {
				return left > right
			}
			return left < right
		}
		return false
	})
}

func rowStrings(columns []string, row outputRow) []string {
	values := make([]string, len(columns))
	for i, column := range columns {
		values[i] = valueString(row[column])
	}
	return values
}

func writeJSONRows(stdout io.Writer, columns []string, rows []outputRow, noIndent bool) error {
	if noIndent {
		if _, err := fmt.Fprint(stdout, "["); err != nil {
			return err
		}
		for i, row := range rows {
			if i > 0 {
				if _, err := fmt.Fprint(stdout, ","); err != nil {
					return err
				}
			}
			if err := writeJSONObject(stdout, columns, func(column string) any { return row[column] }, "", "", noIndent); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(stdout, "]")
		return err
	}

	if len(rows) == 0 {
		_, err := fmt.Fprintln(stdout, "[]")
		return err
	}
	if _, err := fmt.Fprintln(stdout, "["); err != nil {
		return err
	}
	for i, row := range rows {
		if _, err := fmt.Fprint(stdout, "  "); err != nil {
			return err
		}
		if err := writeJSONObject(stdout, columns, func(column string) any { return row[column] }, "  ", "  ", noIndent); err != nil {
			return err
		}
		if i < len(rows)-1 {
			if _, err := fmt.Fprint(stdout, ","); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(stdout); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(stdout, "]")
	return err
}

func writeJSONFields(stdout io.Writer, fields []outputField, noIndent bool) error {
	names := make([]string, 0, len(fields))
	values := make(map[string]any, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
		values[field.Name] = field.Value
	}
	if noIndent {
		if err := writeJSONObject(stdout, names, func(name string) any { return values[name] }, "", "", noIndent); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout)
		return err
	}
	if err := writeJSONObject(stdout, names, func(name string) any { return values[name] }, "", "  ", noIndent); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout)
	return err
}

func writeJSONObject(stdout io.Writer, names []string, value func(string) any, objectPrefix string, indent string, noIndent bool) error {
	if noIndent {
		if _, err := fmt.Fprint(stdout, "{"); err != nil {
			return err
		}
		for i, name := range names {
			if i > 0 {
				if _, err := fmt.Fprint(stdout, ","); err != nil {
					return err
				}
			}
			key, err := json.Marshal(name)
			if err != nil {
				return err
			}
			encoded, err := encodeJSONValue(value(name), "", "")
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "%s:%s", key, encoded); err != nil {
				return err
			}
		}
		_, err := fmt.Fprint(stdout, "}")
		return err
	}

	if _, err := fmt.Fprintln(stdout, "{"); err != nil {
		return err
	}
	for i, name := range names {
		key, err := json.Marshal(name)
		if err != nil {
			return err
		}
		encoded, err := encodeJSONValue(value(name), objectPrefix+indent, indent)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "%s%s%s: %s", objectPrefix, indent, key, encoded); err != nil {
			return err
		}
		if i < len(names)-1 {
			if _, err := fmt.Fprint(stdout, ","); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(stdout); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(stdout, "%s}", objectPrefix)
	return err
}

func encodeJSONValue(value any, prefix string, indent string) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if indent != "" {
		encoder.SetIndent(prefix, indent)
	}
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}

func fieldsToMap(fields []outputField) map[string]any {
	values := make(map[string]any, len(fields))
	for _, field := range fields {
		values[field.Name] = field.Value
	}
	return values
}

func oscTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format("2006-01-02T15:04:05.000000")
}

func valueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case []string:
		return strings.Join(typed, ", ")
	case []any:
		parts := make([]string, len(typed))
		for i, part := range typed {
			parts[i] = valueString(part)
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if len(typed) == 0 {
			return "{}"
		}
		bytes, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(bytes)
	case map[string]string:
		if len(typed) == 0 {
			return "{}"
		}
		bytes, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(bytes)
	case time.Time:
		if typed.IsZero() {
			return ""
		}
		return typed.UTC().Format(time.RFC3339)
	default:
		bytes, err := json.Marshal(typed)
		if err == nil && string(bytes) != "{}" {
			return strings.Trim(string(bytes), "\"")
		}
		return fmt.Sprint(typed)
	}
}

func normalizeColumnName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if unicode.IsSpace(r) || r == '-' {
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
			continue
		}
		b.WriteRune(r)
		lastUnderscore = false
	}
	return b.String()
}

func shellName(value string) string {
	return normalizeColumnName(value)
}

func shellEscape(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}
