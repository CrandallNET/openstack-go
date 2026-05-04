package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/netip"
	"os"
	"reflect"
	"regexp"
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

type prettyValueFormatter interface {
	PrettyString() string
}

type prettyCellColorizer func(rowIndex int, columnIndex int, text string) string

type orderedJSONObject struct {
	keys   []string
	values map[string]any
}

var (
	prettyUUIDPattern        = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	prettyIPCandidatePattern = regexp.MustCompile(`(?i)(?:\b(?:\d{1,3}\.){3}\d{1,3}(?:/\d{1,3})?\b|\b[0-9a-f]*:[0-9a-f:.]*(?:/\d{1,3})?\b|::(?:/\d{1,3})?)`)

	prettyBooleanFalseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	prettyBooleanTrueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	prettyErrorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	prettyFieldStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	prettyFlavorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true)
	prettyIPStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true)
	prettyLabelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	prettyLabelValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	prettyNAStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	prettyNameStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	prettyNumberStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	prettyUUIDStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	prettyWarningStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
)

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
	return renderPrettyTable(stdout, opts, columns, tableRows, prettyListCellColorizer(columns))
}

func renderPrettyShow(stdout io.Writer, opts *Options, fields []outputField) error {
	rows := make([]table.Row, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, table.Row{field.Name, prettyCellValue(field.Value)})
	}
	return renderPrettyTable(stdout, opts, []string{"Field", "Value"}, rows, prettyShowCellColorizer(fields))
}

func renderPrettyTable(stdout io.Writer, opts *Options, headers []string, rows []table.Row, colorizer prettyCellColorizer) error {
	if len(rows) == 0 {
		return renderPrettyEmpty(stdout)
	}

	color := prettyColorEnabled(stdout)
	termWidth := prettyOutputWidth(stdout, opts, color)
	columns := prettyTableColumns(headers, rows, termWidth, color)
	if !color {
		colorizer = nil
	}
	wrappedRows := prettyWrapRows(rows, columns, colorizer)
	model := table.New(
		table.WithColumns(columns),
		table.WithRows(wrappedRows),
		table.WithWidth(prettyTableViewWidth(columns)),
		table.WithHeight(len(wrappedRows)+1),
		table.WithFocused(false),
		table.WithStyles(prettyTableStyles(color)),
	)

	view := prettyAddHeaderSpacer(strings.TrimRight(model.View(), "\n"))
	if color {
		view = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(view)
	}
	_, err := fmt.Fprintln(stdout, view)
	return err
}

func prettyAddHeaderSpacer(view string) string {
	headerEnd := strings.IndexByte(view, '\n')
	if headerEnd == -1 {
		return view
	}
	return view[:headerEnd+1] + "\n" + view[headerEnd+1:]
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
		widths[i] = min(max(maxLineWidth(header), 4), 64)
	}
	for _, row := range rows {
		for i := range headers {
			if i >= len(row) {
				continue
			}
			widths[i] = min(max(widths[i], maxLineWidth(row[i])), 64)
		}
	}
	return widths
}

