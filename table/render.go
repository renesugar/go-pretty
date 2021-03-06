package table

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/text"
	"github.com/jedib0t/go-pretty/util"
)

// Render renders the Table in a human-readable "pretty" format. Example:
//  ┌─────┬────────────┬───────────┬────────┬─────────────────────────────┐
//  │   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
//  │   1 │ Arya       │ Stark     │   3000 │                             │
//  │  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
//  │ 300 │ Tyrion     │ Lannister │   5000 │                             │
//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
//  │     │            │ TOTAL     │  10000 │                             │
//  └─────┴────────────┴───────────┴────────┴─────────────────────────────┘
func (t *Table) Render() string {
	t.initForRender()

	var out strings.Builder
	if t.numColumns > 0 {
		// top-most border
		t.renderRowsBorderTop(&out)

		// header rows
		t.renderRowsHeader(&out)

		// (data) rows
		t.renderRows(&out, t.getRowsSorted(), renderHint{})

		// footer rows
		t.renderRowsFooter(&out)

		// bottom-most border
		t.renderRowsBorderBottom(&out)

		// caption
		if t.caption != "" {
			out.WriteRune('\n')
			out.WriteString(t.caption)
		}
	}
	return t.render(&out)
}

func (t *Table) renderColumn(out *strings.Builder, rowNum int, row rowStr, colIdx int, maxColumnLength int, hint renderHint) {
	// when working on the first column, and autoIndex is true, insert a new
	// column with the row number on it.
	if colIdx == 0 && t.autoIndex {
		t.renderColumnAutoIndex(out, rowNum, hint)
	}

	// when working on column number 2 or more, render the column separator
	if colIdx > 0 {
		t.renderColumnSeparator(out, hint)
	}

	// extract the text, convert-case if not-empty and align horizontally
	var colStr string
	if colIdx < len(row) {
		colStr = t.getFormat(hint).Apply(row[colIdx])
	}
	colStr = t.getAlign(colIdx, hint).Apply(colStr, maxColumnLength)

	// pad both sides of the column (when not a separator row)
	if !hint.isSeparatorRow {
		colStr = t.style.Box.PaddingLeft + colStr + t.style.Box.PaddingRight
	}

	t.renderColumnColorized(out, rowNum, colIdx, colStr, hint)
}

func (t *Table) renderColumnAutoIndex(out *strings.Builder, rowNum int, hint renderHint) {
	var outAutoIndex strings.Builder
	outAutoIndex.Grow(t.maxColumnLengths[0])

	if rowNum < 0 {
		numChars := t.autoIndexVIndexMaxLength + utf8.RuneCountInString(t.style.Box.PaddingLeft) +
			utf8.RuneCountInString(t.style.Box.PaddingRight)
		outAutoIndex.WriteString(strings.Repeat(t.style.Box.MiddleHorizontal, numChars))
	} else {
		outAutoIndex.WriteString(t.style.Box.PaddingLeft)
		rowNumStr := fmt.Sprint(rowNum)
		if hint.isHeaderRow || hint.isFooterRow {
			rowNumStr = strings.Repeat(" ", t.autoIndexVIndexMaxLength)
		}
		outAutoIndex.WriteString(text.AlignRight.Apply(rowNumStr, t.autoIndexVIndexMaxLength))
		outAutoIndex.WriteString(t.style.Box.PaddingRight)
	}

	if t.style.Color.IndexColumn != nil {
		out.WriteString(t.style.Color.IndexColumn.Sprintf(outAutoIndex.String()))
	} else {
		out.WriteString(outAutoIndex.String())
	}
	t.renderColumnSeparator(out, hint)
}

