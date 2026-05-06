package cli

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"unicode"

	"golang.org/x/term"
)

type tableAlignment int

const (
	tableAlignLeft tableAlignment = iota
	tableAlignRight
)

var tableTerminalWidth = detectTableTerminalWidth
var tableWriterIsTerminal = detectTableWriterIsTerminal
var tableRuntimeGOOS = runtime.GOOS

func renderFieldValueTable(stdout io.Writer, opts *Options, fields map[string]string) error {
	rows := make([][]string, 0, len(fields))
	for _, field := range sortedKeys(fields) {
		rows = append(rows, []string{field, fields[field]})
	}
	return renderTable(stdout, opts, []string{"Field", "Value"}, rows, 16, false)
}

func renderTable(stdout io.Writer, opts *Options, headers []string, rows [][]string, minWidth int, printEmpty bool) error {
	return renderTableAligned(stdout, opts, headers, rows, nil, minWidth, printEmpty)
}

func renderTableAligned(stdout io.Writer, opts *Options, headers []string, rows [][]string, alignments []tableAlignment, minWidth int, printEmpty bool) error {
	if len(rows) == 0 && !printEmpty {
		_, err := fmt.Fprintln(stdout)
		return err
	}

	widths := naturalTableWidths(headers, rows)
	if target, ok := tableTargetWidth(stdout, opts); ok {
		widths = assignCliffStyleMaxWidths(widths, target, minWidth)
	}

	return writeTable(stdout, headers, rows, widths, alignments)
}

func naturalTableWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = maxLineWidth(header)
	}
	for _, row := range rows {
		for i := range headers {
			if i >= len(row) {
				continue
			}
			widths[i] = max(widths[i], maxLineWidth(row[i]))
		}
	}
	return widths
}

func tableTargetWidth(stdout io.Writer, opts *Options) (int, bool) {
	if opts != nil && opts.MaxWidth > 0 {
		return opts.MaxWidth, true
	}
	fitWidth := false
	if opts != nil {
		fitWidth = opts.FitWidth
	}
	if !shouldFitTable(stdout, fitWidth) {
		return 0, false
	}
	width, ok := tableTerminalWidth(stdout)
	if !ok || width <= 0 {
		return 0, false
	}
	return width, true
}

func shouldFitTable(stdout io.Writer, fitWidth bool) bool {
	if tableRuntimeGOOS == "windows" {
		return fitWidth
	}
	return tableWriterIsTerminal(stdout) || fitWidth
}

func detectTableTerminalWidth(stdout io.Writer) (int, bool) {
	file, ok := stdout.(*os.File)
	if !ok {
		return 0, false
	}
	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil {
		return 0, false
	}
	return width, true
}

func detectTableWriterIsTerminal(stdout io.Writer) bool {
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func assignCliffStyleMaxWidths(widths []int, termWidth int, minWidth int) []int {
	if termWidth <= 0 || tableDisplayWidth(widths) <= termWidth {
		return widths
	}

	fieldCount := len(widths)
	usableTotalWidth, optimalWidth := cliffWidthInfo(termWidth, fieldCount)
	shrinkFields, shrinkRemaining := cliffShrinkFields(usableTotalWidth, optimalWidth, widths)
	if len(shrinkFields) == 0 {
		return widths
	}

	fitted := append([]int(nil), widths...)
	shrinkTo := shrinkRemaining / len(shrinkFields)
	for _, field := range shrinkFields[:len(shrinkFields)-1] {
		fitted[field] = max(minWidth, shrinkTo)
		shrinkRemaining -= shrinkTo
	}
	fitted[shrinkFields[len(shrinkFields)-1]] = max(minWidth, shrinkRemaining)
	return fitted
}

func cliffWidthInfo(termWidth int, fieldCount int) (int, int) {
	usableTotalWidth := max(0, termWidth-1-3*fieldCount)
	if fieldCount == 0 {
		return usableTotalWidth, 0
	}
	return usableTotalWidth, usableTotalWidth / fieldCount
}

func cliffShrinkFields(usableTotalWidth int, optimalWidth int, widths []int) ([]int, int) {
	var shrinkFields []int
	shrinkRemaining := usableTotalWidth
	for i, width := range widths {
		if width <= optimalWidth {
			shrinkRemaining -= width
			continue
		}
		shrinkFields = append(shrinkFields, i)
	}
	return shrinkFields, shrinkRemaining
}

func tableDisplayWidth(widths []int) int {
	return 1 + 3*len(widths) + sumInts(widths)
}

func writeTable(stdout io.Writer, headers []string, rows [][]string, widths []int, alignments []tableAlignment) error {
	if err := writeTableBorder(stdout, widths); err != nil {
		return err
	}
	if err := writeTableRow(stdout, headers, widths, alignments); err != nil {
		return err
	}
	if err := writeTableBorder(stdout, widths); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeTableRow(stdout, row, widths, alignments); err != nil {
			return err
		}
	}
	return writeTableBorder(stdout, widths)
}