func prettyWrapRows(rows []table.Row, columns []table.Column, colorizer prettyCellColorizer) []table.Row {
	wrappedRows := make([]table.Row, 0, len(rows)*2)
	for rowIndex, row := range rows {
		cellLines := make([][]string, len(columns))
		height := 1
		for i, column := range columns {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			cellLines[i] = wrapTableCell(value, column.Width)
			if colorizer != nil {
				cellLines[i] = prettyColorizeWrappedCellLines(rowIndex, i, cellLines[i], colorizer)
			}
			height = max(height, len(cellLines[i]))
		}
		for line := 0; line < height; line++ {
			wrappedRow := make(table.Row, len(columns))
			for column := range columns {
				if line < len(cellLines[column]) {
					wrappedRow[column] = cellLines[column][line]
				}
			}
			wrappedRows = append(wrappedRows, wrappedRow)
		}
		if rowIndex < len(rows)-1 {
			wrappedRows = append(wrappedRows, make(table.Row, len(columns)))
		}
	}
	return wrappedRows
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

func prettyListCellColorizer(columns []string) prettyCellColorizer {
	return func(rowIndex int, columnIndex int, text string) string {
		if columnIndex >= len(columns) {
			return prettyColorizeTokens(text)
		}
		return prettyColorizeByName(columns[columnIndex], text)
	}
}

func prettyShowCellColorizer(fields []outputField) prettyCellColorizer {
	return func(rowIndex int, columnIndex int, text string) string {
		if columnIndex == 0 {
			return prettyFieldStyle.Render(text)
		}
		if columnIndex == 1 && rowIndex < len(fields) {
			return prettyColorizeByName(fields[rowIndex].Name, text)
		}
		return prettyColorizeTokens(text)
	}
}

func prettyColorizeByName(name string, text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	normalized := normalizeColumnName(name)
	return prettyColorizeLabeledText(text, func(value string) string {
		return prettyColorizeByNormalizedName(normalized, value)
	})
}

func prettyColorizeByNormalizedName(normalized string, text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	if prettyContainsAddressToken(text) {
		return prettyColorizeTokens(text)
	}
	switch {
	case prettyIsNameField(normalized):
		return prettyNameStyle.Render(text)
	case prettyIsFlavorField(normalized):
		return prettyFlavorStyle.Render(text)
	case prettyIsImageField(normalized):
		return prettyColorizeImage(text)
	case prettyIsFlavorComponentField(normalized):
		return prettyNumberStyle.Render(prettyColorizeTokens(text))
	case prettyIsStatusField(normalized):
		return prettyColorizeStatus(text)
	case prettyIsBooleanText(text):
		return prettyColorizeBoolean(text)
	default:
		return prettyColorizeTokens(text)
	}
}

func prettyIsNameField(name string) bool {
	return name == "name" ||
		name == "display_name" ||
		strings.HasSuffix(name, "_name") ||
		strings.HasSuffix(name, "_hostname") ||
		name == "hostname" ||
		name == "hypervisor_hostname"
}

func prettyIsFlavorField(name string) bool {
	return name == "flavor" ||
		name == "flavor_id" ||
		name == "flavor_name" ||
		strings.Contains(name, "flavor")
}

func prettyIsImageField(name string) bool {
	return name == "image" ||
		name == "image_id" ||
		name == "image_name" ||
		strings.Contains(name, "image")
}

func prettyIsFlavorComponentField(name string) bool {
	switch name {
	case "ram", "disk", "ephemeral", "vcpus", "swap", "rxtx_factor":
		return true
	default:
		return false
	}
}

func prettyIsStatusField(name string) bool {
	switch name {
	case "status", "state", "power_state", "task_state", "vm_state":
		return true
	default:
		return false
	}
}

func prettyIsBooleanText(text string) bool {
	trimmed := strings.TrimSpace(text)
	return trimmed == "True" || trimmed == "False"
}

func prettyColorizeBoolean(text string) string {
	switch strings.TrimSpace(text) {
	case "True":
		return prettyBooleanTrueStyle.Render(text)
	case "False":
		return prettyBooleanFalseStyle.Render(text)
	default:
		return text
	}
}

func prettyColorizeStatus(text string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(text))
	switch trimmed {
	case "ACTIVE", "AVAILABLE", "ENABLED", "HEALTHY", "IN_USE", "ONLINE", "READY", "RUNNING", "UP":
		return prettyBooleanTrueStyle.Render(text)
	case "BUILD", "CREATING", "DELETING", "MIGRATING", "PENDING", "REBOOT", "REBUILD", "RESIZE", "SAVING", "VERIFY_RESIZE":
		return prettyWarningStyle.Render(text)
	case "DELETED", "DISABLED", "DOWN", "ERROR", "FAILED", "FAULT", "KILLED", "SHUTOFF", "SUSPENDED":
		return prettyErrorStyle.Render(text)
	default:
		return prettyColorizeTokens(text)
	}
}

func prettyColorizeTokens(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	return prettyColorizeLabeledText(text, prettyColorizeBareTokens)
}

func prettyColorizeBareTokens(text string) string {
	text = prettyIPCandidatePattern.ReplaceAllStringFunc(text, func(candidate string) string {
		if !prettyValidIPToken(candidate) {
			return candidate
		}
		return prettyIPStyle.Render(candidate)
	})
	text = prettyUUIDPattern.ReplaceAllStringFunc(text, func(candidate string) string {
		return prettyColorizeUUID(candidate)
	})
	return text
}

