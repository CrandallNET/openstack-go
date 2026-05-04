package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	yaml "gopkg.in/yaml.v2"
)

type outputRow map[string]any

type outputField struct {
	Name  string
	Value any
}

type orderedJSONObject struct {
	keys   []string
	values map[string]any
}

func (object orderedJSONObject) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteByte('{')
	for i, key := range object.keys {
		if i > 0 {
			buffer.WriteByte(',')
		}
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		encodedValue, err := json.Marshal(object.values[key])
		if err != nil {
			return nil, err
		}
		buffer.Write(encodedKey)
		buffer.WriteByte(':')
		buffer.Write(encodedValue)
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
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
		return renderPrettyList(stdout, opts, columns, rows)
	default:
		tableRows := make([][]string, 0, len(rows))
		for _, row := range rows {
			tableRows = append(tableRows, rowStrings(columns, row))
		}
		return renderTableAligned(stdout, opts, columns, tableRows, listTableAlignments(columns, rows), 8, opts.PrintEmpty)
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
		return renderPrettyShow(stdout, opts, fields)
	default:
		rows := make([][]string, 0, len(fields))
		for _, field := range fields {
			rows = append(rows, []string{field.Name, valueString(field.Value)})
		}
		return renderTable(stdout, opts, []string{"Field", "Value"}, rows, 16, opts.PrintEmpty)
	}
}

func renderPrettyList(stdout io.Writer, opts *Options, columns []string, rows []outputRow) error {
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		tableRow := make(table.Row, 0, len(columns))
		for _, column := range columns {
			tableRow = append(tableRow, prettyCellValue(row[column]))
		}
		tableRows = append(tableRows, tableRow)
	}
	return renderPrettyTable(stdout, opts, columns, tableRows)
}

func renderPrettyShow(stdout io.Writer, opts *Options, fields []outputField) error {
	rows := make([]table.Row, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, table.Row{field.Name, prettyCellValue(field.Value)})
	}
	return renderPrettyTable(stdout, opts, []string{"Field", "Value"}, rows)
}

func renderPrettyTable(stdout io.Writer, opts *Options, headers []string, rows []table.Row) error {
	if len(rows) == 0 {
		return renderPrettyEmpty(stdout)
	}

	color := prettyColorEnabled(stdout)
	termWidth := prettyOutputWidth(stdout, opts, color)
	columns := prettyTableColumns(headers, rows, termWidth, color)
	model := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithWidth(prettyTableViewWidth(columns)),
		table.WithHeight(len(rows)+1),
		table.WithFocused(false),
		table.WithStyles(prettyTableStyles(color)),
	)

	view := strings.TrimRight(model.View(), "\n")
	if color {
		view = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(view)
	}
	_, err := fmt.Fprintln(stdout, view)
	return err
}

func renderPrettyEmpty(stdout io.Writer) error {
	color := prettyColorEnabled(stdout)
	if color {
		_, err := fmt.Fprintln(stdout, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("No rows"))
		return err
	}
	_, err := fmt.Fprintln(stdout, "No rows")
	return err
}

func renderPrettyProgress(stdout io.Writer, opts *Options, label string, percent float64) error {
	color := prettyColorEnabled(stdout)
	width := min(max(prettyOutputWidth(stdout, opts, color)-displayWidth(label)-4, 20), 80)
	percent = math.Max(0, math.Min(1, percent))

	options := []progress.Option{progress.WithWidth(width)}
	if color {
		options = append(options, progress.WithDefaultBlend())
	} else {
		options = append(options,
			progress.WithFillCharacters('=', '-'),
		)
	}

	model := progress.New(options...)
	if !color {
		model.FullColor = lipgloss.NoColor{}
		model.EmptyColor = lipgloss.NoColor{}
	}
	if label != "" {
		label += " "
	}
	_, err := fmt.Fprintf(stdout, "%s%s\n", label, model.ViewAs(percent))
	return err
}

func prettyTableColumns(headers []string, rows []table.Row, termWidth int, color bool) []table.Column {
	widths := prettyNaturalWidths(headers, rows)
	paddingWidth := 2 * len(widths)
	borderWidth := 0
	if color {
		borderWidth = 2
	}
	usableWidth := max(len(widths)*4, termWidth-paddingWidth-borderWidth)
	widths = prettyFitWidths(widths, prettyMinimumWidths(headers), usableWidth)

	columns := make([]table.Column, 0, len(headers))
	for i, header := range headers {
		columns = append(columns, table.Column{Title: header, Width: widths[i]})
	}
	return columns
}