func (t *Table) renderColumnColorized(out *strings.Builder, rowNum int, colIdx int, colStr string, hint renderHint) {
	colors := t.getColors(hint)
	if colIdx < len(colors) && colors[colIdx] != nil {
		out.WriteString(colors[colIdx].Sprint(colStr))
	} else if hint.isHeaderRow && t.style.Color.Header != nil {
		out.WriteString(t.style.Color.Header.Sprint(colStr))
	} else if hint.isFooterRow && t.style.Color.Footer != nil {
		out.WriteString(t.style.Color.Footer.Sprint(colStr))
	} else if hint.isRegularRow() {
		if colIdx == t.indexColumn-1 && t.style.Color.IndexColumn != nil {
			out.WriteString(t.style.Color.IndexColumn.Sprint(colStr))
		} else if rowNum%2 == 0 && t.style.Color.RowAlternate != nil {
			out.WriteString(t.style.Color.RowAlternate.Sprint(colStr))
		} else if t.style.Color.Row != nil {
			out.WriteString(t.style.Color.Row.Sprint(colStr))
		} else {
			out.WriteString(colStr)
		}
	} else {
		out.WriteString(colStr)
	}
}

func (t *Table) renderColumnSeparator(out *strings.Builder, hint renderHint) {
	if t.style.Options.SeparateColumns {
		// type of row determines the character used (top/bottom/separator)
		if hint.isSeparatorRow {
			if hint.isBorderTop {
				out.WriteString(t.style.Box.TopSeparator)
			} else if hint.isBorderBottom {
				out.WriteString(t.style.Box.BottomSeparator)
			} else {
				out.WriteString(t.style.Box.MiddleSeparator)
			}
		} else {
			out.WriteString(t.style.Box.MiddleVertical)
		}
	}
}

func (t *Table) renderLine(out *strings.Builder, rowNum int, row rowStr, hint renderHint) {
	// if the output has content, it means that this call is working on line
	// number 2 or more; separate them with a newline
	if out.Len() > 0 {
		out.WriteRune('\n')
	}

	// use a brand new strings.Builder if a row length limit has been set
	var outLine *strings.Builder
	if t.allowedRowLength > 0 {
		outLine = &strings.Builder{}
	} else {
		outLine = out
	}
	// grow the strings.Builder to the maximum possible row length
	outLine.Grow(t.maxRowLength)

	t.renderMarginLeft(outLine, hint)
	for colIdx, maxColumnLength := range t.maxColumnLengths {
		t.renderColumn(outLine, rowNum, row, colIdx, maxColumnLength, hint)
	}
	t.renderMarginRight(outLine, hint)

	// merge the strings.Builder objects if a new one was created earlier
	if outLine != out {
		outLineStr := outLine.String()
		if util.RuneCountWithoutEscapeSeq(outLineStr) > t.allowedRowLength {
			trimLength := t.allowedRowLength - utf8.RuneCountInString(t.style.Box.UnfinishedRow)
			if trimLength > 0 {
				out.WriteString(util.TrimTextWithoutEscapeSeq(outLineStr, trimLength))
				out.WriteString(t.style.Box.UnfinishedRow)
			}
		} else {
			out.WriteString(outLineStr)
		}
	}

	// if a page size has been set, and said number of lines has already
	// been rendered, and the header is not being rendered right now, render
	// the header all over again with a spacing line
	if hint.isRegularRow() {
		t.numLinesRendered++
		if t.pageSize > 0 && t.numLinesRendered%t.pageSize == 0 && !hint.isLastLineOfLastRow() {
			t.renderRowsFooter(out)
			t.renderRowsBorderBottom(out)
			out.WriteString(t.style.Box.PageSeparator)
			t.renderRowsBorderTop(out)
			t.renderRowsHeader(out)
		}
	}
}

func (t *Table) renderMarginLeft(out *strings.Builder, hint renderHint) {
	if t.style.Options.DrawBorder {
		// type of row determines the character used (top/bottom/separator/etc.)
		if hint.isBorderTop {
			out.WriteString(t.style.Box.TopLeft)
		} else if hint.isBorderBottom {
			out.WriteString(t.style.Box.BottomLeft)
		} else if hint.isSeparatorRow {
			out.WriteString(t.style.Box.LeftSeparator)
		} else {
			out.WriteString(t.style.Box.Left)
		}
	}
}

