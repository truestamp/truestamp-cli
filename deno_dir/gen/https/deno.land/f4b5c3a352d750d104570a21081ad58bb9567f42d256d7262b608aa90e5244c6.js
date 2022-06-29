import { Cell } from "./cell.ts";
import { Row } from "./row.ts";
import { consumeWords, longest, strLength } from "./utils.ts";
/** Table layout renderer. */ export class TableLayout {
    /**
   * Table layout constructor.
   * @param table   Table instance.
   * @param options Render options.
   */ constructor(table, options){
        this.table = table;
        this.options = options;
    }
    /** Generate table string. */ toString() {
        const opts = this.createLayout();
        return opts.rows.length ? this.renderRows(opts) : "";
    }
    /**
   * Generates table layout including row and col span, converts all none
   * Cell/Row values to Cells and Rows and returns the layout rendering
   * settings.
   */ createLayout() {
        Object.keys(this.options.chars).forEach((key)=>{
            if (typeof this.options.chars[key] !== "string") {
                this.options.chars[key] = "";
            }
        });
        const hasBodyBorder = this.table.getBorder() || this.table.hasBodyBorder();
        const hasHeaderBorder = this.table.hasHeaderBorder();
        const hasBorder = hasHeaderBorder || hasBodyBorder;
        const rows = this.#getRows();
        const columns = Math.max(...rows.map((row)=>row.length));
        for (const row of rows){
            const length = row.length;
            if (length < columns) {
                const diff = columns - length;
                for(let i = 0; i < diff; i++){
                    row.push(this.createCell(null, row));
                }
            }
        }
        const padding = [];
        const width = [];
        for(let colIndex = 0; colIndex < columns; colIndex++){
            const minColWidth = Array.isArray(this.options.minColWidth) ? this.options.minColWidth[colIndex] : this.options.minColWidth;
            const maxColWidth = Array.isArray(this.options.maxColWidth) ? this.options.maxColWidth[colIndex] : this.options.maxColWidth;
            const colWidth = longest(colIndex, rows, maxColWidth);
            width[colIndex] = Math.min(maxColWidth, Math.max(minColWidth, colWidth));
            padding[colIndex] = Array.isArray(this.options.padding) ? this.options.padding[colIndex] : this.options.padding;
        }
        return {
            padding,
            width,
            rows,
            columns,
            hasBorder,
            hasBodyBorder,
            hasHeaderBorder
        };
    }
     #getRows() {
        const header = this.table.getHeader();
        const rows = header ? [
            header,
            ...this.table
        ] : this.table.slice();
        const hasSpan = rows.some((row)=>row.some((cell)=>cell instanceof Cell && (cell.getColSpan() > 1 || cell.getRowSpan() > 1)));
        if (hasSpan) {
            return this.spanRows(rows);
        }
        return rows.map((row)=>{
            const newRow = this.createRow(row);
            return newRow.map((cell)=>this.createCell(cell, newRow));
        });
    }
    /**
   * Fills rows and cols by specified row/col span with a reference of the
   * original cell.
   *
   * @param _rows     All table rows.
   * @param rowIndex  Current row index.
   * @param colIndex  Current col index.
   * @param rowSpan   Current row span.
   * @param colSpan   Current col span.
   */ spanRows(_rows, rowIndex = 0, colIndex = 0, rowSpan = [], colSpan = 1) {
        const rows = _rows;
        if (rowIndex >= rows.length && rowSpan.every((span)=>span === 1)) {
            return rows;
        } else if (rows[rowIndex] && colIndex >= rows[rowIndex].length && colIndex >= rowSpan.length && colSpan === 1) {
            return this.spanRows(rows, ++rowIndex, 0, rowSpan, 1);
        }
        if (colSpan > 1) {
            colSpan--;
            rowSpan[colIndex] = rowSpan[colIndex - 1];
            rows[rowIndex].splice(colIndex - 1, 0, rows[rowIndex][colIndex - 1]);
            return this.spanRows(rows, rowIndex, ++colIndex, rowSpan, colSpan);
        }
        if (colIndex === 0) {
            rows[rowIndex] = this.createRow(rows[rowIndex] || []);
        }
        if (rowSpan[colIndex] > 1) {
            rowSpan[colIndex]--;
            rows[rowIndex].splice(colIndex, 0, rows[rowIndex - 1][colIndex]);
            return this.spanRows(rows, rowIndex, ++colIndex, rowSpan, colSpan);
        }
        rows[rowIndex][colIndex] = this.createCell(rows[rowIndex][colIndex] || null, rows[rowIndex]);
        colSpan = rows[rowIndex][colIndex].getColSpan();
        rowSpan[colIndex] = rows[rowIndex][colIndex].getRowSpan();
        return this.spanRows(rows, rowIndex, ++colIndex, rowSpan, colSpan);
    }
    /**
   * Create a new row from existing row or cell array.
   * @param row Original row.
   */ createRow(row) {
        return Row.from(row).border(this.table.getBorder(), false).align(this.table.getAlign(), false);
    }
    /**
   * Create a new cell from existing cell or cell value.
   * @param cell  Original cell.
   * @param row   Parent row.
   */ createCell(cell, row) {
        return Cell.from(cell ?? "").border(row.getBorder(), false).align(row.getAlign(), false);
    }
    /**
   * Render table layout.
   * @param opts Render options.
   */ renderRows(opts) {
        let result = "";
        const rowSpan = new Array(opts.columns).fill(1);
        for(let rowIndex = 0; rowIndex < opts.rows.length; rowIndex++){
            result += this.renderRow(rowSpan, rowIndex, opts);
        }
        return result.slice(0, -1);
    }
    /**
   * Render row.
   * @param rowSpan     Current row span.
   * @param rowIndex    Current row index.
   * @param opts        Render options.
   * @param isMultiline Is multiline row.
   */ renderRow(rowSpan, rowIndex, opts, isMultiline) {
        const row = opts.rows[rowIndex];
        const prevRow = opts.rows[rowIndex - 1];
        const nextRow = opts.rows[rowIndex + 1];
        let result = "";
        let colSpan = 1;
        // border top row
        if (!isMultiline && rowIndex === 0 && row.hasBorder()) {
            result += this.renderBorderRow(undefined, row, rowSpan, opts);
        }
        let isMultilineRow = false;
        result += " ".repeat(this.options.indent || 0);
        for(let colIndex = 0; colIndex < opts.columns; colIndex++){
            if (colSpan > 1) {
                colSpan--;
                rowSpan[colIndex] = rowSpan[colIndex - 1];
                continue;
            }
            result += this.renderCell(colIndex, row, opts);
            if (rowSpan[colIndex] > 1) {
                if (!isMultiline) {
                    rowSpan[colIndex]--;
                }
            } else if (!prevRow || prevRow[colIndex] !== row[colIndex]) {
                rowSpan[colIndex] = row[colIndex].getRowSpan();
            }
            colSpan = row[colIndex].getColSpan();
            if (rowSpan[colIndex] === 1 && row[colIndex].length) {
                isMultilineRow = true;
            }
        }
        if (opts.columns > 0) {
            if (row[opts.columns - 1].getBorder()) {
                result += this.options.chars.right;
            } else if (opts.hasBorder) {
                result += " ";
            }
        }
        result += "\n";
        if (isMultilineRow) {
            return result + this.renderRow(rowSpan, rowIndex, opts, isMultilineRow);
        }
        // border mid row
        if (rowIndex === 0 && opts.hasHeaderBorder || rowIndex < opts.rows.length - 1 && opts.hasBodyBorder) {
            result += this.renderBorderRow(row, nextRow, rowSpan, opts);
        }
        // border bottom row
        if (rowIndex === opts.rows.length - 1 && row.hasBorder()) {
            result += this.renderBorderRow(row, undefined, rowSpan, opts);
        }
        return result;
    }
    /**
   * Render cell.
   * @param colIndex  Current col index.
   * @param row       Current row.
   * @param opts      Render options.
   * @param noBorder  Disable border.
   */ renderCell(colIndex, row, opts, noBorder) {
        let result = "";
        const prevCell = row[colIndex - 1];
        const cell = row[colIndex];
        if (!noBorder) {
            if (colIndex === 0) {
                if (cell.getBorder()) {
                    result += this.options.chars.left;
                } else if (opts.hasBorder) {
                    result += " ";
                }
            } else {
                if (cell.getBorder() || prevCell?.getBorder()) {
                    result += this.options.chars.middle;
                } else if (opts.hasBorder) {
                    result += " ";
                }
            }
        }
        let maxLength = opts.width[colIndex];
        const colSpan = cell.getColSpan();
        if (colSpan > 1) {
            for(let o = 1; o < colSpan; o++){
                // add padding and with of next cell
                maxLength += opts.width[colIndex + o] + opts.padding[colIndex + o];
                if (opts.hasBorder) {
                    // add padding again and border with
                    maxLength += opts.padding[colIndex + o] + 1;
                }
            }
        }
        const { current , next  } = this.renderCellValue(cell, maxLength);
        row[colIndex].setValue(next);
        if (opts.hasBorder) {
            result += " ".repeat(opts.padding[colIndex]);
        }
        result += current;
        if (opts.hasBorder || colIndex < opts.columns - 1) {
            result += " ".repeat(opts.padding[colIndex]);
        }
        return result;
    }
    /**
   * Render specified length of cell. Returns the rendered value and a new cell
   * with the rest value.
   * @param cell      Cell to render.
   * @param maxLength Max length of content to render.
   */ renderCellValue(cell, maxLength) {
        const length = Math.min(maxLength, strLength(cell.toString()));
        let words = consumeWords(length, cell.toString());
        // break word if word is longer than max length
        const breakWord = strLength(words) > length;
        if (breakWord) {
            words = words.slice(0, length);
        }
        // get next content and remove leading space if breakWord is not true
        const next = cell.toString().slice(words.length + (breakWord ? 0 : 1));
        const fillLength = maxLength - strLength(words);
        // Align content
        const align = cell.getAlign();
        let current;
        if (fillLength === 0) {
            current = words;
        } else if (align === "left") {
            current = words + " ".repeat(fillLength);
        } else if (align === "center") {
            current = " ".repeat(Math.floor(fillLength / 2)) + words + " ".repeat(Math.ceil(fillLength / 2));
        } else if (align === "right") {
            current = " ".repeat(fillLength) + words;
        } else {
            throw new Error("Unknown direction: " + align);
        }
        return {
            current,
            next: cell.clone(next)
        };
    }
    /**
   * Render border row.
   * @param prevRow Previous row.
   * @param nextRow Next row.
   * @param rowSpan Current row span.
   * @param opts    Render options.
   */ renderBorderRow(prevRow, nextRow, rowSpan, opts) {
        let result = "";
        let colSpan = 1;
        for(let colIndex = 0; colIndex < opts.columns; colIndex++){
            if (rowSpan[colIndex] > 1) {
                if (!nextRow) {
                    throw new Error("invalid layout");
                }
                if (colSpan > 1) {
                    colSpan--;
                    continue;
                }
            }
            result += this.renderBorderCell(colIndex, prevRow, nextRow, rowSpan, opts);
            colSpan = nextRow?.[colIndex].getColSpan() ?? 1;
        }
        return result.length ? " ".repeat(this.options.indent) + result + "\n" : "";
    }
    /**
   * Render border cell.
   * @param colIndex  Current index.
   * @param prevRow   Previous row.
   * @param nextRow   Next row.
   * @param rowSpan   Current row span.
   * @param opts      Render options.
   */ renderBorderCell(colIndex, prevRow, nextRow, rowSpan, opts) {
        // a1 | b1
        // -------
        // a2 | b2
        const a1 = prevRow?.[colIndex - 1];
        const a2 = nextRow?.[colIndex - 1];
        const b1 = prevRow?.[colIndex];
        const b2 = nextRow?.[colIndex];
        const a1Border = !!a1?.getBorder();
        const a2Border = !!a2?.getBorder();
        const b1Border = !!b1?.getBorder();
        const b2Border = !!b2?.getBorder();
        const hasColSpan = (cell)=>(cell?.getColSpan() ?? 1) > 1;
        const hasRowSpan = (cell)=>(cell?.getRowSpan() ?? 1) > 1;
        let result = "";
        if (colIndex === 0) {
            if (rowSpan[colIndex] > 1) {
                if (b1Border) {
                    result += this.options.chars.left;
                } else {
                    result += " ";
                }
            } else if (b1Border && b2Border) {
                result += this.options.chars.leftMid;
            } else if (b1Border) {
                result += this.options.chars.bottomLeft;
            } else if (b2Border) {
                result += this.options.chars.topLeft;
            } else {
                result += " ";
            }
        } else if (colIndex < opts.columns) {
            if (a1Border && b2Border || b1Border && a2Border) {
                const a1ColSpan = hasColSpan(a1);
                const a2ColSpan = hasColSpan(a2);
                const b1ColSpan = hasColSpan(b1);
                const b2ColSpan = hasColSpan(b2);
                const a1RowSpan = hasRowSpan(a1);
                const a2RowSpan = hasRowSpan(a2);
                const b1RowSpan = hasRowSpan(b1);
                const b2RowSpan = hasRowSpan(b2);
                const hasAllBorder = a1Border && b2Border && b1Border && a2Border;
                const hasAllRowSpan = a1RowSpan && b1RowSpan && a2RowSpan && b2RowSpan;
                const hasAllColSpan = a1ColSpan && b1ColSpan && a2ColSpan && b2ColSpan;
                if (hasAllRowSpan && hasAllBorder) {
                    result += this.options.chars.middle;
                } else if (hasAllColSpan && hasAllBorder && a1 === b1 && a2 === b2) {
                    result += this.options.chars.mid;
                } else if (a1ColSpan && b1ColSpan && a1 === b1) {
                    result += this.options.chars.topMid;
                } else if (a2ColSpan && b2ColSpan && a2 === b2) {
                    result += this.options.chars.bottomMid;
                } else if (a1RowSpan && a2RowSpan && a1 === a2) {
                    result += this.options.chars.leftMid;
                } else if (b1RowSpan && b2RowSpan && b1 === b2) {
                    result += this.options.chars.rightMid;
                } else {
                    result += this.options.chars.midMid;
                }
            } else if (a1Border && b1Border) {
                if (hasColSpan(a1) && hasColSpan(b1) && a1 === b1) {
                    result += this.options.chars.bottom;
                } else {
                    result += this.options.chars.bottomMid;
                }
            } else if (b1Border && b2Border) {
                if (rowSpan[colIndex] > 1) {
                    result += this.options.chars.left;
                } else {
                    result += this.options.chars.leftMid;
                }
            } else if (b2Border && a2Border) {
                if (hasColSpan(a2) && hasColSpan(b2) && a2 === b2) {
                    result += this.options.chars.top;
                } else {
                    result += this.options.chars.topMid;
                }
            } else if (a1Border && a2Border) {
                if (hasRowSpan(a1) && a1 === a2) {
                    result += this.options.chars.right;
                } else {
                    result += this.options.chars.rightMid;
                }
            } else if (a1Border) {
                result += this.options.chars.bottomRight;
            } else if (b1Border) {
                result += this.options.chars.bottomLeft;
            } else if (a2Border) {
                result += this.options.chars.topRight;
            } else if (b2Border) {
                result += this.options.chars.topLeft;
            } else {
                result += " ";
            }
        }
        const length = opts.padding[colIndex] + opts.width[colIndex] + opts.padding[colIndex];
        if (rowSpan[colIndex] > 1 && nextRow) {
            result += this.renderCell(colIndex, nextRow, opts, true);
            if (nextRow[colIndex] === nextRow[nextRow.length - 1]) {
                if (b1Border) {
                    result += this.options.chars.right;
                } else {
                    result += " ";
                }
                return result;
            }
        } else if (b1Border && b2Border) {
            result += this.options.chars.mid.repeat(length);
        } else if (b1Border) {
            result += this.options.chars.bottom.repeat(length);
        } else if (b2Border) {
            result += this.options.chars.top.repeat(length);
        } else {
            result += " ".repeat(length);
        }
        if (colIndex === opts.columns - 1) {
            if (b1Border && b2Border) {
                result += this.options.chars.rightMid;
            } else if (b1Border) {
                result += this.options.chars.bottomRight;
            } else if (b2Border) {
                result += this.options.chars.topRight;
            } else {
                result += " ";
            }
        }
        return result;
    }
    table;
    options;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvdGFibGUvbGF5b3V0LnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENlbGwsIERpcmVjdGlvbiwgSUNlbGwgfSBmcm9tIFwiLi9jZWxsLnRzXCI7XG5pbXBvcnQgeyBJUm93LCBSb3cgfSBmcm9tIFwiLi9yb3cudHNcIjtcbmltcG9ydCB0eXBlIHsgSUJvcmRlck9wdGlvbnMsIElUYWJsZVNldHRpbmdzLCBUYWJsZSB9IGZyb20gXCIuL3RhYmxlLnRzXCI7XG5pbXBvcnQgeyBjb25zdW1lV29yZHMsIGxvbmdlc3QsIHN0ckxlbmd0aCB9IGZyb20gXCIuL3V0aWxzLnRzXCI7XG5cbi8qKiBMYXlvdXQgcmVuZGVyIHNldHRpbmdzLiAqL1xuaW50ZXJmYWNlIElSZW5kZXJTZXR0aW5ncyB7XG4gIHBhZGRpbmc6IG51bWJlcltdO1xuICB3aWR0aDogbnVtYmVyW107XG4gIGNvbHVtbnM6IG51bWJlcjtcbiAgaGFzQm9yZGVyOiBib29sZWFuO1xuICBoYXNIZWFkZXJCb3JkZXI6IGJvb2xlYW47XG4gIGhhc0JvZHlCb3JkZXI6IGJvb2xlYW47XG4gIHJvd3M6IFJvdzxDZWxsPltdO1xufVxuXG4vKiogVGFibGUgbGF5b3V0IHJlbmRlcmVyLiAqL1xuZXhwb3J0IGNsYXNzIFRhYmxlTGF5b3V0IHtcbiAgLyoqXG4gICAqIFRhYmxlIGxheW91dCBjb25zdHJ1Y3Rvci5cbiAgICogQHBhcmFtIHRhYmxlICAgVGFibGUgaW5zdGFuY2UuXG4gICAqIEBwYXJhbSBvcHRpb25zIFJlbmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGNvbnN0cnVjdG9yKFxuICAgIHByaXZhdGUgdGFibGU6IFRhYmxlLFxuICAgIHByaXZhdGUgb3B0aW9uczogSVRhYmxlU2V0dGluZ3MsXG4gICkge31cblxuICAvKiogR2VuZXJhdGUgdGFibGUgc3RyaW5nLiAqL1xuICBwdWJsaWMgdG9TdHJpbmcoKTogc3RyaW5nIHtcbiAgICBjb25zdCBvcHRzOiBJUmVuZGVyU2V0dGluZ3MgPSB0aGlzLmNyZWF0ZUxheW91dCgpO1xuICAgIHJldHVybiBvcHRzLnJvd3MubGVuZ3RoID8gdGhpcy5yZW5kZXJSb3dzKG9wdHMpIDogXCJcIjtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZW5lcmF0ZXMgdGFibGUgbGF5b3V0IGluY2x1ZGluZyByb3cgYW5kIGNvbCBzcGFuLCBjb252ZXJ0cyBhbGwgbm9uZVxuICAgKiBDZWxsL1JvdyB2YWx1ZXMgdG8gQ2VsbHMgYW5kIFJvd3MgYW5kIHJldHVybnMgdGhlIGxheW91dCByZW5kZXJpbmdcbiAgICogc2V0dGluZ3MuXG4gICAqL1xuICBwcm90ZWN0ZWQgY3JlYXRlTGF5b3V0KCk6IElSZW5kZXJTZXR0aW5ncyB7XG4gICAgT2JqZWN0LmtleXModGhpcy5vcHRpb25zLmNoYXJzKS5mb3JFYWNoKChrZXk6IHN0cmluZykgPT4ge1xuICAgICAgaWYgKHR5cGVvZiB0aGlzLm9wdGlvbnMuY2hhcnNba2V5IGFzIGtleW9mIElCb3JkZXJPcHRpb25zXSAhPT0gXCJzdHJpbmdcIikge1xuICAgICAgICB0aGlzLm9wdGlvbnMuY2hhcnNba2V5IGFzIGtleW9mIElCb3JkZXJPcHRpb25zXSA9IFwiXCI7XG4gICAgICB9XG4gICAgfSk7XG5cbiAgICBjb25zdCBoYXNCb2R5Qm9yZGVyOiBib29sZWFuID0gdGhpcy50YWJsZS5nZXRCb3JkZXIoKSB8fFxuICAgICAgdGhpcy50YWJsZS5oYXNCb2R5Qm9yZGVyKCk7XG4gICAgY29uc3QgaGFzSGVhZGVyQm9yZGVyOiBib29sZWFuID0gdGhpcy50YWJsZS5oYXNIZWFkZXJCb3JkZXIoKTtcbiAgICBjb25zdCBoYXNCb3JkZXI6IGJvb2xlYW4gPSBoYXNIZWFkZXJCb3JkZXIgfHwgaGFzQm9keUJvcmRlcjtcblxuICAgIGNvbnN0IHJvd3MgPSB0aGlzLiNnZXRSb3dzKCk7XG5cbiAgICBjb25zdCBjb2x1bW5zOiBudW1iZXIgPSBNYXRoLm1heCguLi5yb3dzLm1hcCgocm93KSA9PiByb3cubGVuZ3RoKSk7XG4gICAgZm9yIChjb25zdCByb3cgb2Ygcm93cykge1xuICAgICAgY29uc3QgbGVuZ3RoOiBudW1iZXIgPSByb3cubGVuZ3RoO1xuICAgICAgaWYgKGxlbmd0aCA8IGNvbHVtbnMpIHtcbiAgICAgICAgY29uc3QgZGlmZiA9IGNvbHVtbnMgLSBsZW5ndGg7XG4gICAgICAgIGZvciAobGV0IGkgPSAwOyBpIDwgZGlmZjsgaSsrKSB7XG4gICAgICAgICAgcm93LnB1c2godGhpcy5jcmVhdGVDZWxsKG51bGwsIHJvdykpO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgY29uc3QgcGFkZGluZzogbnVtYmVyW10gPSBbXTtcbiAgICBjb25zdCB3aWR0aDogbnVtYmVyW10gPSBbXTtcbiAgICBmb3IgKGxldCBjb2xJbmRleCA9IDA7IGNvbEluZGV4IDwgY29sdW1uczsgY29sSW5kZXgrKykge1xuICAgICAgY29uc3QgbWluQ29sV2lkdGg6IG51bWJlciA9IEFycmF5LmlzQXJyYXkodGhpcy5vcHRpb25zLm1pbkNvbFdpZHRoKVxuICAgICAgICA/IHRoaXMub3B0aW9ucy5taW5Db2xXaWR0aFtjb2xJbmRleF1cbiAgICAgICAgOiB0aGlzLm9wdGlvbnMubWluQ29sV2lkdGg7XG4gICAgICBjb25zdCBtYXhDb2xXaWR0aDogbnVtYmVyID0gQXJyYXkuaXNBcnJheSh0aGlzLm9wdGlvbnMubWF4Q29sV2lkdGgpXG4gICAgICAgID8gdGhpcy5vcHRpb25zLm1heENvbFdpZHRoW2NvbEluZGV4XVxuICAgICAgICA6IHRoaXMub3B0aW9ucy5tYXhDb2xXaWR0aDtcbiAgICAgIGNvbnN0IGNvbFdpZHRoOiBudW1iZXIgPSBsb25nZXN0KGNvbEluZGV4LCByb3dzLCBtYXhDb2xXaWR0aCk7XG4gICAgICB3aWR0aFtjb2xJbmRleF0gPSBNYXRoLm1pbihtYXhDb2xXaWR0aCwgTWF0aC5tYXgobWluQ29sV2lkdGgsIGNvbFdpZHRoKSk7XG4gICAgICBwYWRkaW5nW2NvbEluZGV4XSA9IEFycmF5LmlzQXJyYXkodGhpcy5vcHRpb25zLnBhZGRpbmcpXG4gICAgICAgID8gdGhpcy5vcHRpb25zLnBhZGRpbmdbY29sSW5kZXhdXG4gICAgICAgIDogdGhpcy5vcHRpb25zLnBhZGRpbmc7XG4gICAgfVxuXG4gICAgcmV0dXJuIHtcbiAgICAgIHBhZGRpbmcsXG4gICAgICB3aWR0aCxcbiAgICAgIHJvd3MsXG4gICAgICBjb2x1bW5zLFxuICAgICAgaGFzQm9yZGVyLFxuICAgICAgaGFzQm9keUJvcmRlcixcbiAgICAgIGhhc0hlYWRlckJvcmRlcixcbiAgICB9O1xuICB9XG5cbiAgI2dldFJvd3MoKSB7XG4gICAgY29uc3QgaGVhZGVyOiBSb3cgfCB1bmRlZmluZWQgPSB0aGlzLnRhYmxlLmdldEhlYWRlcigpO1xuICAgIGNvbnN0IHJvd3MgPSBoZWFkZXIgPyBbaGVhZGVyLCAuLi50aGlzLnRhYmxlXSA6IHRoaXMudGFibGUuc2xpY2UoKTtcbiAgICBjb25zdCBoYXNTcGFuID0gcm93cy5zb21lKChyb3cpID0+XG4gICAgICByb3cuc29tZSgoY2VsbCkgPT5cbiAgICAgICAgY2VsbCBpbnN0YW5jZW9mIENlbGwgJiYgKGNlbGwuZ2V0Q29sU3BhbigpID4gMSB8fCBjZWxsLmdldFJvd1NwYW4oKSA+IDEpXG4gICAgICApXG4gICAgKTtcblxuICAgIGlmIChoYXNTcGFuKSB7XG4gICAgICByZXR1cm4gdGhpcy5zcGFuUm93cyhyb3dzKTtcbiAgICB9XG5cbiAgICByZXR1cm4gcm93cy5tYXAoKHJvdykgPT4ge1xuICAgICAgY29uc3QgbmV3Um93ID0gdGhpcy5jcmVhdGVSb3cocm93KTtcbiAgICAgIHJldHVybiBuZXdSb3cubWFwKChjZWxsKSA9PiB0aGlzLmNyZWF0ZUNlbGwoY2VsbCwgbmV3Um93KSk7XG4gICAgfSkgYXMgQXJyYXk8Um93PENlbGw+PjtcbiAgfVxuXG4gIC8qKlxuICAgKiBGaWxscyByb3dzIGFuZCBjb2xzIGJ5IHNwZWNpZmllZCByb3cvY29sIHNwYW4gd2l0aCBhIHJlZmVyZW5jZSBvZiB0aGVcbiAgICogb3JpZ2luYWwgY2VsbC5cbiAgICpcbiAgICogQHBhcmFtIF9yb3dzICAgICBBbGwgdGFibGUgcm93cy5cbiAgICogQHBhcmFtIHJvd0luZGV4ICBDdXJyZW50IHJvdyBpbmRleC5cbiAgICogQHBhcmFtIGNvbEluZGV4ICBDdXJyZW50IGNvbCBpbmRleC5cbiAgICogQHBhcmFtIHJvd1NwYW4gICBDdXJyZW50IHJvdyBzcGFuLlxuICAgKiBAcGFyYW0gY29sU3BhbiAgIEN1cnJlbnQgY29sIHNwYW4uXG4gICAqL1xuICBwcm90ZWN0ZWQgc3BhblJvd3MoXG4gICAgX3Jvd3M6IElSb3dbXSxcbiAgICByb3dJbmRleCA9IDAsXG4gICAgY29sSW5kZXggPSAwLFxuICAgIHJvd1NwYW46IG51bWJlcltdID0gW10sXG4gICAgY29sU3BhbiA9IDEsXG4gICk6IFJvdzxDZWxsPltdIHtcbiAgICBjb25zdCByb3dzOiBSb3c8Q2VsbD5bXSA9IF9yb3dzIGFzIFJvdzxDZWxsPltdO1xuXG4gICAgaWYgKHJvd0luZGV4ID49IHJvd3MubGVuZ3RoICYmIHJvd1NwYW4uZXZlcnkoKHNwYW4pID0+IHNwYW4gPT09IDEpKSB7XG4gICAgICByZXR1cm4gcm93cztcbiAgICB9IGVsc2UgaWYgKFxuICAgICAgcm93c1tyb3dJbmRleF0gJiYgY29sSW5kZXggPj0gcm93c1tyb3dJbmRleF0ubGVuZ3RoICYmXG4gICAgICBjb2xJbmRleCA+PSByb3dTcGFuLmxlbmd0aCAmJiBjb2xTcGFuID09PSAxXG4gICAgKSB7XG4gICAgICByZXR1cm4gdGhpcy5zcGFuUm93cyhyb3dzLCArK3Jvd0luZGV4LCAwLCByb3dTcGFuLCAxKTtcbiAgICB9XG5cbiAgICBpZiAoY29sU3BhbiA+IDEpIHtcbiAgICAgIGNvbFNwYW4tLTtcbiAgICAgIHJvd1NwYW5bY29sSW5kZXhdID0gcm93U3Bhbltjb2xJbmRleCAtIDFdO1xuICAgICAgcm93c1tyb3dJbmRleF0uc3BsaWNlKGNvbEluZGV4IC0gMSwgMCwgcm93c1tyb3dJbmRleF1bY29sSW5kZXggLSAxXSk7XG4gICAgICByZXR1cm4gdGhpcy5zcGFuUm93cyhyb3dzLCByb3dJbmRleCwgKytjb2xJbmRleCwgcm93U3BhbiwgY29sU3Bhbik7XG4gICAgfVxuXG4gICAgaWYgKGNvbEluZGV4ID09PSAwKSB7XG4gICAgICByb3dzW3Jvd0luZGV4XSA9IHRoaXMuY3JlYXRlUm93KHJvd3Nbcm93SW5kZXhdIHx8IFtdKTtcbiAgICB9XG5cbiAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPiAxKSB7XG4gICAgICByb3dTcGFuW2NvbEluZGV4XS0tO1xuICAgICAgcm93c1tyb3dJbmRleF0uc3BsaWNlKGNvbEluZGV4LCAwLCByb3dzW3Jvd0luZGV4IC0gMV1bY29sSW5kZXhdKTtcbiAgICAgIHJldHVybiB0aGlzLnNwYW5Sb3dzKHJvd3MsIHJvd0luZGV4LCArK2NvbEluZGV4LCByb3dTcGFuLCBjb2xTcGFuKTtcbiAgICB9XG5cbiAgICByb3dzW3Jvd0luZGV4XVtjb2xJbmRleF0gPSB0aGlzLmNyZWF0ZUNlbGwoXG4gICAgICByb3dzW3Jvd0luZGV4XVtjb2xJbmRleF0gfHwgbnVsbCxcbiAgICAgIHJvd3Nbcm93SW5kZXhdLFxuICAgICk7XG5cbiAgICBjb2xTcGFuID0gcm93c1tyb3dJbmRleF1bY29sSW5kZXhdLmdldENvbFNwYW4oKTtcbiAgICByb3dTcGFuW2NvbEluZGV4XSA9IHJvd3Nbcm93SW5kZXhdW2NvbEluZGV4XS5nZXRSb3dTcGFuKCk7XG5cbiAgICByZXR1cm4gdGhpcy5zcGFuUm93cyhyb3dzLCByb3dJbmRleCwgKytjb2xJbmRleCwgcm93U3BhbiwgY29sU3Bhbik7XG4gIH1cblxuICAvKipcbiAgICogQ3JlYXRlIGEgbmV3IHJvdyBmcm9tIGV4aXN0aW5nIHJvdyBvciBjZWxsIGFycmF5LlxuICAgKiBAcGFyYW0gcm93IE9yaWdpbmFsIHJvdy5cbiAgICovXG4gIHByb3RlY3RlZCBjcmVhdGVSb3cocm93OiBJUm93KTogUm93PENlbGw+IHtcbiAgICByZXR1cm4gUm93LmZyb20ocm93KVxuICAgICAgLmJvcmRlcih0aGlzLnRhYmxlLmdldEJvcmRlcigpLCBmYWxzZSlcbiAgICAgIC5hbGlnbih0aGlzLnRhYmxlLmdldEFsaWduKCksIGZhbHNlKSBhcyBSb3c8Q2VsbD47XG4gIH1cblxuICAvKipcbiAgICogQ3JlYXRlIGEgbmV3IGNlbGwgZnJvbSBleGlzdGluZyBjZWxsIG9yIGNlbGwgdmFsdWUuXG4gICAqIEBwYXJhbSBjZWxsICBPcmlnaW5hbCBjZWxsLlxuICAgKiBAcGFyYW0gcm93ICAgUGFyZW50IHJvdy5cbiAgICovXG4gIHByb3RlY3RlZCBjcmVhdGVDZWxsKGNlbGw6IElDZWxsIHwgbnVsbCwgcm93OiBSb3cpOiBDZWxsIHtcbiAgICByZXR1cm4gQ2VsbC5mcm9tKGNlbGwgPz8gXCJcIilcbiAgICAgIC5ib3JkZXIocm93LmdldEJvcmRlcigpLCBmYWxzZSlcbiAgICAgIC5hbGlnbihyb3cuZ2V0QWxpZ24oKSwgZmFsc2UpO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciB0YWJsZSBsYXlvdXQuXG4gICAqIEBwYXJhbSBvcHRzIFJlbmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlclJvd3Mob3B0czogSVJlbmRlclNldHRpbmdzKTogc3RyaW5nIHtcbiAgICBsZXQgcmVzdWx0ID0gXCJcIjtcbiAgICBjb25zdCByb3dTcGFuOiBudW1iZXJbXSA9IG5ldyBBcnJheShvcHRzLmNvbHVtbnMpLmZpbGwoMSk7XG5cbiAgICBmb3IgKGxldCByb3dJbmRleCA9IDA7IHJvd0luZGV4IDwgb3B0cy5yb3dzLmxlbmd0aDsgcm93SW5kZXgrKykge1xuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyUm93KHJvd1NwYW4sIHJvd0luZGV4LCBvcHRzKTtcbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0LnNsaWNlKDAsIC0xKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgcm93LlxuICAgKiBAcGFyYW0gcm93U3BhbiAgICAgQ3VycmVudCByb3cgc3Bhbi5cbiAgICogQHBhcmFtIHJvd0luZGV4ICAgIEN1cnJlbnQgcm93IGluZGV4LlxuICAgKiBAcGFyYW0gb3B0cyAgICAgICAgUmVuZGVyIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBpc011bHRpbGluZSBJcyBtdWx0aWxpbmUgcm93LlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlclJvdyhcbiAgICByb3dTcGFuOiBudW1iZXJbXSxcbiAgICByb3dJbmRleDogbnVtYmVyLFxuICAgIG9wdHM6IElSZW5kZXJTZXR0aW5ncyxcbiAgICBpc011bHRpbGluZT86IGJvb2xlYW4sXG4gICk6IHN0cmluZyB7XG4gICAgY29uc3Qgcm93OiBSb3c8Q2VsbD4gPSBvcHRzLnJvd3Nbcm93SW5kZXhdO1xuICAgIGNvbnN0IHByZXZSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCA9IG9wdHMucm93c1tyb3dJbmRleCAtIDFdO1xuICAgIGNvbnN0IG5leHRSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCA9IG9wdHMucm93c1tyb3dJbmRleCArIDFdO1xuICAgIGxldCByZXN1bHQgPSBcIlwiO1xuXG4gICAgbGV0IGNvbFNwYW4gPSAxO1xuXG4gICAgLy8gYm9yZGVyIHRvcCByb3dcbiAgICBpZiAoIWlzTXVsdGlsaW5lICYmIHJvd0luZGV4ID09PSAwICYmIHJvdy5oYXNCb3JkZXIoKSkge1xuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyQm9yZGVyUm93KHVuZGVmaW5lZCwgcm93LCByb3dTcGFuLCBvcHRzKTtcbiAgICB9XG5cbiAgICBsZXQgaXNNdWx0aWxpbmVSb3cgPSBmYWxzZTtcblxuICAgIHJlc3VsdCArPSBcIiBcIi5yZXBlYXQodGhpcy5vcHRpb25zLmluZGVudCB8fCAwKTtcblxuICAgIGZvciAobGV0IGNvbEluZGV4ID0gMDsgY29sSW5kZXggPCBvcHRzLmNvbHVtbnM7IGNvbEluZGV4KyspIHtcbiAgICAgIGlmIChjb2xTcGFuID4gMSkge1xuICAgICAgICBjb2xTcGFuLS07XG4gICAgICAgIHJvd1NwYW5bY29sSW5kZXhdID0gcm93U3Bhbltjb2xJbmRleCAtIDFdO1xuICAgICAgICBjb250aW51ZTtcbiAgICAgIH1cblxuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyQ2VsbChjb2xJbmRleCwgcm93LCBvcHRzKTtcblxuICAgICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSkge1xuICAgICAgICBpZiAoIWlzTXVsdGlsaW5lKSB7XG4gICAgICAgICAgcm93U3Bhbltjb2xJbmRleF0tLTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmICghcHJldlJvdyB8fCBwcmV2Um93W2NvbEluZGV4XSAhPT0gcm93W2NvbEluZGV4XSkge1xuICAgICAgICByb3dTcGFuW2NvbEluZGV4XSA9IHJvd1tjb2xJbmRleF0uZ2V0Um93U3BhbigpO1xuICAgICAgfVxuXG4gICAgICBjb2xTcGFuID0gcm93W2NvbEluZGV4XS5nZXRDb2xTcGFuKCk7XG5cbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA9PT0gMSAmJiByb3dbY29sSW5kZXhdLmxlbmd0aCkge1xuICAgICAgICBpc011bHRpbGluZVJvdyA9IHRydWU7XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKG9wdHMuY29sdW1ucyA+IDApIHtcbiAgICAgIGlmIChyb3dbb3B0cy5jb2x1bW5zIC0gMV0uZ2V0Qm9yZGVyKCkpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodDtcbiAgICAgIH0gZWxzZSBpZiAob3B0cy5oYXNCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgfVxuICAgIH1cblxuICAgIHJlc3VsdCArPSBcIlxcblwiO1xuXG4gICAgaWYgKGlzTXVsdGlsaW5lUm93KSB7IC8vIHNraXAgYm9yZGVyXG4gICAgICByZXR1cm4gcmVzdWx0ICsgdGhpcy5yZW5kZXJSb3cocm93U3Bhbiwgcm93SW5kZXgsIG9wdHMsIGlzTXVsdGlsaW5lUm93KTtcbiAgICB9XG5cbiAgICAvLyBib3JkZXIgbWlkIHJvd1xuICAgIGlmIChcbiAgICAgIChyb3dJbmRleCA9PT0gMCAmJiBvcHRzLmhhc0hlYWRlckJvcmRlcikgfHxcbiAgICAgIChyb3dJbmRleCA8IG9wdHMucm93cy5sZW5ndGggLSAxICYmIG9wdHMuaGFzQm9keUJvcmRlcilcbiAgICApIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckJvcmRlclJvdyhyb3csIG5leHRSb3csIHJvd1NwYW4sIG9wdHMpO1xuICAgIH1cblxuICAgIC8vIGJvcmRlciBib3R0b20gcm93XG4gICAgaWYgKHJvd0luZGV4ID09PSBvcHRzLnJvd3MubGVuZ3RoIC0gMSAmJiByb3cuaGFzQm9yZGVyKCkpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckJvcmRlclJvdyhyb3csIHVuZGVmaW5lZCwgcm93U3Bhbiwgb3B0cyk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdDtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgY2VsbC5cbiAgICogQHBhcmFtIGNvbEluZGV4ICBDdXJyZW50IGNvbCBpbmRleC5cbiAgICogQHBhcmFtIHJvdyAgICAgICBDdXJyZW50IHJvdy5cbiAgICogQHBhcmFtIG9wdHMgICAgICBSZW5kZXIgb3B0aW9ucy5cbiAgICogQHBhcmFtIG5vQm9yZGVyICBEaXNhYmxlIGJvcmRlci5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJDZWxsKFxuICAgIGNvbEluZGV4OiBudW1iZXIsXG4gICAgcm93OiBSb3c8Q2VsbD4sXG4gICAgb3B0czogSVJlbmRlclNldHRpbmdzLFxuICAgIG5vQm9yZGVyPzogYm9vbGVhbixcbiAgKTogc3RyaW5nIHtcbiAgICBsZXQgcmVzdWx0ID0gXCJcIjtcbiAgICBjb25zdCBwcmV2Q2VsbDogQ2VsbCB8IHVuZGVmaW5lZCA9IHJvd1tjb2xJbmRleCAtIDFdO1xuXG4gICAgY29uc3QgY2VsbDogQ2VsbCA9IHJvd1tjb2xJbmRleF07XG5cbiAgICBpZiAoIW5vQm9yZGVyKSB7XG4gICAgICBpZiAoY29sSW5kZXggPT09IDApIHtcbiAgICAgICAgaWYgKGNlbGwuZ2V0Qm9yZGVyKCkpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnQ7XG4gICAgICAgIH0gZWxzZSBpZiAob3B0cy5oYXNCb3JkZXIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIGlmIChjZWxsLmdldEJvcmRlcigpIHx8IHByZXZDZWxsPy5nZXRCb3JkZXIoKSkge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkZGxlO1xuICAgICAgICB9IGVsc2UgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgbGV0IG1heExlbmd0aDogbnVtYmVyID0gb3B0cy53aWR0aFtjb2xJbmRleF07XG5cbiAgICBjb25zdCBjb2xTcGFuOiBudW1iZXIgPSBjZWxsLmdldENvbFNwYW4oKTtcbiAgICBpZiAoY29sU3BhbiA+IDEpIHtcbiAgICAgIGZvciAobGV0IG8gPSAxOyBvIDwgY29sU3BhbjsgbysrKSB7XG4gICAgICAgIC8vIGFkZCBwYWRkaW5nIGFuZCB3aXRoIG9mIG5leHQgY2VsbFxuICAgICAgICBtYXhMZW5ndGggKz0gb3B0cy53aWR0aFtjb2xJbmRleCArIG9dICsgb3B0cy5wYWRkaW5nW2NvbEluZGV4ICsgb107XG4gICAgICAgIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgICAgIC8vIGFkZCBwYWRkaW5nIGFnYWluIGFuZCBib3JkZXIgd2l0aFxuICAgICAgICAgIG1heExlbmd0aCArPSBvcHRzLnBhZGRpbmdbY29sSW5kZXggKyBvXSArIDE7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG5cbiAgICBjb25zdCB7IGN1cnJlbnQsIG5leHQgfSA9IHRoaXMucmVuZGVyQ2VsbFZhbHVlKGNlbGwsIG1heExlbmd0aCk7XG5cbiAgICByb3dbY29sSW5kZXhdLnNldFZhbHVlKG5leHQpO1xuXG4gICAgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICByZXN1bHQgKz0gXCIgXCIucmVwZWF0KG9wdHMucGFkZGluZ1tjb2xJbmRleF0pO1xuICAgIH1cblxuICAgIHJlc3VsdCArPSBjdXJyZW50O1xuXG4gICAgaWYgKG9wdHMuaGFzQm9yZGVyIHx8IGNvbEluZGV4IDwgb3B0cy5jb2x1bW5zIC0gMSkge1xuICAgICAgcmVzdWx0ICs9IFwiIFwiLnJlcGVhdChvcHRzLnBhZGRpbmdbY29sSW5kZXhdKTtcbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0O1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciBzcGVjaWZpZWQgbGVuZ3RoIG9mIGNlbGwuIFJldHVybnMgdGhlIHJlbmRlcmVkIHZhbHVlIGFuZCBhIG5ldyBjZWxsXG4gICAqIHdpdGggdGhlIHJlc3QgdmFsdWUuXG4gICAqIEBwYXJhbSBjZWxsICAgICAgQ2VsbCB0byByZW5kZXIuXG4gICAqIEBwYXJhbSBtYXhMZW5ndGggTWF4IGxlbmd0aCBvZiBjb250ZW50IHRvIHJlbmRlci5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJDZWxsVmFsdWUoXG4gICAgY2VsbDogQ2VsbCxcbiAgICBtYXhMZW5ndGg6IG51bWJlcixcbiAgKTogeyBjdXJyZW50OiBzdHJpbmc7IG5leHQ6IENlbGwgfSB7XG4gICAgY29uc3QgbGVuZ3RoOiBudW1iZXIgPSBNYXRoLm1pbihcbiAgICAgIG1heExlbmd0aCxcbiAgICAgIHN0ckxlbmd0aChjZWxsLnRvU3RyaW5nKCkpLFxuICAgICk7XG4gICAgbGV0IHdvcmRzOiBzdHJpbmcgPSBjb25zdW1lV29yZHMobGVuZ3RoLCBjZWxsLnRvU3RyaW5nKCkpO1xuXG4gICAgLy8gYnJlYWsgd29yZCBpZiB3b3JkIGlzIGxvbmdlciB0aGFuIG1heCBsZW5ndGhcbiAgICBjb25zdCBicmVha1dvcmQgPSBzdHJMZW5ndGgod29yZHMpID4gbGVuZ3RoO1xuICAgIGlmIChicmVha1dvcmQpIHtcbiAgICAgIHdvcmRzID0gd29yZHMuc2xpY2UoMCwgbGVuZ3RoKTtcbiAgICB9XG5cbiAgICAvLyBnZXQgbmV4dCBjb250ZW50IGFuZCByZW1vdmUgbGVhZGluZyBzcGFjZSBpZiBicmVha1dvcmQgaXMgbm90IHRydWVcbiAgICBjb25zdCBuZXh0ID0gY2VsbC50b1N0cmluZygpLnNsaWNlKHdvcmRzLmxlbmd0aCArIChicmVha1dvcmQgPyAwIDogMSkpO1xuICAgIGNvbnN0IGZpbGxMZW5ndGggPSBtYXhMZW5ndGggLSBzdHJMZW5ndGgod29yZHMpO1xuXG4gICAgLy8gQWxpZ24gY29udGVudFxuICAgIGNvbnN0IGFsaWduOiBEaXJlY3Rpb24gPSBjZWxsLmdldEFsaWduKCk7XG4gICAgbGV0IGN1cnJlbnQ6IHN0cmluZztcbiAgICBpZiAoZmlsbExlbmd0aCA9PT0gMCkge1xuICAgICAgY3VycmVudCA9IHdvcmRzO1xuICAgIH0gZWxzZSBpZiAoYWxpZ24gPT09IFwibGVmdFwiKSB7XG4gICAgICBjdXJyZW50ID0gd29yZHMgKyBcIiBcIi5yZXBlYXQoZmlsbExlbmd0aCk7XG4gICAgfSBlbHNlIGlmIChhbGlnbiA9PT0gXCJjZW50ZXJcIikge1xuICAgICAgY3VycmVudCA9IFwiIFwiLnJlcGVhdChNYXRoLmZsb29yKGZpbGxMZW5ndGggLyAyKSkgKyB3b3JkcyArXG4gICAgICAgIFwiIFwiLnJlcGVhdChNYXRoLmNlaWwoZmlsbExlbmd0aCAvIDIpKTtcbiAgICB9IGVsc2UgaWYgKGFsaWduID09PSBcInJpZ2h0XCIpIHtcbiAgICAgIGN1cnJlbnQgPSBcIiBcIi5yZXBlYXQoZmlsbExlbmd0aCkgKyB3b3JkcztcbiAgICB9IGVsc2Uge1xuICAgICAgdGhyb3cgbmV3IEVycm9yKFwiVW5rbm93biBkaXJlY3Rpb246IFwiICsgYWxpZ24pO1xuICAgIH1cblxuICAgIHJldHVybiB7XG4gICAgICBjdXJyZW50LFxuICAgICAgbmV4dDogY2VsbC5jbG9uZShuZXh0KSxcbiAgICB9O1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciBib3JkZXIgcm93LlxuICAgKiBAcGFyYW0gcHJldlJvdyBQcmV2aW91cyByb3cuXG4gICAqIEBwYXJhbSBuZXh0Um93IE5leHQgcm93LlxuICAgKiBAcGFyYW0gcm93U3BhbiBDdXJyZW50IHJvdyBzcGFuLlxuICAgKiBAcGFyYW0gb3B0cyAgICBSZW5kZXIgb3B0aW9ucy5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJCb3JkZXJSb3coXG4gICAgcHJldlJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkLFxuICAgIG5leHRSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCxcbiAgICByb3dTcGFuOiBudW1iZXJbXSxcbiAgICBvcHRzOiBJUmVuZGVyU2V0dGluZ3MsXG4gICk6IHN0cmluZyB7XG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG5cbiAgICBsZXQgY29sU3BhbiA9IDE7XG4gICAgZm9yIChsZXQgY29sSW5kZXggPSAwOyBjb2xJbmRleCA8IG9wdHMuY29sdW1uczsgY29sSW5kZXgrKykge1xuICAgICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSkge1xuICAgICAgICBpZiAoIW5leHRSb3cpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJpbnZhbGlkIGxheW91dFwiKTtcbiAgICAgICAgfVxuICAgICAgICBpZiAoY29sU3BhbiA+IDEpIHtcbiAgICAgICAgICBjb2xTcGFuLS07XG4gICAgICAgICAgY29udGludWU7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckJvcmRlckNlbGwoXG4gICAgICAgIGNvbEluZGV4LFxuICAgICAgICBwcmV2Um93LFxuICAgICAgICBuZXh0Um93LFxuICAgICAgICByb3dTcGFuLFxuICAgICAgICBvcHRzLFxuICAgICAgKTtcbiAgICAgIGNvbFNwYW4gPSBuZXh0Um93Py5bY29sSW5kZXhdLmdldENvbFNwYW4oKSA/PyAxO1xuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQubGVuZ3RoID8gXCIgXCIucmVwZWF0KHRoaXMub3B0aW9ucy5pbmRlbnQpICsgcmVzdWx0ICsgXCJcXG5cIiA6IFwiXCI7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIGJvcmRlciBjZWxsLlxuICAgKiBAcGFyYW0gY29sSW5kZXggIEN1cnJlbnQgaW5kZXguXG4gICAqIEBwYXJhbSBwcmV2Um93ICAgUHJldmlvdXMgcm93LlxuICAgKiBAcGFyYW0gbmV4dFJvdyAgIE5leHQgcm93LlxuICAgKiBAcGFyYW0gcm93U3BhbiAgIEN1cnJlbnQgcm93IHNwYW4uXG4gICAqIEBwYXJhbSBvcHRzICAgICAgUmVuZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQm9yZGVyQ2VsbChcbiAgICBjb2xJbmRleDogbnVtYmVyLFxuICAgIHByZXZSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCxcbiAgICBuZXh0Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQsXG4gICAgcm93U3BhbjogbnVtYmVyW10sXG4gICAgb3B0czogSVJlbmRlclNldHRpbmdzLFxuICApOiBzdHJpbmcge1xuICAgIC8vIGExIHwgYjFcbiAgICAvLyAtLS0tLS0tXG4gICAgLy8gYTIgfCBiMlxuXG4gICAgY29uc3QgYTE6IENlbGwgfCB1bmRlZmluZWQgPSBwcmV2Um93Py5bY29sSW5kZXggLSAxXTtcbiAgICBjb25zdCBhMjogQ2VsbCB8IHVuZGVmaW5lZCA9IG5leHRSb3c/Lltjb2xJbmRleCAtIDFdO1xuICAgIGNvbnN0IGIxOiBDZWxsIHwgdW5kZWZpbmVkID0gcHJldlJvdz8uW2NvbEluZGV4XTtcbiAgICBjb25zdCBiMjogQ2VsbCB8IHVuZGVmaW5lZCA9IG5leHRSb3c/Lltjb2xJbmRleF07XG5cbiAgICBjb25zdCBhMUJvcmRlciA9ICEhYTE/LmdldEJvcmRlcigpO1xuICAgIGNvbnN0IGEyQm9yZGVyID0gISFhMj8uZ2V0Qm9yZGVyKCk7XG4gICAgY29uc3QgYjFCb3JkZXIgPSAhIWIxPy5nZXRCb3JkZXIoKTtcbiAgICBjb25zdCBiMkJvcmRlciA9ICEhYjI/LmdldEJvcmRlcigpO1xuXG4gICAgY29uc3QgaGFzQ29sU3BhbiA9IChjZWxsOiBDZWxsIHwgdW5kZWZpbmVkKTogYm9vbGVhbiA9PlxuICAgICAgKGNlbGw/LmdldENvbFNwYW4oKSA/PyAxKSA+IDE7XG4gICAgY29uc3QgaGFzUm93U3BhbiA9IChjZWxsOiBDZWxsIHwgdW5kZWZpbmVkKTogYm9vbGVhbiA9PlxuICAgICAgKGNlbGw/LmdldFJvd1NwYW4oKSA/PyAxKSA+IDE7XG5cbiAgICBsZXQgcmVzdWx0ID0gXCJcIjtcblxuICAgIGlmIChjb2xJbmRleCA9PT0gMCkge1xuICAgICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSkge1xuICAgICAgICBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyICYmIGIyQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdE1pZDtcbiAgICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21MZWZ0O1xuICAgICAgfSBlbHNlIGlmIChiMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcExlZnQ7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICB9XG4gICAgfSBlbHNlIGlmIChjb2xJbmRleCA8IG9wdHMuY29sdW1ucykge1xuICAgICAgaWYgKChhMUJvcmRlciAmJiBiMkJvcmRlcikgfHwgKGIxQm9yZGVyICYmIGEyQm9yZGVyKSkge1xuICAgICAgICBjb25zdCBhMUNvbFNwYW46IGJvb2xlYW4gPSBoYXNDb2xTcGFuKGExKTtcbiAgICAgICAgY29uc3QgYTJDb2xTcGFuOiBib29sZWFuID0gaGFzQ29sU3BhbihhMik7XG4gICAgICAgIGNvbnN0IGIxQ29sU3BhbjogYm9vbGVhbiA9IGhhc0NvbFNwYW4oYjEpO1xuICAgICAgICBjb25zdCBiMkNvbFNwYW46IGJvb2xlYW4gPSBoYXNDb2xTcGFuKGIyKTtcblxuICAgICAgICBjb25zdCBhMVJvd1NwYW46IGJvb2xlYW4gPSBoYXNSb3dTcGFuKGExKTtcbiAgICAgICAgY29uc3QgYTJSb3dTcGFuOiBib29sZWFuID0gaGFzUm93U3BhbihhMik7XG4gICAgICAgIGNvbnN0IGIxUm93U3BhbjogYm9vbGVhbiA9IGhhc1Jvd1NwYW4oYjEpO1xuICAgICAgICBjb25zdCBiMlJvd1NwYW46IGJvb2xlYW4gPSBoYXNSb3dTcGFuKGIyKTtcblxuICAgICAgICBjb25zdCBoYXNBbGxCb3JkZXIgPSBhMUJvcmRlciAmJiBiMkJvcmRlciAmJiBiMUJvcmRlciAmJiBhMkJvcmRlcjtcbiAgICAgICAgY29uc3QgaGFzQWxsUm93U3BhbiA9IGExUm93U3BhbiAmJiBiMVJvd1NwYW4gJiYgYTJSb3dTcGFuICYmIGIyUm93U3BhbjtcbiAgICAgICAgY29uc3QgaGFzQWxsQ29sU3BhbiA9IGExQ29sU3BhbiAmJiBiMUNvbFNwYW4gJiYgYTJDb2xTcGFuICYmIGIyQ29sU3BhbjtcblxuICAgICAgICBpZiAoaGFzQWxsUm93U3BhbiAmJiBoYXNBbGxCb3JkZXIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLm1pZGRsZTtcbiAgICAgICAgfSBlbHNlIGlmIChoYXNBbGxDb2xTcGFuICYmIGhhc0FsbEJvcmRlciAmJiBhMSA9PT0gYjEgJiYgYTIgPT09IGIyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWQ7XG4gICAgICAgIH0gZWxzZSBpZiAoYTFDb2xTcGFuICYmIGIxQ29sU3BhbiAmJiBhMSA9PT0gYjEpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcE1pZDtcbiAgICAgICAgfSBlbHNlIGlmIChhMkNvbFNwYW4gJiYgYjJDb2xTcGFuICYmIGEyID09PSBiMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTWlkO1xuICAgICAgICB9IGVsc2UgaWYgKGExUm93U3BhbiAmJiBhMlJvd1NwYW4gJiYgYTEgPT09IGEyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0TWlkO1xuICAgICAgICB9IGVsc2UgaWYgKGIxUm93U3BhbiAmJiBiMlJvd1NwYW4gJiYgYjEgPT09IGIyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodE1pZDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLm1pZE1pZDtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChhMUJvcmRlciAmJiBiMUJvcmRlcikge1xuICAgICAgICBpZiAoaGFzQ29sU3BhbihhMSkgJiYgaGFzQ29sU3BhbihiMSkgJiYgYTEgPT09IGIxKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b207XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21NaWQ7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIgJiYgYjJCb3JkZXIpIHtcbiAgICAgICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSkge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnRNaWQ7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIgJiYgYTJCb3JkZXIpIHtcbiAgICAgICAgaWYgKGhhc0NvbFNwYW4oYTIpICYmIGhhc0NvbFNwYW4oYjIpICYmIGEyID09PSBiMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wTWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGExQm9yZGVyICYmIGEyQm9yZGVyKSB7XG4gICAgICAgIGlmIChoYXNSb3dTcGFuKGExKSAmJiBhMSA9PT0gYTIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0O1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHRNaWQ7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYTFCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21SaWdodDtcbiAgICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21MZWZ0O1xuICAgICAgfSBlbHNlIGlmIChhMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcFJpZ2h0O1xuICAgICAgfSBlbHNlIGlmIChiMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcExlZnQ7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICB9XG4gICAgfVxuXG4gICAgY29uc3QgbGVuZ3RoID0gb3B0cy5wYWRkaW5nW2NvbEluZGV4XSArIG9wdHMud2lkdGhbY29sSW5kZXhdICtcbiAgICAgIG9wdHMucGFkZGluZ1tjb2xJbmRleF07XG5cbiAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPiAxICYmIG5leHRSb3cpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckNlbGwoXG4gICAgICAgIGNvbEluZGV4LFxuICAgICAgICBuZXh0Um93LFxuICAgICAgICBvcHRzLFxuICAgICAgICB0cnVlLFxuICAgICAgKTtcbiAgICAgIGlmIChuZXh0Um93W2NvbEluZGV4XSA9PT0gbmV4dFJvd1tuZXh0Um93Lmxlbmd0aCAtIDFdKSB7XG4gICAgICAgIGlmIChiMUJvcmRlcikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiByZXN1bHQ7XG4gICAgICB9XG4gICAgfSBlbHNlIGlmIChiMUJvcmRlciAmJiBiMkJvcmRlcikge1xuICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWQucmVwZWF0KGxlbmd0aCk7XG4gICAgfSBlbHNlIGlmIChiMUJvcmRlcikge1xuICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b20ucmVwZWF0KGxlbmd0aCk7XG4gICAgfSBlbHNlIGlmIChiMkJvcmRlcikge1xuICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3AucmVwZWF0KGxlbmd0aCk7XG4gICAgfSBlbHNlIHtcbiAgICAgIHJlc3VsdCArPSBcIiBcIi5yZXBlYXQobGVuZ3RoKTtcbiAgICB9XG5cbiAgICBpZiAoY29sSW5kZXggPT09IG9wdHMuY29sdW1ucyAtIDEpIHtcbiAgICAgIGlmIChiMUJvcmRlciAmJiBiMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0TWlkO1xuICAgICAgfSBlbHNlIGlmIChiMUJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbVJpZ2h0O1xuICAgICAgfSBlbHNlIGlmIChiMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcFJpZ2h0O1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgfVxuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQ7XG4gIH1cbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUFTLElBQUksUUFBMEIsV0FBVyxDQUFDO0FBQ25ELFNBQWUsR0FBRyxRQUFRLFVBQVUsQ0FBQztBQUVyQyxTQUFTLFlBQVksRUFBRSxPQUFPLEVBQUUsU0FBUyxRQUFRLFlBQVksQ0FBQztBQWE5RCw2QkFBNkIsQ0FDN0IsT0FBTyxNQUFNLFdBQVc7SUFDdEI7Ozs7S0FJRyxDQUNILFlBQ1UsS0FBWSxFQUNaLE9BQXVCLENBQy9CO2FBRlEsS0FBWSxHQUFaLEtBQVk7YUFDWixPQUF1QixHQUF2QixPQUF1QjtLQUM3QjtJQUVKLDZCQUE2QixDQUM3QixBQUFPLFFBQVEsR0FBVztRQUN4QixNQUFNLElBQUksR0FBb0IsSUFBSSxDQUFDLFlBQVksRUFBRSxBQUFDO1FBQ2xELE9BQU8sSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsSUFBSSxDQUFDLFVBQVUsQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLENBQUM7S0FDdEQ7SUFFRDs7OztLQUlHLENBQ0gsQUFBVSxZQUFZLEdBQW9CO1FBQ3hDLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsQ0FBQyxPQUFPLENBQUMsQ0FBQyxHQUFXLEdBQUs7WUFDdkQsSUFBSSxPQUFPLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBeUIsS0FBSyxRQUFRLEVB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SUFBSSxZQUFZLElBQUksRUFBRSxLQUFLLEVBQUUsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUNsRSxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDO2lCQUNsQyxNQUFNLElBQUksU0FBUyxJQUFJLFNBQVMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUM5QyxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDO2lCQUNyQyxNQUFNLElBQUksU0FBUyxJQUFJLFNBQVMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUM5QyxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDO2lCQUN4QyxNQUFNLElBQUksU0FBUyxJQUFJLFNBQVMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUM5QyxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsT0FBTyxDQUFDO2lCQUN0QyxNQUFNLElBQUksU0FBUyxJQUFJLFNBQVMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUM5QyxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsUUFBUSxDQUFDO2lCQUN2QyxNQUFNO29CQUNMLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxNQUFNLENBQUM7aUJBQ3JDO2FBQ0YsTUFBTSxJQUFJLFFBQVEsSUFBSSxRQUFRLEVBQUU7Z0JBQy9CLElBQUksVUFBVSxDQUFDLEVBQUUsQ0FBQyxJQUFJLFVBQVUsQ0FBQyxFQUFFLENBQUMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUNqRCxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDO2lCQUNyQyxNQUFNO29CQUNMLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxTQUFTLENBQUM7aUJBQ3hDO2FBQ0YsTUFB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