func writeTableBorder(stdout io.Writer, widths []int) error {
	if _, err := fmt.Fprint(stdout, "+"); err != nil {
		return err
	}
	for _, width := range widths {
		if _, err := fmt.Fprintf(stdout, "%s+", strings.Repeat("-", width+2)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(stdout)
	return err
}

func writeTableRow(stdout io.Writer, row []string, widths []int, alignments []tableAlignment) error {
	wrapped := make([][]string, len(widths))
	height := 1
	for i, width := range widths {
		value := ""
		if i < len(row) {
			value = row[i]
		}
		wrapped[i] = wrapTableCell(value, width)
		height = max(height, len(wrapped[i]))
	}

	for line := 0; line < height; line++ {
		if _, err := fmt.Fprint(stdout, "|"); err != nil {
			return err
		}
		for i, width := range widths {
			value := ""
			if line < len(wrapped[i]) {
				value = wrapped[i][line]
			}
			value = alignTableCell(value, width, alignmentAt(alignments, i))
			if _, err := fmt.Fprintf(stdout, " %s |", value); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(stdout); err != nil {
			return err
		}
	}
	return nil
}

func alignmentAt(alignments []tableAlignment, index int) tableAlignment {
	if index < len(alignments) {
		return alignments[index]
	}
	return tableAlignLeft
}

func alignTableCell(value string, width int, alignment tableAlignment) string {
	if alignment == tableAlignRight {
		return padLeft(value, width)
	}
	return padRight(value, width)
}

func wrapTableCell(value string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", " ")
	lines := strings.Split(value, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapTableLine(line, width)...)
	}
	if len(wrapped) == 0 {
		return []string{""}
	}
	return wrapped
}

func wrapTableLine(line string, width int) []string {
	line = strings.TrimRightFunc(line, unicode.IsSpace)
	if line == "" {
		return []string{""}
	}

	var wrapped []string
	for displayWidth(line) > width {
		head, tail := splitTableLine(line, width)
		wrapped = append(wrapped, head)
		line = strings.TrimLeftFunc(tail, unicode.IsSpace)
		if line == "" {
			return wrapped
		}
	}
	return append(wrapped, line)
}

func splitTableLine(line string, width int) (string, string) {
	runes := []rune(line)
	if len(runes) <= width {
		return line, ""
	}

	splitAt := width
	for i := min(width, len(runes)) - 1; i > 0; i-- {
		if unicode.IsSpace(runes[i]) {
			splitAt = i
			break
		}
	}
	if splitAt == 0 {
		splitAt = width
	}

	head := strings.TrimRightFunc(string(runes[:splitAt]), unicode.IsSpace)
	tail := string(runes[splitAt:])
	if head == "" {
		head = string(runes[:width])
		tail = string(runes[width:])
	}
	return head, tail
}

func padRight(value string, width int) string {
	padding := width - displayWidth(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func padLeft(value string, width int) string {
	padding := width - displayWidth(value)
	if padding <= 0 {
		return value
	}
	return strings.Repeat(" ", padding) + value
}

func maxLineWidth(value string) int {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", " ")
	width := 0
	for _, line := range strings.Split(value, "\n") {
		width = max(width, displayWidth(line))
	}
	return width
}

func displayWidth(value string) int {
	return len([]rune(value))
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