func prettyTableViewWidth(columns []table.Column) int {
	width := 0
	for _, column := range columns {
		width += column.Width + 2
	}
	return width
}

func prettyNaturalWidths(headers []string, rows []table.Row) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = min(max(displayWidth(header), 4), 64)
	}
	for _, row := range rows {
		for i := range headers {
			if i >= len(row) {
				continue
			}
			widths[i] = min(max(widths[i], displayWidth(row[i])), 64)
		}
	}
	return widths
}

func prettyMinimumWidths(headers []string) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = min(max(displayWidth(header), 4), 12)
	}
	return widths
}

func prettyFitWidths(widths []int, minimums []int, usableWidth int) []int {
	fitted := append([]int(nil), widths...)
	for sumInts(fitted) > usableWidth {
		index := -1
		shrinkable := 0
		for i, width := range fitted {
			available := width - minimums[i]
			if available > shrinkable {
				index = i
				shrinkable = available
			}
		}
		if index == -1 {
			break
		}
		fitted[index]--
	}
	return fitted
}

func prettyTableStyles(color bool) table.Styles {
	cell := lipgloss.NewStyle().Padding(0, 1)
	if !color {
		return table.Styles{
			Header:   cell.Copy(),
			Cell:     cell.Copy(),
			Selected: lipgloss.NewStyle(),
		}
	}
	header := cell.Copy().Bold(true).Foreground(lipgloss.Color("39"))
	selected := lipgloss.NewStyle()
	return table.Styles{
		Header:   header,
		Cell:     cell.Foreground(lipgloss.Color("252")),
		Selected: selected.Foreground(lipgloss.Color("252")),
	}
}

func prettyOutputWidth(stdout io.Writer, opts *Options, color bool) int {
	if opts != nil && opts.MaxWidth > 0 {
		return opts.MaxWidth
	}
	if (opts != nil && opts.FitWidth) || color {
		if width, ok := tableTerminalWidth(stdout); ok && width > 0 {
			return width
		}
	}
	return 100
}

func prettyColorEnabled(stdout io.Writer) bool {
	return os.Getenv("NO_COLOR") == "" && os.Getenv("CLICOLOR") != "0" && tableWriterIsTerminal(stdout)
}

func prettyCellValue(value any) string {
	text := valueString(value)
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	text = strings.Join(strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '\t'
	}), " ")
	return strings.TrimSpace(text)
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

func listTableAlignments(columns []string, rows []outputRow) []tableAlignment {
	alignments := make([]tableAlignment, len(columns))
	for i, column := range columns {
		if numericListColumn(column, rows) {
			alignments[i] = tableAlignRight
		}
	}
	return alignments
}

func numericListColumn(column string, rows []outputRow) bool {
	if strings.EqualFold(column, "id") {
		return false
	}
	found := false
	for _, row := range rows {
		value := row[column]
		if value == nil {
			continue
		}
		if !isNumericValue(value) {
			return false
		}
		found = true
	}
	return found
}

func isNumericValue(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return true
	default:
		kind := reflect.ValueOf(value).Kind()
		return kind >= reflect.Int && kind <= reflect.Float64
	}
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
		return "None"
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case []string:
		if len(typed) == 0 {
			return "[]"
		}
		return strings.Join(typed, ", ")
	case []any:
		if len(typed) == 0 {
			return "[]"
		}
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
	case orderedJSONObject:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed.values)
		}
		return string(bytes)
	case time.Time:
		if typed.IsZero() {
			return "None"
		}
		return typed.UTC().Format(time.RFC3339)
	default:
		reflected := reflect.ValueOf(value)
		switch reflected.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if reflected.IsNil() {
				return "None"
			}
		}
		switch reflected.Kind() {
		case reflect.Map:
			if reflected.Len() == 0 {
				return "{}"
			}
		case reflect.Array, reflect.Slice:
			if reflected.Len() == 0 {
				return "[]"
			}
		}
		bytes, err := json.Marshal(typed)
		if err == nil {
			encoded := string(bytes)
			switch encoded {
			case "null":
				return "None"
			case "{}", "[]":
				return encoded
			}
			if unquoted, err := strconv.Unquote(encoded); err == nil {
				return unquoted
			}
			return encoded
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