func (t *Table) renderMarginRight(out *strings.Builder, hint renderHint) {
	if t.style.Options.DrawBorder {
		// type of row determines the character used (top/bottom/separator/etc.)
		if hint.isBorderTop {
			out.WriteString(t.style.Box.TopRight)
		} else if hint.isBorderBottom {
			out.WriteString(t.style.Box.BottomRight)
		} else if hint.isSeparatorRow {
			out.WriteString(t.style.Box.RightSeparator)
		} else {
			out.WriteString(t.style.Box.Right)
		}
	}
}

func (t *Table) renderRow(out *strings.Builder, rowNum int, row rowStr, hint renderHint) {
	if len(row) > 0 {
		// fit every column into the allowedColumnLength/maxColumnLength limit and
		// in the process find the max. number of lines in any column in this row
		colMaxLines := 0
		rowWrapped := make(rowStr, len(row))
		for colIdx, colStr := range row {
			rowWrapped[colIdx] = util.WrapText(colStr, t.maxColumnLengths[colIdx])
			colNumLines := strings.Count(rowWrapped[colIdx], "\n") + 1
			if colNumLines > colMaxLines {
				colMaxLines = colNumLines
			}
		}

		// if there is just 1 line in all columns, add the row as such; else split
		// each column into individual lines and render them one-by-one
		if colMaxLines == 1 {
			hint.isLastLineOfRow = true
			t.renderLine(out, rowNum, row, hint)
		} else {
			// convert one row into N # of rows based on colMaxLines
			rowLines := make([]rowStr, len(row))
			for colIdx, colStr := range rowWrapped {
				rowLines[colIdx] = t.getVAlign(colIdx, hint).ApplyStr(colStr, colMaxLines)
			}
			for colLineIdx := 0; colLineIdx < colMaxLines; colLineIdx++ {
				rowLine := make(rowStr, len(rowLines))
				for colIdx, colLines := range rowLines {
					rowLine[colIdx] = colLines[colLineIdx]
				}
				hint.isLastLineOfRow = bool(colLineIdx == colMaxLines-1)
				t.renderLine(out, rowNum, rowLine, hint)
			}
		}
	}
}

func (t *Table) renderRowSeparator(out *strings.Builder, hint renderHint) {
	if (hint.isBorderTop || hint.isBorderBottom) && !t.style.Options.DrawBorder {
		return
	} else if hint.isHeaderRow && !t.style.Options.SeparateHeader {
		return
	} else if hint.isFooterRow && !t.style.Options.SeparateFooter {
		return
	}
	hint.isSeparatorRow = true
	t.renderLine(out, -1, t.rowSeparator, hint)
}

func (t *Table) renderRows(out *strings.Builder, rows []rowStr, hint renderHint) {
	hintSeparator := hint
	hintSeparator.isSeparatorRow = true

	for idx, row := range rows {
		hint.isFirstRow = bool(idx == 0)
		hint.isLastRow = bool(idx == len(rows)-1)

		t.renderRow(out, idx+1, row, hint)
		if t.style.Options.SeparateRows && idx < len(rows)-1 {
			t.renderRowSeparator(out, hintSeparator)
		}
	}
}

func (t *Table) renderRowsBorderBottom(out *strings.Builder) {
	t.renderRowSeparator(out, renderHint{isBorderBottom: true})
}

func (t *Table) renderRowsBorderTop(out *strings.Builder) {
	t.renderRowSeparator(out, renderHint{isBorderTop: true})
}

func (t *Table) renderRowsFooter(out *strings.Builder) {
	if len(t.rowsFooter) > 0 {
		t.renderRowSeparator(out, renderHint{isFooterRow: true, isSeparatorRow: true})
		t.renderRows(out, t.rowsFooter, renderHint{isFooterRow: true})
	}
}

func (t *Table) renderRowsHeader(out *strings.Builder) {
	// header rows or auto-index row
	if len(t.rowsHeader) > 0 || t.autoIndex {
		if len(t.rowsHeader) > 0 {
			t.renderRows(out, t.rowsHeader, renderHint{isHeaderRow: true})
		} else if t.autoIndex {
			t.renderRow(out, 0, t.getAutoIndexColumnIDs(), renderHint{isHeaderRow: true})
		}
		t.renderRowSeparator(out, renderHint{isHeaderRow: true, isSeparatorRow: true})
	}
}