func prettyColorizeLabeledText(text string, colorizeValue func(string) string) string {
	prefix, label, value, ok := prettySplitLabelPrefix(text)
	if !ok {
		return colorizeValue(text)
	}
	return prefix + prettyLabelStyle.Render(label) + prettyColorizeLabelValue(value)
}

func prettyColorizeWrappedCellLines(rowIndex int, columnIndex int, lines []string, colorizer prettyCellColorizer) []string {
	colored := make([]string, len(lines))
	pendingLabelValue := false
	for index, line := range lines {
		if line == "" {
			pendingLabelValue = false
			colored[index] = line
			continue
		}
		if prettyLineIsListMarker(line) {
			pendingLabelValue = false
			colored[index] = colorizer(rowIndex, columnIndex, line)
			continue
		}
		_, _, value, ok := prettySplitLabelPrefix(line)
		if ok {
			pendingLabelValue = strings.TrimSpace(value) == ""
			colored[index] = colorizer(rowIndex, columnIndex, line)
			continue
		}
		if pendingLabelValue {
			colored[index] = prettyColorizeLabelValue(line)
			continue
		}
		colored[index] = colorizer(rowIndex, columnIndex, line)
	}
	return colored
}

func prettyColorizeLabelValue(text string) string {
	if text == "" {
		return text
	}
	return prettyLabelValueStyle.Render(text)
}

func prettyLineIsListMarker(text string) bool {
	return strings.TrimSpace(text) == "-"
}

func prettySplitLabelPrefix(text string) (string, string, string, bool) {
	prefixLength := len(text) - len(strings.TrimLeft(text, " "))
	prefix := text[:prefixLength]
	remainder := text[prefixLength:]
	if strings.HasPrefix(remainder, "- ") {
		prefix += "- "
		remainder = remainder[2:]
	}

	colon := strings.IndexByte(remainder, ':')
	if colon <= 0 {
		return "", "", "", false
	}
	if colon+1 < len(remainder) && remainder[colon+1] != ' ' {
		return "", "", "", false
	}

	labelText := strings.TrimSpace(remainder[:colon])
	if labelText == "" {
		return "", "", "", false
	}
	return prefix, remainder[:colon+1], remainder[colon+1:], true
}

func prettyColorizeImage(text string) string {
	return strings.ReplaceAll(prettyColorizeTokens(text), "N/A", prettyNAStyle.Render("N/A"))
}

func prettyColorizeUUID(uuid string) string {
	parts := strings.Split(uuid, "-")
	for index, part := range parts {
		parts[index] = prettyUUIDStyle.Render(part)
	}
	return strings.Join(parts, "-")
}

func prettyContainsAddressToken(text string) bool {
	if prettyUUIDPattern.MatchString(text) {
		return true
	}
	for _, candidate := range prettyIPCandidatePattern.FindAllString(text, -1) {
		if prettyValidIPToken(candidate) {
			return true
		}
	}
	return false
}

func prettyValidIPToken(candidate string) bool {
	if strings.Contains(candidate, "/") {
		_, err := netip.ParsePrefix(candidate)
		return err == nil
	}
	_, err := netip.ParseAddr(candidate)
	return err == nil
}

func prettyCellValue(value any) string {
	text := prettyValueString(value)
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	text = strings.ReplaceAll(text, "\t", "  ")
	return strings.TrimSpace(text)
}

func prettyValueString(value any) string {
	if formatter, ok := value.(prettyValueFormatter); ok {
		return formatter.PrettyString()
	}
	if text, ok := prettyStructuredString(value); ok {
		return text
	}
	if text, ok := prettyJSONText(value); ok {
		return text
	}
	return valueString(value)
}

