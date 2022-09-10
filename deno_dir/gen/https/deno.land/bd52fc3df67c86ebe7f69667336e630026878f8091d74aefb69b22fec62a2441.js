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
            for(let i = 0; i < row.length; i++){
                newRow[i] = this.createCell(row[i], newRow);
            }
            return newRow;
        });
    }
    /**
   * Fills rows and cols by specified row/col span with a reference of the
   * original cell.
   */ spanRows(rows) {
        const rowSpan = [];
        let colSpan = 1;
        let rowIndex = -1;
        while(true){
            rowIndex++;
            if (rowIndex === rows.length && rowSpan.every((span)=>span === 1)) {
                break;
            }
            const row = rows[rowIndex] = this.createRow(rows[rowIndex] || []);
            let colIndex = -1;
            while(true){
                colIndex++;
                if (colIndex === row.length && colIndex === rowSpan.length && colSpan === 1) {
                    break;
                }
                if (colSpan > 1) {
                    colSpan--;
                    rowSpan[colIndex] = rowSpan[colIndex - 1];
                    row.splice(colIndex, this.getDeleteCount(rows, rowIndex, colIndex), row[colIndex - 1]);
                    continue;
                }
                if (rowSpan[colIndex] > 1) {
                    rowSpan[colIndex]--;
                    rows[rowIndex].splice(colIndex, this.getDeleteCount(rows, rowIndex, colIndex), rows[rowIndex - 1][colIndex]);
                    continue;
                }
                const cell = row[colIndex] = this.createCell(row[colIndex] || null, row);
                colSpan = cell.getColSpan();
                rowSpan[colIndex] = cell.getRowSpan();
            }
        }
        return rows;
    }
    getDeleteCount(rows, rowIndex, colIndex) {
        return colIndex <= rows[rowIndex].length - 1 && typeof rows[rowIndex][colIndex] === "undefined" ? 1 : 0;
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvdGFibGUvbGF5b3V0LnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENlbGwsIERpcmVjdGlvbiwgSUNlbGwgfSBmcm9tIFwiLi9jZWxsLnRzXCI7XG5pbXBvcnQgeyBJUm93LCBSb3cgfSBmcm9tIFwiLi9yb3cudHNcIjtcbmltcG9ydCB0eXBlIHsgSUJvcmRlck9wdGlvbnMsIElUYWJsZVNldHRpbmdzLCBUYWJsZSB9IGZyb20gXCIuL3RhYmxlLnRzXCI7XG5pbXBvcnQgeyBjb25zdW1lV29yZHMsIGxvbmdlc3QsIHN0ckxlbmd0aCB9IGZyb20gXCIuL3V0aWxzLnRzXCI7XG5cbi8qKiBMYXlvdXQgcmVuZGVyIHNldHRpbmdzLiAqL1xuaW50ZXJmYWNlIElSZW5kZXJTZXR0aW5ncyB7XG4gIHBhZGRpbmc6IG51bWJlcltdO1xuICB3aWR0aDogbnVtYmVyW107XG4gIGNvbHVtbnM6IG51bWJlcjtcbiAgaGFzQm9yZGVyOiBib29sZWFuO1xuICBoYXNIZWFkZXJCb3JkZXI6IGJvb2xlYW47XG4gIGhhc0JvZHlCb3JkZXI6IGJvb2xlYW47XG4gIHJvd3M6IFJvdzxDZWxsPltdO1xufVxuXG4vKiogVGFibGUgbGF5b3V0IHJlbmRlcmVyLiAqL1xuZXhwb3J0IGNsYXNzIFRhYmxlTGF5b3V0IHtcbiAgLyoqXG4gICAqIFRhYmxlIGxheW91dCBjb25zdHJ1Y3Rvci5cbiAgICogQHBhcmFtIHRhYmxlICAgVGFibGUgaW5zdGFuY2UuXG4gICAqIEBwYXJhbSBvcHRpb25zIFJlbmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIGNvbnN0cnVjdG9yKFxuICAgIHByaXZhdGUgdGFibGU6IFRhYmxlLFxuICAgIHByaXZhdGUgb3B0aW9uczogSVRhYmxlU2V0dGluZ3MsXG4gICkge31cblxuICAvKiogR2VuZXJhdGUgdGFibGUgc3RyaW5nLiAqL1xuICBwdWJsaWMgdG9TdHJpbmcoKTogc3RyaW5nIHtcbiAgICBjb25zdCBvcHRzOiBJUmVuZGVyU2V0dGluZ3MgPSB0aGlzLmNyZWF0ZUxheW91dCgpO1xuICAgIHJldHVybiBvcHRzLnJvd3MubGVuZ3RoID8gdGhpcy5yZW5kZXJSb3dzKG9wdHMpIDogXCJcIjtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZW5lcmF0ZXMgdGFibGUgbGF5b3V0IGluY2x1ZGluZyByb3cgYW5kIGNvbCBzcGFuLCBjb252ZXJ0cyBhbGwgbm9uZVxuICAgKiBDZWxsL1JvdyB2YWx1ZXMgdG8gQ2VsbHMgYW5kIFJvd3MgYW5kIHJldHVybnMgdGhlIGxheW91dCByZW5kZXJpbmdcbiAgICogc2V0dGluZ3MuXG4gICAqL1xuICBwcm90ZWN0ZWQgY3JlYXRlTGF5b3V0KCk6IElSZW5kZXJTZXR0aW5ncyB7XG4gICAgT2JqZWN0LmtleXModGhpcy5vcHRpb25zLmNoYXJzKS5mb3JFYWNoKChrZXk6IHN0cmluZykgPT4ge1xuICAgICAgaWYgKHR5cGVvZiB0aGlzLm9wdGlvbnMuY2hhcnNba2V5IGFzIGtleW9mIElCb3JkZXJPcHRpb25zXSAhPT0gXCJzdHJpbmdcIikge1xuICAgICAgICB0aGlzLm9wdGlvbnMuY2hhcnNba2V5IGFzIGtleW9mIElCb3JkZXJPcHRpb25zXSA9IFwiXCI7XG4gICAgICB9XG4gICAgfSk7XG5cbiAgICBjb25zdCBoYXNCb2R5Qm9yZGVyOiBib29sZWFuID0gdGhpcy50YWJsZS5nZXRCb3JkZXIoKSB8fFxuICAgICAgdGhpcy50YWJsZS5oYXNCb2R5Qm9yZGVyKCk7XG4gICAgY29uc3QgaGFzSGVhZGVyQm9yZGVyOiBib29sZWFuID0gdGhpcy50YWJsZS5oYXNIZWFkZXJCb3JkZXIoKTtcbiAgICBjb25zdCBoYXNCb3JkZXI6IGJvb2xlYW4gPSBoYXNIZWFkZXJCb3JkZXIgfHwgaGFzQm9keUJvcmRlcjtcblxuICAgIGNvbnN0IHJvd3MgPSB0aGlzLiNnZXRSb3dzKCk7XG5cbiAgICBjb25zdCBjb2x1bW5zOiBudW1iZXIgPSBNYXRoLm1heCguLi5yb3dzLm1hcCgocm93KSA9PiByb3cubGVuZ3RoKSk7XG4gICAgZm9yIChjb25zdCByb3cgb2Ygcm93cykge1xuICAgICAgY29uc3QgbGVuZ3RoOiBudW1iZXIgPSByb3cubGVuZ3RoO1xuICAgICAgaWYgKGxlbmd0aCA8IGNvbHVtbnMpIHtcbiAgICAgICAgY29uc3QgZGlmZiA9IGNvbHVtbnMgLSBsZW5ndGg7XG4gICAgICAgIGZvciAobGV0IGkgPSAwOyBpIDwgZGlmZjsgaSsrKSB7XG4gICAgICAgICAgcm93LnB1c2godGhpcy5jcmVhdGVDZWxsKG51bGwsIHJvdykpO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgY29uc3QgcGFkZGluZzogbnVtYmVyW10gPSBbXTtcbiAgICBjb25zdCB3aWR0aDogbnVtYmVyW10gPSBbXTtcbiAgICBmb3IgKGxldCBjb2xJbmRleCA9IDA7IGNvbEluZGV4IDwgY29sdW1uczsgY29sSW5kZXgrKykge1xuICAgICAgY29uc3QgbWluQ29sV2lkdGg6IG51bWJlciA9IEFycmF5LmlzQXJyYXkodGhpcy5vcHRpb25zLm1pbkNvbFdpZHRoKVxuICAgICAgICA/IHRoaXMub3B0aW9ucy5taW5Db2xXaWR0aFtjb2xJbmRleF1cbiAgICAgICAgOiB0aGlzLm9wdGlvbnMubWluQ29sV2lkdGg7XG4gICAgICBjb25zdCBtYXhDb2xXaWR0aDogbnVtYmVyID0gQXJyYXkuaXNBcnJheSh0aGlzLm9wdGlvbnMubWF4Q29sV2lkdGgpXG4gICAgICAgID8gdGhpcy5vcHRpb25zLm1heENvbFdpZHRoW2NvbEluZGV4XVxuICAgICAgICA6IHRoaXMub3B0aW9ucy5tYXhDb2xXaWR0aDtcbiAgICAgIGNvbnN0IGNvbFdpZHRoOiBudW1iZXIgPSBsb25nZXN0KGNvbEluZGV4LCByb3dzLCBtYXhDb2xXaWR0aCk7XG4gICAgICB3aWR0aFtjb2xJbmRleF0gPSBNYXRoLm1pbihtYXhDb2xXaWR0aCwgTWF0aC5tYXgobWluQ29sV2lkdGgsIGNvbFdpZHRoKSk7XG4gICAgICBwYWRkaW5nW2NvbEluZGV4XSA9IEFycmF5LmlzQXJyYXkodGhpcy5vcHRpb25zLnBhZGRpbmcpXG4gICAgICAgID8gdGhpcy5vcHRpb25zLnBhZGRpbmdbY29sSW5kZXhdXG4gICAgICAgIDogdGhpcy5vcHRpb25zLnBhZGRpbmc7XG4gICAgfVxuXG4gICAgcmV0dXJuIHtcbiAgICAgIHBhZGRpbmcsXG4gICAgICB3aWR0aCxcbiAgICAgIHJvd3MsXG4gICAgICBjb2x1bW5zLFxuICAgICAgaGFzQm9yZGVyLFxuICAgICAgaGFzQm9keUJvcmRlcixcbiAgICAgIGhhc0hlYWRlckJvcmRlcixcbiAgICB9O1xuICB9XG5cbiAgI2dldFJvd3MoKTogQXJyYXk8Um93PENlbGw+PiB7XG4gICAgY29uc3QgaGVhZGVyOiBSb3cgfCB1bmRlZmluZWQgPSB0aGlzLnRhYmxlLmdldEhlYWRlcigpO1xuICAgIGNvbnN0IHJvd3MgPSBoZWFkZXIgPyBbaGVhZGVyLCAuLi50aGlzLnRhYmxlXSA6IHRoaXMudGFibGUuc2xpY2UoKTtcbiAgICBjb25zdCBoYXNTcGFuID0gcm93cy5zb21lKChyb3cpID0+XG4gICAgICByb3cuc29tZSgoY2VsbCkgPT5cbiAgICAgICAgY2VsbCBpbnN0YW5jZW9mIENlbGwgJiYgKGNlbGwuZ2V0Q29sU3BhbigpID4gMSB8fCBjZWxsLmdldFJvd1NwYW4oKSA+IDEpXG4gICAgICApXG4gICAgKTtcblxuICAgIGlmIChoYXNTcGFuKSB7XG4gICAgICByZXR1cm4gdGhpcy5zcGFuUm93cyhyb3dzKTtcbiAgICB9XG5cbiAgICByZXR1cm4gcm93cy5tYXAoKHJvdykgPT4ge1xuICAgICAgY29uc3QgbmV3Um93ID0gdGhpcy5jcmVhdGVSb3cocm93KTtcbiAgICAgIGZvciAobGV0IGkgPSAwOyBpIDwgcm93Lmxlbmd0aDsgaSsrKSB7XG4gICAgICAgIG5ld1Jvd1tpXSA9IHRoaXMuY3JlYXRlQ2VsbChyb3dbaV0sIG5ld1Jvdyk7XG4gICAgICB9XG4gICAgICByZXR1cm4gbmV3Um93O1xuICAgIH0pO1xuICB9XG5cbiAgLyoqXG4gICAqIEZpbGxzIHJvd3MgYW5kIGNvbHMgYnkgc3BlY2lmaWVkIHJvdy9jb2wgc3BhbiB3aXRoIGEgcmVmZXJlbmNlIG9mIHRoZVxuICAgKiBvcmlnaW5hbCBjZWxsLlxuICAgKi9cbiAgcHJvdGVjdGVkIHNwYW5Sb3dzKHJvd3M6IEFycmF5PElSb3c+KSB7XG4gICAgY29uc3Qgcm93U3BhbjogQXJyYXk8bnVtYmVyPiA9IFtdO1xuICAgIGxldCBjb2xTcGFuID0gMTtcbiAgICBsZXQgcm93SW5kZXggPSAtMTtcblxuICAgIHdoaWxlICh0cnVlKSB7XG4gICAgICByb3dJbmRleCsrO1xuICAgICAgaWYgKHJvd0luZGV4ID09PSByb3dzLmxlbmd0aCAmJiByb3dTcGFuLmV2ZXJ5KChzcGFuKSA9PiBzcGFuID09PSAxKSkge1xuICAgICAgICBicmVhaztcbiAgICAgIH1cbiAgICAgIGNvbnN0IHJvdyA9IHJvd3Nbcm93SW5kZXhdID0gdGhpcy5jcmVhdGVSb3cocm93c1tyb3dJbmRleF0gfHwgW10pO1xuICAgICAgbGV0IGNvbEluZGV4ID0gLTE7XG5cbiAgICAgIHdoaWxlICh0cnVlKSB7XG4gICAgICAgIGNvbEluZGV4Kys7XG4gICAgICAgIGlmIChcbiAgICAgICAgICBjb2xJbmRleCA9PT0gcm93Lmxlbmd0aCAmJlxuICAgICAgICAgIGNvbEluZGV4ID09PSByb3dTcGFuLmxlbmd0aCAmJiBjb2xTcGFuID09PSAxXG4gICAgICAgICkge1xuICAgICAgICAgIGJyZWFrO1xuICAgICAgICB9XG5cbiAgICAgICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICAgICAgY29sU3Bhbi0tO1xuICAgICAgICAgIHJvd1NwYW5bY29sSW5kZXhdID0gcm93U3Bhbltjb2xJbmRleCAtIDFdO1xuICAgICAgICAgIHJvdy5zcGxpY2UoXG4gICAgICAgICAgICBjb2xJbmRleCxcbiAgICAgICAgICAgIHRoaXMuZ2V0RGVsZXRlQ291bnQocm93cywgcm93SW5kZXgsIGNvbEluZGV4KSxcbiAgICAgICAgICAgIHJvd1tjb2xJbmRleCAtIDFdLFxuICAgICAgICAgICk7XG5cbiAgICAgICAgICBjb250aW51ZTtcbiAgICAgICAgfVxuXG4gICAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgICByb3dTcGFuW2NvbEluZGV4XS0tO1xuICAgICAgICAgIHJvd3Nbcm93SW5kZXhdLnNwbGljZShcbiAgICAgICAgICAgIGNvbEluZGV4LFxuICAgICAgICAgICAgdGhpcy5nZXREZWxldGVDb3VudChyb3dzLCByb3dJbmRleCwgY29sSW5kZXgpLFxuICAgICAgICAgICAgcm93c1tyb3dJbmRleCAtIDFdW2NvbEluZGV4XSxcbiAgICAgICAgICApO1xuXG4gICAgICAgICAgY29udGludWU7XG4gICAgICAgIH1cblxuICAgICAgICBjb25zdCBjZWxsID0gcm93W2NvbEluZGV4XSA9IHRoaXMuY3JlYXRlQ2VsbChcbiAgICAgICAgICByb3dbY29sSW5kZXhdIHx8IG51bGwsXG4gICAgICAgICAgcm93LFxuICAgICAgICApO1xuXG4gICAgICAgIGNvbFNwYW4gPSBjZWxsLmdldENvbFNwYW4oKTtcbiAgICAgICAgcm93U3Bhbltjb2xJbmRleF0gPSBjZWxsLmdldFJvd1NwYW4oKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4gcm93cyBhcyBBcnJheTxSb3c8Q2VsbD4+O1xuICB9XG5cbiAgcHJvdGVjdGVkIGdldERlbGV0ZUNvdW50KFxuICAgIHJvd3M6IEFycmF5PEFycmF5PHVua25vd24+PixcbiAgICByb3dJbmRleDogbnVtYmVyLFxuICAgIGNvbEluZGV4OiBudW1iZXIsXG4gICkge1xuICAgIHJldHVybiBjb2xJbmRleCA8PSByb3dzW3Jvd0luZGV4XS5sZW5ndGggLSAxICYmXG4gICAgICAgIHR5cGVvZiByb3dzW3Jvd0luZGV4XVtjb2xJbmRleF0gPT09IFwidW5kZWZpbmVkXCJcbiAgICAgID8gMVxuICAgICAgOiAwO1xuICB9XG5cbiAgLyoqXG4gICAqIENyZWF0ZSBhIG5ldyByb3cgZnJvbSBleGlzdGluZyByb3cgb3IgY2VsbCBhcnJheS5cbiAgICogQHBhcmFtIHJvdyBPcmlnaW5hbCByb3cuXG4gICAqL1xuICBwcm90ZWN0ZWQgY3JlYXRlUm93KHJvdzogSVJvdyk6IFJvdzxDZWxsPiB7XG4gICAgcmV0dXJuIFJvdy5mcm9tKHJvdylcbiAgICAgIC5ib3JkZXIodGhpcy50YWJsZS5nZXRCb3JkZXIoKSwgZmFsc2UpXG4gICAgICAuYWxpZ24odGhpcy50YWJsZS5nZXRBbGlnbigpLCBmYWxzZSkgYXMgUm93PENlbGw+O1xuICB9XG5cbiAgLyoqXG4gICAqIENyZWF0ZSBhIG5ldyBjZWxsIGZyb20gZXhpc3RpbmcgY2VsbCBvciBjZWxsIHZhbHVlLlxuICAgKiBAcGFyYW0gY2VsbCAgT3JpZ2luYWwgY2VsbC5cbiAgICogQHBhcmFtIHJvdyAgIFBhcmVudCByb3cuXG4gICAqL1xuICBwcm90ZWN0ZWQgY3JlYXRlQ2VsbChjZWxsOiBJQ2VsbCB8IG51bGwgfCB1bmRlZmluZWQsIHJvdzogUm93KTogQ2VsbCB7XG4gICAgcmV0dXJuIENlbGwuZnJvbShjZWxsID8/IFwiXCIpXG4gICAgICAuYm9yZGVyKHJvdy5nZXRCb3JkZXIoKSwgZmFsc2UpXG4gICAgICAuYWxpZ24ocm93LmdldEFsaWduKCksIGZhbHNlKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgdGFibGUgbGF5b3V0LlxuICAgKiBAcGFyYW0gb3B0cyBSZW5kZXIgb3B0aW9ucy5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJSb3dzKG9wdHM6IElSZW5kZXJTZXR0aW5ncyk6IHN0cmluZyB7XG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG4gICAgY29uc3Qgcm93U3BhbjogbnVtYmVyW10gPSBuZXcgQXJyYXkob3B0cy5jb2x1bW5zKS5maWxsKDEpO1xuXG4gICAgZm9yIChsZXQgcm93SW5kZXggPSAwOyByb3dJbmRleCA8IG9wdHMucm93cy5sZW5ndGg7IHJvd0luZGV4KyspIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlclJvdyhyb3dTcGFuLCByb3dJbmRleCwgb3B0cyk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdC5zbGljZSgwLCAtMSk7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIHJvdy5cbiAgICogQHBhcmFtIHJvd1NwYW4gICAgIEN1cnJlbnQgcm93IHNwYW4uXG4gICAqIEBwYXJhbSByb3dJbmRleCAgICBDdXJyZW50IHJvdyBpbmRleC5cbiAgICogQHBhcmFtIG9wdHMgICAgICAgIFJlbmRlciBvcHRpb25zLlxuICAgKiBAcGFyYW0gaXNNdWx0aWxpbmUgSXMgbXVsdGlsaW5lIHJvdy5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJSb3coXG4gICAgcm93U3BhbjogbnVtYmVyW10sXG4gICAgcm93SW5kZXg6IG51bWJlcixcbiAgICBvcHRzOiBJUmVuZGVyU2V0dGluZ3MsXG4gICAgaXNNdWx0aWxpbmU/OiBib29sZWFuLFxuICApOiBzdHJpbmcge1xuICAgIGNvbnN0IHJvdzogUm93PENlbGw+ID0gb3B0cy5yb3dzW3Jvd0luZGV4XTtcbiAgICBjb25zdCBwcmV2Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQgPSBvcHRzLnJvd3Nbcm93SW5kZXggLSAxXTtcbiAgICBjb25zdCBuZXh0Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQgPSBvcHRzLnJvd3Nbcm93SW5kZXggKyAxXTtcbiAgICBsZXQgcmVzdWx0ID0gXCJcIjtcblxuICAgIGxldCBjb2xTcGFuID0gMTtcblxuICAgIC8vIGJvcmRlciB0b3Agcm93XG4gICAgaWYgKCFpc011bHRpbGluZSAmJiByb3dJbmRleCA9PT0gMCAmJiByb3cuaGFzQm9yZGVyKCkpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckJvcmRlclJvdyh1bmRlZmluZWQsIHJvdywgcm93U3Bhbiwgb3B0cyk7XG4gICAgfVxuXG4gICAgbGV0IGlzTXVsdGlsaW5lUm93ID0gZmFsc2U7XG5cbiAgICByZXN1bHQgKz0gXCIgXCIucmVwZWF0KHRoaXMub3B0aW9ucy5pbmRlbnQgfHwgMCk7XG5cbiAgICBmb3IgKGxldCBjb2xJbmRleCA9IDA7IGNvbEluZGV4IDwgb3B0cy5jb2x1bW5zOyBjb2xJbmRleCsrKSB7XG4gICAgICBpZiAoY29sU3BhbiA+IDEpIHtcbiAgICAgICAgY29sU3Bhbi0tO1xuICAgICAgICByb3dTcGFuW2NvbEluZGV4XSA9IHJvd1NwYW5bY29sSW5kZXggLSAxXTtcbiAgICAgICAgY29udGludWU7XG4gICAgICB9XG5cbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckNlbGwoY29sSW5kZXgsIHJvdywgb3B0cyk7XG5cbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgaWYgKCFpc011bHRpbGluZSkge1xuICAgICAgICAgIHJvd1NwYW5bY29sSW5kZXhdLS07XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoIXByZXZSb3cgfHwgcHJldlJvd1tjb2xJbmRleF0gIT09IHJvd1tjb2xJbmRleF0pIHtcbiAgICAgICAgcm93U3Bhbltjb2xJbmRleF0gPSByb3dbY29sSW5kZXhdLmdldFJvd1NwYW4oKTtcbiAgICAgIH1cblxuICAgICAgY29sU3BhbiA9IHJvd1tjb2xJbmRleF0uZ2V0Q29sU3BhbigpO1xuXG4gICAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPT09IDEgJiYgcm93W2NvbEluZGV4XS5sZW5ndGgpIHtcbiAgICAgICAgaXNNdWx0aWxpbmVSb3cgPSB0cnVlO1xuICAgICAgfVxuICAgIH1cblxuICAgIGlmIChvcHRzLmNvbHVtbnMgPiAwKSB7XG4gICAgICBpZiAocm93W29wdHMuY29sdW1ucyAtIDFdLmdldEJvcmRlcigpKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHQ7XG4gICAgICB9IGVsc2UgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXN1bHQgKz0gXCJcXG5cIjtcblxuICAgIGlmIChpc011bHRpbGluZVJvdykgeyAvLyBza2lwIGJvcmRlclxuICAgICAgcmV0dXJuIHJlc3VsdCArIHRoaXMucmVuZGVyUm93KHJvd1NwYW4sIHJvd0luZGV4LCBvcHRzLCBpc011bHRpbGluZVJvdyk7XG4gICAgfVxuXG4gICAgLy8gYm9yZGVyIG1pZCByb3dcbiAgICBpZiAoXG4gICAgICAocm93SW5kZXggPT09IDAgJiYgb3B0cy5oYXNIZWFkZXJCb3JkZXIpIHx8XG4gICAgICAocm93SW5kZXggPCBvcHRzLnJvd3MubGVuZ3RoIC0gMSAmJiBvcHRzLmhhc0JvZHlCb3JkZXIpXG4gICAgKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJSb3cocm93LCBuZXh0Um93LCByb3dTcGFuLCBvcHRzKTtcbiAgICB9XG5cbiAgICAvLyBib3JkZXIgYm90dG9tIHJvd1xuICAgIGlmIChyb3dJbmRleCA9PT0gb3B0cy5yb3dzLmxlbmd0aCAtIDEgJiYgcm93Lmhhc0JvcmRlcigpKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJSb3cocm93LCB1bmRlZmluZWQsIHJvd1NwYW4sIG9wdHMpO1xuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQ7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIGNlbGwuXG4gICAqIEBwYXJhbSBjb2xJbmRleCAgQ3VycmVudCBjb2wgaW5kZXguXG4gICAqIEBwYXJhbSByb3cgICAgICAgQ3VycmVudCByb3cuXG4gICAqIEBwYXJhbSBvcHRzICAgICAgUmVuZGVyIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBub0JvcmRlciAgRGlzYWJsZSBib3JkZXIuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQ2VsbChcbiAgICBjb2xJbmRleDogbnVtYmVyLFxuICAgIHJvdzogUm93PENlbGw+LFxuICAgIG9wdHM6IElSZW5kZXJTZXR0aW5ncyxcbiAgICBub0JvcmRlcj86IGJvb2xlYW4sXG4gICk6IHN0cmluZyB7XG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG4gICAgY29uc3QgcHJldkNlbGw6IENlbGwgfCB1bmRlZmluZWQgPSByb3dbY29sSW5kZXggLSAxXTtcblxuICAgIGNvbnN0IGNlbGw6IENlbGwgPSByb3dbY29sSW5kZXhdO1xuXG4gICAgaWYgKCFub0JvcmRlcikge1xuICAgICAgaWYgKGNvbEluZGV4ID09PSAwKSB7XG4gICAgICAgIGlmIChjZWxsLmdldEJvcmRlcigpKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0O1xuICAgICAgICB9IGVsc2UgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBpZiAoY2VsbC5nZXRCb3JkZXIoKSB8fCBwcmV2Q2VsbD8uZ2V0Qm9yZGVyKCkpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLm1pZGRsZTtcbiAgICAgICAgfSBlbHNlIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIGxldCBtYXhMZW5ndGg6IG51bWJlciA9IG9wdHMud2lkdGhbY29sSW5kZXhdO1xuXG4gICAgY29uc3QgY29sU3BhbjogbnVtYmVyID0gY2VsbC5nZXRDb2xTcGFuKCk7XG4gICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICBmb3IgKGxldCBvID0gMTsgbyA8IGNvbFNwYW47IG8rKykge1xuICAgICAgICAvLyBhZGQgcGFkZGluZyBhbmQgd2l0aCBvZiBuZXh0IGNlbGxcbiAgICAgICAgbWF4TGVuZ3RoICs9IG9wdHMud2lkdGhbY29sSW5kZXggKyBvXSArIG9wdHMucGFkZGluZ1tjb2xJbmRleCArIG9dO1xuICAgICAgICBpZiAob3B0cy5oYXNCb3JkZXIpIHtcbiAgICAgICAgICAvLyBhZGQgcGFkZGluZyBhZ2FpbiBhbmQgYm9yZGVyIHdpdGhcbiAgICAgICAgICBtYXhMZW5ndGggKz0gb3B0cy5wYWRkaW5nW2NvbEluZGV4ICsgb10gKyAxO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgY29uc3QgeyBjdXJyZW50LCBuZXh0IH0gPSB0aGlzLnJlbmRlckNlbGxWYWx1ZShjZWxsLCBtYXhMZW5ndGgpO1xuXG4gICAgcm93W2NvbEluZGV4XS5zZXRWYWx1ZShuZXh0KTtcblxuICAgIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgcmVzdWx0ICs9IFwiIFwiLnJlcGVhdChvcHRzLnBhZGRpbmdbY29sSW5kZXhdKTtcbiAgICB9XG5cbiAgICByZXN1bHQgKz0gY3VycmVudDtcblxuICAgIGlmIChvcHRzLmhhc0JvcmRlciB8fCBjb2xJbmRleCA8IG9wdHMuY29sdW1ucyAtIDEpIHtcbiAgICAgIHJlc3VsdCArPSBcIiBcIi5yZXBlYXQob3B0cy5wYWRkaW5nW2NvbEluZGV4XSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdDtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgc3BlY2lmaWVkIGxlbmd0aCBvZiBjZWxsLiBSZXR1cm5zIHRoZSByZW5kZXJlZCB2YWx1ZSBhbmQgYSBuZXcgY2VsbFxuICAgKiB3aXRoIHRoZSByZXN0IHZhbHVlLlxuICAgKiBAcGFyYW0gY2VsbCAgICAgIENlbGwgdG8gcmVuZGVyLlxuICAgKiBAcGFyYW0gbWF4TGVuZ3RoIE1heCBsZW5ndGggb2YgY29udGVudCB0byByZW5kZXIuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQ2VsbFZhbHVlKFxuICAgIGNlbGw6IENlbGwsXG4gICAgbWF4TGVuZ3RoOiBudW1iZXIsXG4gICk6IHsgY3VycmVudDogc3RyaW5nOyBuZXh0OiBDZWxsIH0ge1xuICAgIGNvbnN0IGxlbmd0aDogbnVtYmVyID0gTWF0aC5taW4oXG4gICAgICBtYXhMZW5ndGgsXG4gICAgICBzdHJMZW5ndGgoY2VsbC50b1N0cmluZygpKSxcbiAgICApO1xuICAgIGxldCB3b3Jkczogc3RyaW5nID0gY29uc3VtZVdvcmRzKGxlbmd0aCwgY2VsbC50b1N0cmluZygpKTtcblxuICAgIC8vIGJyZWFrIHdvcmQgaWYgd29yZCBpcyBsb25nZXIgdGhhbiBtYXggbGVuZ3RoXG4gICAgY29uc3QgYnJlYWtXb3JkID0gc3RyTGVuZ3RoKHdvcmRzKSA+IGxlbmd0aDtcbiAgICBpZiAoYnJlYWtXb3JkKSB7XG4gICAgICB3b3JkcyA9IHdvcmRzLnNsaWNlKDAsIGxlbmd0aCk7XG4gICAgfVxuXG4gICAgLy8gZ2V0IG5leHQgY29udGVudCBhbmQgcmVtb3ZlIGxlYWRpbmcgc3BhY2UgaWYgYnJlYWtXb3JkIGlzIG5vdCB0cnVlXG4gICAgY29uc3QgbmV4dCA9IGNlbGwudG9TdHJpbmcoKS5zbGljZSh3b3Jkcy5sZW5ndGggKyAoYnJlYWtXb3JkID8gMCA6IDEpKTtcbiAgICBjb25zdCBmaWxsTGVuZ3RoID0gbWF4TGVuZ3RoIC0gc3RyTGVuZ3RoKHdvcmRzKTtcblxuICAgIC8vIEFsaWduIGNvbnRlbnRcbiAgICBjb25zdCBhbGlnbjogRGlyZWN0aW9uID0gY2VsbC5nZXRBbGlnbigpO1xuICAgIGxldCBjdXJyZW50OiBzdHJpbmc7XG4gICAgaWYgKGZpbGxMZW5ndGggPT09IDApIHtcbiAgICAgIGN1cnJlbnQgPSB3b3JkcztcbiAgICB9IGVsc2UgaWYgKGFsaWduID09PSBcImxlZnRcIikge1xuICAgICAgY3VycmVudCA9IHdvcmRzICsgXCIgXCIucmVwZWF0KGZpbGxMZW5ndGgpO1xuICAgIH0gZWxzZSBpZiAoYWxpZ24gPT09IFwiY2VudGVyXCIpIHtcbiAgICAgIGN1cnJlbnQgPSBcIiBcIi5yZXBlYXQoTWF0aC5mbG9vcihmaWxsTGVuZ3RoIC8gMikpICsgd29yZHMgK1xuICAgICAgICBcIiBcIi5yZXBlYXQoTWF0aC5jZWlsKGZpbGxMZW5ndGggLyAyKSk7XG4gICAgfSBlbHNlIGlmIChhbGlnbiA9PT0gXCJyaWdodFwiKSB7XG4gICAgICBjdXJyZW50ID0gXCIgXCIucmVwZWF0KGZpbGxMZW5ndGgpICsgd29yZHM7XG4gICAgfSBlbHNlIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihcIlVua25vd24gZGlyZWN0aW9uOiBcIiArIGFsaWduKTtcbiAgICB9XG5cbiAgICByZXR1cm4ge1xuICAgICAgY3VycmVudCxcbiAgICAgIG5leHQ6IGNlbGwuY2xvbmUobmV4dCksXG4gICAgfTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgYm9yZGVyIHJvdy5cbiAgICogQHBhcmFtIHByZXZSb3cgUHJldmlvdXMgcm93LlxuICAgKiBAcGFyYW0gbmV4dFJvdyBOZXh0IHJvdy5cbiAgICogQHBhcmFtIHJvd1NwYW4gQ3VycmVudCByb3cgc3Bhbi5cbiAgICogQHBhcmFtIG9wdHMgICAgUmVuZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQm9yZGVyUm93KFxuICAgIHByZXZSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCxcbiAgICBuZXh0Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQsXG4gICAgcm93U3BhbjogbnVtYmVyW10sXG4gICAgb3B0czogSVJlbmRlclNldHRpbmdzLFxuICApOiBzdHJpbmcge1xuICAgIGxldCByZXN1bHQgPSBcIlwiO1xuXG4gICAgbGV0IGNvbFNwYW4gPSAxO1xuICAgIGZvciAobGV0IGNvbEluZGV4ID0gMDsgY29sSW5kZXggPCBvcHRzLmNvbHVtbnM7IGNvbEluZGV4KyspIHtcbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgaWYgKCFuZXh0Um93KSB7XG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiaW52YWxpZCBsYXlvdXRcIik7XG4gICAgICAgIH1cbiAgICAgICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICAgICAgY29sU3Bhbi0tO1xuICAgICAgICAgIGNvbnRpbnVlO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJDZWxsKFxuICAgICAgICBjb2xJbmRleCxcbiAgICAgICAgcHJldlJvdyxcbiAgICAgICAgbmV4dFJvdyxcbiAgICAgICAgcm93U3BhbixcbiAgICAgICAgb3B0cyxcbiAgICAgICk7XG4gICAgICBjb2xTcGFuID0gbmV4dFJvdz8uW2NvbEluZGV4XS5nZXRDb2xTcGFuKCkgPz8gMTtcbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0Lmxlbmd0aCA/IFwiIFwiLnJlcGVhdCh0aGlzLm9wdGlvbnMuaW5kZW50KSArIHJlc3VsdCArIFwiXFxuXCIgOiBcIlwiO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciBib3JkZXIgY2VsbC5cbiAgICogQHBhcmFtIGNvbEluZGV4ICBDdXJyZW50IGluZGV4LlxuICAgKiBAcGFyYW0gcHJldlJvdyAgIFByZXZpb3VzIHJvdy5cbiAgICogQHBhcmFtIG5leHRSb3cgICBOZXh0IHJvdy5cbiAgICogQHBhcmFtIHJvd1NwYW4gICBDdXJyZW50IHJvdyBzcGFuLlxuICAgKiBAcGFyYW0gb3B0cyAgICAgIFJlbmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlckJvcmRlckNlbGwoXG4gICAgY29sSW5kZXg6IG51bWJlcixcbiAgICBwcmV2Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQsXG4gICAgbmV4dFJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkLFxuICAgIHJvd1NwYW46IG51bWJlcltdLFxuICAgIG9wdHM6IElSZW5kZXJTZXR0aW5ncyxcbiAgKTogc3RyaW5nIHtcbiAgICAvLyBhMSB8IGIxXG4gICAgLy8gLS0tLS0tLVxuICAgIC8vIGEyIHwgYjJcblxuICAgIGNvbnN0IGExOiBDZWxsIHwgdW5kZWZpbmVkID0gcHJldlJvdz8uW2NvbEluZGV4IC0gMV07XG4gICAgY29uc3QgYTI6IENlbGwgfCB1bmRlZmluZWQgPSBuZXh0Um93Py5bY29sSW5kZXggLSAxXTtcbiAgICBjb25zdCBiMTogQ2VsbCB8IHVuZGVmaW5lZCA9IHByZXZSb3c/Lltjb2xJbmRleF07XG4gICAgY29uc3QgYjI6IENlbGwgfCB1bmRlZmluZWQgPSBuZXh0Um93Py5bY29sSW5kZXhdO1xuXG4gICAgY29uc3QgYTFCb3JkZXIgPSAhIWExPy5nZXRCb3JkZXIoKTtcbiAgICBjb25zdCBhMkJvcmRlciA9ICEhYTI/LmdldEJvcmRlcigpO1xuICAgIGNvbnN0IGIxQm9yZGVyID0gISFiMT8uZ2V0Qm9yZGVyKCk7XG4gICAgY29uc3QgYjJCb3JkZXIgPSAhIWIyPy5nZXRCb3JkZXIoKTtcblxuICAgIGNvbnN0IGhhc0NvbFNwYW4gPSAoY2VsbDogQ2VsbCB8IHVuZGVmaW5lZCk6IGJvb2xlYW4gPT5cbiAgICAgIChjZWxsPy5nZXRDb2xTcGFuKCkgPz8gMSkgPiAxO1xuICAgIGNvbnN0IGhhc1Jvd1NwYW4gPSAoY2VsbDogQ2VsbCB8IHVuZGVmaW5lZCk6IGJvb2xlYW4gPT5cbiAgICAgIChjZWxsPy5nZXRSb3dTcGFuKCkgPz8gMSkgPiAxO1xuXG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG5cbiAgICBpZiAoY29sSW5kZXggPT09IDApIHtcbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0O1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChiMUJvcmRlciAmJiBiMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnRNaWQ7XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTGVmdDtcbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BMZWZ0O1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgfVxuICAgIH0gZWxzZSBpZiAoY29sSW5kZXggPCBvcHRzLmNvbHVtbnMpIHtcbiAgICAgIGlmICgoYTFCb3JkZXIgJiYgYjJCb3JkZXIpIHx8IChiMUJvcmRlciAmJiBhMkJvcmRlcikpIHtcbiAgICAgICAgY29uc3QgYTFDb2xTcGFuOiBib29sZWFuID0gaGFzQ29sU3BhbihhMSk7XG4gICAgICAgIGNvbnN0IGEyQ29sU3BhbjogYm9vbGVhbiA9IGhhc0NvbFNwYW4oYTIpO1xuICAgICAgICBjb25zdCBiMUNvbFNwYW46IGJvb2xlYW4gPSBoYXNDb2xTcGFuKGIxKTtcbiAgICAgICAgY29uc3QgYjJDb2xTcGFuOiBib29sZWFuID0gaGFzQ29sU3BhbihiMik7XG5cbiAgICAgICAgY29uc3QgYTFSb3dTcGFuOiBib29sZWFuID0gaGFzUm93U3BhbihhMSk7XG4gICAgICAgIGNvbnN0IGEyUm93U3BhbjogYm9vbGVhbiA9IGhhc1Jvd1NwYW4oYTIpO1xuICAgICAgICBjb25zdCBiMVJvd1NwYW46IGJvb2xlYW4gPSBoYXNSb3dTcGFuKGIxKTtcbiAgICAgICAgY29uc3QgYjJSb3dTcGFuOiBib29sZWFuID0gaGFzUm93U3BhbihiMik7XG5cbiAgICAgICAgY29uc3QgaGFzQWxsQm9yZGVyID0gYTFCb3JkZXIgJiYgYjJCb3JkZXIgJiYgYjFCb3JkZXIgJiYgYTJCb3JkZXI7XG4gICAgICAgIGNvbnN0IGhhc0FsbFJvd1NwYW4gPSBhMVJvd1NwYW4gJiYgYjFSb3dTcGFuICYmIGEyUm93U3BhbiAmJiBiMlJvd1NwYW47XG4gICAgICAgIGNvbnN0IGhhc0FsbENvbFNwYW4gPSBhMUNvbFNwYW4gJiYgYjFDb2xTcGFuICYmIGEyQ29sU3BhbiAmJiBiMkNvbFNwYW47XG5cbiAgICAgICAgaWYgKGhhc0FsbFJvd1NwYW4gJiYgaGFzQWxsQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWRkbGU7XG4gICAgICAgIH0gZWxzZSBpZiAoaGFzQWxsQ29sU3BhbiAmJiBoYXNBbGxCb3JkZXIgJiYgYTEgPT09IGIxICYmIGEyID09PSBiMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkO1xuICAgICAgICB9IGVsc2UgaWYgKGExQ29sU3BhbiAmJiBiMUNvbFNwYW4gJiYgYTEgPT09IGIxKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BNaWQ7XG4gICAgICAgIH0gZWxzZSBpZiAoYTJDb2xTcGFuICYmIGIyQ29sU3BhbiAmJiBhMiA9PT0gYjIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbU1pZDtcbiAgICAgICAgfSBlbHNlIGlmIChhMVJvd1NwYW4gJiYgYTJSb3dTcGFuICYmIGExID09PSBhMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdE1pZDtcbiAgICAgICAgfSBlbHNlIGlmIChiMVJvd1NwYW4gJiYgYjJSb3dTcGFuICYmIGIxID09PSBiMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHRNaWQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWRNaWQ7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYTFCb3JkZXIgJiYgYjFCb3JkZXIpIHtcbiAgICAgICAgaWYgKGhhc0NvbFNwYW4oYTEpICYmIGhhc0NvbFNwYW4oYjEpICYmIGExID09PSBiMSkge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyICYmIGIyQm9yZGVyKSB7XG4gICAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0TWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGIyQm9yZGVyICYmIGEyQm9yZGVyKSB7XG4gICAgICAgIGlmIChoYXNDb2xTcGFuKGEyKSAmJiBoYXNDb2xTcGFuKGIyKSAmJiBhMiA9PT0gYjIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcE1pZDtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChhMUJvcmRlciAmJiBhMkJvcmRlcikge1xuICAgICAgICBpZiAoaGFzUm93U3BhbihhMSkgJiYgYTEgPT09IGEyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0TWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGExQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tUmlnaHQ7XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTGVmdDtcbiAgICAgIH0gZWxzZSBpZiAoYTJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BSaWdodDtcbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BMZWZ0O1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IGxlbmd0aCA9IG9wdHMucGFkZGluZ1tjb2xJbmRleF0gKyBvcHRzLndpZHRoW2NvbEluZGV4XSArXG4gICAgICBvcHRzLnBhZGRpbmdbY29sSW5kZXhdO1xuXG4gICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSAmJiBuZXh0Um93KSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJDZWxsKFxuICAgICAgICBjb2xJbmRleCxcbiAgICAgICAgbmV4dFJvdyxcbiAgICAgICAgb3B0cyxcbiAgICAgICAgdHJ1ZSxcbiAgICAgICk7XG4gICAgICBpZiAobmV4dFJvd1tjb2xJbmRleF0gPT09IG5leHRSb3dbbmV4dFJvdy5sZW5ndGggLSAxXSkge1xuICAgICAgICBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0O1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgICByZXR1cm4gcmVzdWx0O1xuICAgICAgfVxuICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIgJiYgYjJCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkLnJlcGVhdChsZW5ndGgpO1xuICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tLnJlcGVhdChsZW5ndGgpO1xuICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wLnJlcGVhdChsZW5ndGgpO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXN1bHQgKz0gXCIgXCIucmVwZWF0KGxlbmd0aCk7XG4gICAgfVxuXG4gICAgaWYgKGNvbEluZGV4ID09PSBvcHRzLmNvbHVtbnMgLSAxKSB7XG4gICAgICBpZiAoYjFCb3JkZXIgJiYgYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodE1pZDtcbiAgICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21SaWdodDtcbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BSaWdodDtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0O1xuICB9XG59XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsU0FBUyxJQUFJLFFBQTBCLFdBQVcsQ0FBQztBQUNuRCxTQUFlLEdBQUcsUUFBUSxVQUFVLENBQUM7QUFFckMsU0FBUyxZQUFZLEVBQUUsT0FBTyxFQUFFLFNBQVMsUUFBUSxZQUFZLENBQUM7QUFhOUQsMkJBQTJCLEdBQzNCLE9BQU8sTUFBTSxXQUFXO0lBQ3RCOzs7O0dBSUMsR0FDRCxZQUNVLEtBQVksRUFDWixPQUF1QixDQUMvQjtRQUZRLGFBQUEsS0FBWSxDQUFBO1FBQ1osZUFBQSxPQUF1QixDQUFBO0lBQzlCO0lBRUgsMkJBQTJCLEdBQ3BCLFFBQVEsR0FBVztRQUN4QixNQUFNLElBQUksR0FBb0IsSUFBSSxDQUFDLFlBQVksRUFBRSxBQUFDO1FBQ2xELE9BQU8sSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsSUFBSSxDQUFDLFVBQVUsQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLENBQUM7SUFDdkQ7SUFFQTs7OztHQUlDLEdBQ1MsWUFBWSxHQUFvQjtRQUN4QyxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxP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