func prettyJSONText(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !(strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
		return "", false
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	decoded, err := decodeOrderedJSONValue(decoder)
	if err != nil {
		return "", false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", false
	}
	return prettyStructuredString(decoded)
}

func prettyStructuredString(value any) (string, bool) {
	lines, ok := prettyStructuredLines(value, 0)
	if !ok {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

func prettyStructuredLines(value any, indent int) ([]string, bool) {
	switch typed := value.(type) {
	case orderedJSONObject:
		if len(typed.keys) == 0 {
			return []string{indentString(indent) + "{}"}, true
		}
		lines := make([]string, 0, len(typed.keys))
		for _, key := range typed.keys {
			lines = appendPrettyMapEntry(lines, indent, key, typed.values[key])
		}
		return lines, true
	case map[string]any:
		if len(typed) == 0 {
			return []string{indentString(indent) + "{}"}, true
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			lines = appendPrettyMapEntry(lines, indent, key, typed[key])
		}
		return lines, true
	case map[string]string:
		if len(typed) == 0 {
			return []string{indentString(indent) + "{}"}, true
		}
		lines := make([]string, 0, len(typed))
		for _, key := range sortedKeys(typed) {
			lines = appendPrettyMapEntry(lines, indent, key, typed[key])
		}
		return lines, true
	case []any:
		return prettyListLines(typed, indent), true
	case []string:
		values := make([]any, len(typed))
		for i, item := range typed {
			values[i] = item
		}
		return prettyListLines(values, indent), true
	default:
		reflected := reflect.ValueOf(value)
		if !reflected.IsValid() {
			return nil, false
		}
		switch reflected.Kind() {
		case reflect.Interface, reflect.Pointer:
			if reflected.IsNil() {
				return nil, false
			}
			return prettyStructuredLines(reflected.Elem().Interface(), indent)
		case reflect.Struct:
			if _, ok := value.(time.Time); ok {
				return nil, false
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, false
			}
			decoder := json.NewDecoder(bytes.NewReader(encoded))
			decoder.UseNumber()
			decoded, err := decodeOrderedJSONValue(decoder)
			if err != nil {
				return nil, false
			}
			return prettyStructuredLines(decoded, indent)
		case reflect.Map, reflect.Array, reflect.Slice:
			if reflected.Kind() != reflect.Array && reflected.IsNil() {
				return nil, false
			}
			if reflected.Type().Elem().Kind() == reflect.Uint8 {
				return nil, false
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, false
			}
			decoder := json.NewDecoder(bytes.NewReader(encoded))
			decoder.UseNumber()
			decoded, err := decodeOrderedJSONValue(decoder)
			if err != nil {
				return nil, false
			}
			return prettyStructuredLines(decoded, indent)
		default:
			return nil, false
		}
	}
}

func appendPrettyMapEntry(lines []string, indent int, key string, value any) []string {
	prefix := indentString(indent)
	if childLines, ok := prettyStructuredLines(value, indent+2); ok {
		if len(childLines) == 1 && strings.TrimSpace(childLines[0]) == "{}" {
			return append(lines, fmt.Sprintf("%s%s: {}", prefix, key))
		}
		if len(childLines) == 1 && strings.TrimSpace(childLines[0]) == "[]" {
			return append(lines, fmt.Sprintf("%s%s: []", prefix, key))
		}
		lines = append(lines, fmt.Sprintf("%s%s:", prefix, key))
		return append(lines, childLines...)
	}
	scalar := prettyScalarString(value)
	scalarLines := strings.Split(strings.ReplaceAll(scalar, "\r\n", "\n"), "\n")
	if len(scalarLines) == 1 {
		return append(lines, fmt.Sprintf("%s%s: %s", prefix, key, scalarLines[0]))
	}
	lines = append(lines, fmt.Sprintf("%s%s:", prefix, key))
	for _, line := range scalarLines {
		lines = append(lines, indentString(indent+2)+line)
	}
	return lines
}

func prettyListLines(values []any, indent int) []string {
	if len(values) == 0 {
		return []string{indentString(indent) + "[]"}
	}
	lines := make([]string, 0, len(values))
	prefix := indentString(indent)
	for _, value := range values {
		if childLines, ok := prettyStructuredLines(value, indent+2); ok {
			lines = append(lines, prefix+"-")
			lines = append(lines, childLines...)
			continue
		}
		scalar := prettyScalarString(value)
		scalarLines := strings.Split(strings.ReplaceAll(scalar, "\r\n", "\n"), "\n")
		lines = append(lines, prefix+"- "+scalarLines[0])
		for _, line := range scalarLines[1:] {
			lines = append(lines, indentString(indent+2)+line)
		}
	}
	return lines
}

func prettyScalarString(value any) string {
	if text, ok := prettyJSONText(value); ok {
		return text
	}
	return valueString(value)
}

func indentString(width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(" ", width)
}

func prettyOutput(opts *Options) bool {
	return opts != nil && opts.Format == "pretty"
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
