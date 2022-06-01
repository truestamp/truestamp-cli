import { Cell } from "./cell.ts";
import { Row } from "./row.ts";
import { consumeWords, longest, strLength } from "./utils.ts";
export class TableLayout {
    table;
    options;
    constructor(table, options) {
        this.table = table;
        this.options = options;
    }
    toString() {
        const opts = this.createLayout();
        return opts.rows.length ? this.renderRows(opts) : "";
    }
    createLayout() {
        Object.keys(this.options.chars).forEach((key) => {
            if (typeof this.options.chars[key] !== "string") {
                this.options.chars[key] = "";
            }
        });
        const hasBodyBorder = this.table.getBorder() ||
            this.table.hasBodyBorder();
        const hasHeaderBorder = this.table.hasHeaderBorder();
        const hasBorder = hasHeaderBorder || hasBodyBorder;
        const rows = this.#getRows();
        const columns = Math.max(...rows.map((row) => row.length));
        for (const row of rows) {
            const length = row.length;
            if (length < columns) {
                const diff = columns - length;
                for (let i = 0; i < diff; i++) {
                    row.push(this.createCell(null, row));
                }
            }
        }
        const padding = [];
        const width = [];
        for (let colIndex = 0; colIndex < columns; colIndex++) {
            const minColWidth = Array.isArray(this.options.minColWidth)
                ? this.options.minColWidth[colIndex]
                : this.options.minColWidth;
            const maxColWidth = Array.isArray(this.options.maxColWidth)
                ? this.options.maxColWidth[colIndex]
                : this.options.maxColWidth;
            const colWidth = longest(colIndex, rows, maxColWidth);
            width[colIndex] = Math.min(maxColWidth, Math.max(minColWidth, colWidth));
            padding[colIndex] = Array.isArray(this.options.padding)
                ? this.options.padding[colIndex]
                : this.options.padding;
        }
        return {
            padding,
            width,
            rows,
            columns,
            hasBorder,
            hasBodyBorder,
            hasHeaderBorder,
        };
    }
    #getRows() {
        const header = this.table.getHeader();
        const rows = header ? [header, ...this.table] : this.table.slice();
        const hasSpan = rows.some((row) => row.some((cell) => cell instanceof Cell && (cell.getColSpan() > 1 || cell.getRowSpan() > 1)));
        if (hasSpan) {
            return this.spanRows(rows);
        }
        return rows.map((row) => {
            const newRow = this.createRow(row);
            return newRow.map((cell) => this.createCell(cell, newRow));
        });
    }
    spanRows(_rows, rowIndex = 0, colIndex = 0, rowSpan = [], colSpan = 1) {
        const rows = _rows;
        if (rowIndex >= rows.length && rowSpan.every((span) => span === 1)) {
            return rows;
        }
        else if (rows[rowIndex] && colIndex >= rows[rowIndex].length &&
            colIndex >= rowSpan.length && colSpan === 1) {
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
    createRow(row) {
        return Row.from(row)
            .border(this.table.getBorder(), false)
            .align(this.table.getAlign(), false);
    }
    createCell(cell, row) {
        return Cell.from(cell ?? "")
            .border(row.getBorder(), false)
            .align(row.getAlign(), false);
    }
    renderRows(opts) {
        let result = "";
        const rowSpan = new Array(opts.columns).fill(1);
        for (let rowIndex = 0; rowIndex < opts.rows.length; rowIndex++) {
            result += this.renderRow(rowSpan, rowIndex, opts);
        }
        return result.slice(0, -1);
    }
    renderRow(rowSpan, rowIndex, opts, isMultiline) {
        const row = opts.rows[rowIndex];
        const prevRow = opts.rows[rowIndex - 1];
        const nextRow = opts.rows[rowIndex + 1];
        let result = "";
        let colSpan = 1;
        if (!isMultiline && rowIndex === 0 && row.hasBorder()) {
            result += this.renderBorderRow(undefined, row, rowSpan, opts);
        }
        let isMultilineRow = false;
        result += " ".repeat(this.options.indent || 0);
        for (let colIndex = 0; colIndex < opts.columns; colIndex++) {
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
            }
            else if (!prevRow || prevRow[colIndex] !== row[colIndex]) {
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
            }
            else if (opts.hasBorder) {
                result += " ";
            }
        }
        result += "\n";
        if (isMultilineRow) {
            return result + this.renderRow(rowSpan, rowIndex, opts, isMultilineRow);
        }
        if ((rowIndex === 0 && opts.hasHeaderBorder) ||
            (rowIndex < opts.rows.length - 1 && opts.hasBodyBorder)) {
            result += this.renderBorderRow(row, nextRow, rowSpan, opts);
        }
        if (rowIndex === opts.rows.length - 1 && row.hasBorder()) {
            result += this.renderBorderRow(row, undefined, rowSpan, opts);
        }
        return result;
    }
    renderCell(colIndex, row, opts, noBorder) {
        let result = "";
        const prevCell = row[colIndex - 1];
        const cell = row[colIndex];
        if (!noBorder) {
            if (colIndex === 0) {
                if (cell.getBorder()) {
                    result += this.options.chars.left;
                }
                else if (opts.hasBorder) {
                    result += " ";
                }
            }
            else {
                if (cell.getBorder() || prevCell?.getBorder()) {
                    result += this.options.chars.middle;
                }
                else if (opts.hasBorder) {
                    result += " ";
                }
            }
        }
        let maxLength = opts.width[colIndex];
        const colSpan = cell.getColSpan();
        if (colSpan > 1) {
            for (let o = 1; o < colSpan; o++) {
                maxLength += opts.width[colIndex + o] + opts.padding[colIndex + o];
                if (opts.hasBorder) {
                    maxLength += opts.padding[colIndex + o] + 1;
                }
            }
        }
        const { current, next } = this.renderCellValue(cell, maxLength);
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
    renderCellValue(cell, maxLength) {
        const length = Math.min(maxLength, strLength(cell.toString()));
        let words = consumeWords(length, cell.toString());
        const breakWord = strLength(words) > length;
        if (breakWord) {
            words = words.slice(0, length);
        }
        const next = cell.toString().slice(words.length + (breakWord ? 0 : 1));
        const fillLength = maxLength - strLength(words);
        const align = cell.getAlign();
        let current;
        if (fillLength === 0) {
            current = words;
        }
        else if (align === "left") {
            current = words + " ".repeat(fillLength);
        }
        else if (align === "center") {
            current = " ".repeat(Math.floor(fillLength / 2)) + words +
                " ".repeat(Math.ceil(fillLength / 2));
        }
        else if (align === "right") {
            current = " ".repeat(fillLength) + words;
        }
        else {
            throw new Error("Unknown direction: " + align);
        }
        return {
            current,
            next: cell.clone(next),
        };
    }
    renderBorderRow(prevRow, nextRow, rowSpan, opts) {
        let result = "";
        let colSpan = 1;
        for (let colIndex = 0; colIndex < opts.columns; colIndex++) {
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
    renderBorderCell(colIndex, prevRow, nextRow, rowSpan, opts) {
        const a1 = prevRow?.[colIndex - 1];
        const a2 = nextRow?.[colIndex - 1];
        const b1 = prevRow?.[colIndex];
        const b2 = nextRow?.[colIndex];
        const a1Border = !!a1?.getBorder();
        const a2Border = !!a2?.getBorder();
        const b1Border = !!b1?.getBorder();
        const b2Border = !!b2?.getBorder();
        const hasColSpan = (cell) => (cell?.getColSpan() ?? 1) > 1;
        const hasRowSpan = (cell) => (cell?.getRowSpan() ?? 1) > 1;
        let result = "";
        if (colIndex === 0) {
            if (rowSpan[colIndex] > 1) {
                if (b1Border) {
                    result += this.options.chars.left;
                }
                else {
                    result += " ";
                }
            }
            else if (b1Border && b2Border) {
                result += this.options.chars.leftMid;
            }
            else if (b1Border) {
                result += this.options.chars.bottomLeft;
            }
            else if (b2Border) {
                result += this.options.chars.topLeft;
            }
            else {
                result += " ";
            }
        }
        else if (colIndex < opts.columns) {
            if ((a1Border && b2Border) || (b1Border && a2Border)) {
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
                }
                else if (hasAllColSpan && hasAllBorder && a1 === b1 && a2 === b2) {
                    result += this.options.chars.mid;
                }
                else if (a1ColSpan && b1ColSpan && a1 === b1) {
                    result += this.options.chars.topMid;
                }
                else if (a2ColSpan && b2ColSpan && a2 === b2) {
                    result += this.options.chars.bottomMid;
                }
                else if (a1RowSpan && a2RowSpan && a1 === a2) {
                    result += this.options.chars.leftMid;
                }
                else if (b1RowSpan && b2RowSpan && b1 === b2) {
                    result += this.options.chars.rightMid;
                }
                else {
                    result += this.options.chars.midMid;
                }
            }
            else if (a1Border && b1Border) {
                if (hasColSpan(a1) && hasColSpan(b1) && a1 === b1) {
                    result += this.options.chars.bottom;
                }
                else {
                    result += this.options.chars.bottomMid;
                }
            }
            else if (b1Border && b2Border) {
                if (rowSpan[colIndex] > 1) {
                    result += this.options.chars.left;
                }
                else {
                    result += this.options.chars.leftMid;
                }
            }
            else if (b2Border && a2Border) {
                if (hasColSpan(a2) && hasColSpan(b2) && a2 === b2) {
                    result += this.options.chars.top;
                }
                else {
                    result += this.options.chars.topMid;
                }
            }
            else if (a1Border && a2Border) {
                if (hasRowSpan(a1) && a1 === a2) {
                    result += this.options.chars.right;
                }
                else {
                    result += this.options.chars.rightMid;
                }
            }
            else if (a1Border) {
                result += this.options.chars.bottomRight;
            }
            else if (b1Border) {
                result += this.options.chars.bottomLeft;
            }
            else if (a2Border) {
                result += this.options.chars.topRight;
            }
            else if (b2Border) {
                result += this.options.chars.topLeft;
            }
            else {
                result += " ";
            }
        }
        const length = opts.padding[colIndex] + opts.width[colIndex] +
            opts.padding[colIndex];
        if (rowSpan[colIndex] > 1 && nextRow) {
            result += this.renderCell(colIndex, nextRow, opts, true);
            if (nextRow[colIndex] === nextRow[nextRow.length - 1]) {
                if (b1Border) {
                    result += this.options.chars.right;
                }
                else {
                    result += " ";
                }
                return result;
            }
        }
        else if (b1Border && b2Border) {
            result += this.options.chars.mid.repeat(length);
        }
        else if (b1Border) {
            result += this.options.chars.bottom.repeat(length);
        }
        else if (b2Border) {
            result += this.options.chars.top.repeat(length);
        }
        else {
            result += " ".repeat(length);
        }
        if (colIndex === opts.columns - 1) {
            if (b1Border && b2Border) {
                result += this.options.chars.rightMid;
            }
            else if (b1Border) {
                result += this.options.chars.bottomRight;
            }
            else if (b2Border) {
                result += this.options.chars.topRight;
            }
            else {
                result += " ";
            }
        }
        return result;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibGF5b3V0LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibGF5b3V0LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxJQUFJLEVBQW9CLE1BQU0sV0FBVyxDQUFDO0FBQ25ELE9BQU8sRUFBUSxHQUFHLEVBQUUsTUFBTSxVQUFVLENBQUM7QUFFckMsT0FBTyxFQUFFLFlBQVksRUFBRSxPQUFPLEVBQUUsU0FBUyxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBYzlELE1BQU0sT0FBTyxXQUFXO0lBT1o7SUFDQTtJQUZWLFlBQ1UsS0FBWSxFQUNaLE9BQXVCO1FBRHZCLFVBQUssR0FBTCxLQUFLLENBQU87UUFDWixZQUFPLEdBQVAsT0FBTyxDQUFnQjtJQUM5QixDQUFDO0lBR0csUUFBUTtRQUNiLE1BQU0sSUFBSSxHQUFvQixJQUFJLENBQUMsWUFBWSxFQUFFLENBQUM7UUFDbEQsT0FBTyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLFVBQVUsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDO0lBQ3ZELENBQUM7SUFPUyxZQUFZO1FBQ3BCLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsQ0FBQyxPQUFPLENBQUMsQ0FBQyxHQUFXLEVBQUUsRUFBRTtZQUN0RCxJQUFJLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBMkIsQ0FBQyxLQUFLLFFBQVEsRUFBRTtnQkFDdkUsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBMkIsQ0FBQyxHQUFHLEVBQUUsQ0FBQzthQUN0RDtRQUNILENBQUMsQ0FBQyxDQUFDO1FBRUgsTUFBTSxhQUFhLEdBQVksSUFBSSxDQUFDLEtBQUssQ0FBQyxTQUFTLEVBQUU7WUFDbkQsSUFBSSxDQUFDLEtBQUssQ0FBQyxhQUFhLEVBQUUsQ0FBQztRQUM3QixNQUFNLGVBQWUs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QUMsUUFBUSxDQUFDO2FBQ3ZDO2lCQUFNLElBQUksUUFBUSxFQUFFO2dCQUNuQixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsV0FBVyxDQUFDO2FBQzFDO2lCQUFNLElBQUksUUFBUSxFQUFFO2dCQUNuQixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsUUFBUSxDQUFDO2FBQ3ZDO2lCQUFNO2dCQUNMLE1BQU0sSUFBSSxHQUFHLENBQUM7YUFDZjtTQUNGO1FBRUQsT0FBTyxNQUFNLENBQUM7SUFDaEIsQ0FBQztDQUNGIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ2VsbCwgRGlyZWN0aW9uLCBJQ2VsbCB9IGZyb20gXCIuL2NlbGwudHNcIjtcbmltcG9ydCB7IElSb3csIFJvdyB9IGZyb20gXCIuL3Jvdy50c1wiO1xuaW1wb3J0IHR5cGUgeyBJQm9yZGVyT3B0aW9ucywgSVRhYmxlU2V0dGluZ3MsIFRhYmxlIH0gZnJvbSBcIi4vdGFibGUudHNcIjtcbmltcG9ydCB7IGNvbnN1bWVXb3JkcywgbG9uZ2VzdCwgc3RyTGVuZ3RoIH0gZnJvbSBcIi4vdXRpbHMudHNcIjtcblxuLyoqIExheW91dCByZW5kZXIgc2V0dGluZ3MuICovXG5pbnRlcmZhY2UgSVJlbmRlclNldHRpbmdzIHtcbiAgcGFkZGluZzogbnVtYmVyW107XG4gIHdpZHRoOiBudW1iZXJbXTtcbiAgY29sdW1uczogbnVtYmVyO1xuICBoYXNCb3JkZXI6IGJvb2xlYW47XG4gIGhhc0hlYWRlckJvcmRlcjogYm9vbGVhbjtcbiAgaGFzQm9keUJvcmRlcjogYm9vbGVhbjtcbiAgcm93czogUm93PENlbGw+W107XG59XG5cbi8qKiBUYWJsZSBsYXlvdXQgcmVuZGVyZXIuICovXG5leHBvcnQgY2xhc3MgVGFibGVMYXlvdXQge1xuICAvKipcbiAgICogVGFibGUgbGF5b3V0IGNvbnN0cnVjdG9yLlxuICAgKiBAcGFyYW0gdGFibGUgICBUYWJsZSBpbnN0YW5jZS5cbiAgICogQHBhcmFtIG9wdGlvbnMgUmVuZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgY29uc3RydWN0b3IoXG4gICAgcHJpdmF0ZSB0YWJsZTogVGFibGUsXG4gICAgcHJpdmF0ZSBvcHRpb25zOiBJVGFibGVTZXR0aW5ncyxcbiAgKSB7fVxuXG4gIC8qKiBHZW5lcmF0ZSB0YWJsZSBzdHJpbmcuICovXG4gIHB1YmxpYyB0b1N0cmluZygpOiBzdHJpbmcge1xuICAgIGNvbnN0IG9wdHM6IElSZW5kZXJTZXR0aW5ncyA9IHRoaXMuY3JlYXRlTGF5b3V0KCk7XG4gICAgcmV0dXJuIG9wdHMucm93cy5sZW5ndGggPyB0aGlzLnJlbmRlclJvd3Mob3B0cykgOiBcIlwiO1xuICB9XG5cbiAgLyoqXG4gICAqIEdlbmVyYXRlcyB0YWJsZSBsYXlvdXQgaW5jbHVkaW5nIHJvdyBhbmQgY29sIHNwYW4sIGNvbnZlcnRzIGFsbCBub25lXG4gICAqIENlbGwvUm93IHZhbHVlcyB0byBDZWxscyBhbmQgUm93cyBhbmQgcmV0dXJucyB0aGUgbGF5b3V0IHJlbmRlcmluZ1xuICAgKiBzZXR0aW5ncy5cbiAgICovXG4gIHByb3RlY3RlZCBjcmVhdGVMYXlvdXQoKTogSVJlbmRlclNldHRpbmdzIHtcbiAgICBPYmplY3Qua2V5cyh0aGlzLm9wdGlvbnMuY2hhcnMpLmZvckVhY2goKGtleTogc3RyaW5nKSA9PiB7XG4gICAgICBpZiAodHlwZW9mIHRoaXMub3B0aW9ucy5jaGFyc1trZXkgYXMga2V5b2YgSUJvcmRlck9wdGlvbnNdICE9PSBcInN0cmluZ1wiKSB7XG4gICAgICAgIHRoaXMub3B0aW9ucy5jaGFyc1trZXkgYXMga2V5b2YgSUJvcmRlck9wdGlvbnNdID0gXCJcIjtcbiAgICAgIH1cbiAgICB9KTtcblxuICAgIGNvbnN0IGhhc0JvZHlCb3JkZXI6IGJvb2xlYW4gPSB0aGlzLnRhYmxlLmdldEJvcmRlcigpIHx8XG4gICAgICB0aGlzLnRhYmxlLmhhc0JvZHlCb3JkZXIoKTtcbiAgICBjb25zdCBoYXNIZWFkZXJCb3JkZXI6IGJvb2xlYW4gPSB0aGlzLnRhYmxlLmhhc0hlYWRlckJvcmRlcigpO1xuICAgIGNvbnN0IGhhc0JvcmRlcjogYm9vbGVhbiA9IGhhc0hlYWRlckJvcmRlciB8fCBoYXNCb2R5Qm9yZGVyO1xuXG4gICAgY29uc3Qgcm93cyA9IHRoaXMuI2dldFJvd3MoKTtcblxuICAgIGNvbnN0IGNvbHVtbnM6IG51bWJlciA9IE1hdGgubWF4KC4uLnJvd3MubWFwKChyb3cpID0+IHJvdy5sZW5ndGgpKTtcbiAgICBmb3IgKGNvbnN0IHJvdyBvZiByb3dzKSB7XG4gICAgICBjb25zdCBsZW5ndGg6IG51bWJlciA9IHJvdy5sZW5ndGg7XG4gICAgICBpZiAobGVuZ3RoIDwgY29sdW1ucykge1xuICAgICAgICBjb25zdCBkaWZmID0gY29sdW1ucyAtIGxlbmd0aDtcbiAgICAgICAgZm9yIChsZXQgaSA9IDA7IGkgPCBkaWZmOyBpKyspIHtcbiAgICAgICAgICByb3cucHVzaCh0aGlzLmNyZWF0ZUNlbGwobnVsbCwgcm93KSk7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG5cbiAgICBjb25zdCBwYWRkaW5nOiBudW1iZXJbXSA9IFtdO1xuICAgIGNvbnN0IHdpZHRoOiBudW1iZXJbXSA9IFtdO1xuICAgIGZvciAobGV0IGNvbEluZGV4ID0gMDsgY29sSW5kZXggPCBjb2x1bW5zOyBjb2xJbmRleCsrKSB7XG4gICAgICBjb25zdCBtaW5Db2xXaWR0aDogbnVtYmVyID0gQXJyYXkuaXNBcnJheSh0aGlzLm9wdGlvbnMubWluQ29sV2lkdGgpXG4gICAgICAgID8gdGhpcy5vcHRpb25zLm1pbkNvbFdpZHRoW2NvbEluZGV4XVxuICAgICAgICA6IHRoaXMub3B0aW9ucy5taW5Db2xXaWR0aDtcbiAgICAgIGNvbnN0IG1heENvbFdpZHRoOiBudW1iZXIgPSBBcnJheS5pc0FycmF5KHRoaXMub3B0aW9ucy5tYXhDb2xXaWR0aClcbiAgICAgICAgPyB0aGlzLm9wdGlvbnMubWF4Q29sV2lkdGhbY29sSW5kZXhdXG4gICAgICAgIDogdGhpcy5vcHRpb25zLm1heENvbFdpZHRoO1xuICAgICAgY29uc3QgY29sV2lkdGg6IG51bWJlciA9IGxvbmdlc3QoY29sSW5kZXgsIHJvd3MsIG1heENvbFdpZHRoKTtcbiAgICAgIHdpZHRoW2NvbEluZGV4XSA9IE1hdGgubWluKG1heENvbFdpZHRoLCBNYXRoLm1heChtaW5Db2xXaWR0aCwgY29sV2lkdGgpKTtcbiAgICAgIHBhZGRpbmdbY29sSW5kZXhdID0gQXJyYXkuaXNBcnJheSh0aGlzLm9wdGlvbnMucGFkZGluZylcbiAgICAgICAgPyB0aGlzLm9wdGlvbnMucGFkZGluZ1tjb2xJbmRleF1cbiAgICAgICAgOiB0aGlzLm9wdGlvbnMucGFkZGluZztcbiAgICB9XG5cbiAgICByZXR1cm4ge1xuICAgICAgcGFkZGluZyxcbiAgICAgIHdpZHRoLFxuICAgICAgcm93cyxcbiAgICAgIGNvbHVtbnMsXG4gICAgICBoYXNCb3JkZXIsXG4gICAgICBoYXNCb2R5Qm9yZGVyLFxuICAgICAgaGFzSGVhZGVyQm9yZGVyLFxuICAgIH07XG4gIH1cblxuICAjZ2V0Um93cygpIHtcbiAgICBjb25zdCBoZWFkZXI6IFJvdyB8IHVuZGVmaW5lZCA9IHRoaXMudGFibGUuZ2V0SGVhZGVyKCk7XG4gICAgY29uc3Qgcm93cyA9IGhlYWRlciA/IFtoZWFkZXIsIC4uLnRoaXMudGFibGVdIDogdGhpcy50YWJsZS5zbGljZSgpO1xuICAgIGNvbnN0IGhhc1NwYW4gPSByb3dzLnNvbWUoKHJvdykgPT5cbiAgICAgIHJvdy5zb21lKChjZWxsKSA9PlxuICAgICAgICBjZWxsIGluc3RhbmNlb2YgQ2VsbCAmJiAoY2VsbC5nZXRDb2xTcGFuKCkgPiAxIHx8IGNlbGwuZ2V0Um93U3BhbigpID4gMSlcbiAgICAgIClcbiAgICApO1xuXG4gICAgaWYgKGhhc1NwYW4pIHtcbiAgICAgIHJldHVybiB0aGlzLnNwYW5Sb3dzKHJvd3MpO1xuICAgIH1cblxuICAgIHJldHVybiByb3dzLm1hcCgocm93KSA9PiB7XG4gICAgICBjb25zdCBuZXdSb3cgPSB0aGlzLmNyZWF0ZVJvdyhyb3cpO1xuICAgICAgcmV0dXJuIG5ld1Jvdy5tYXAoKGNlbGwpID0+IHRoaXMuY3JlYXRlQ2VsbChjZWxsLCBuZXdSb3cpKTtcbiAgICB9KSBhcyBBcnJheTxSb3c8Q2VsbD4+O1xuICB9XG5cbiAgLyoqXG4gICAqIEZpbGxzIHJvd3MgYW5kIGNvbHMgYnkgc3BlY2lmaWVkIHJvdy9jb2wgc3BhbiB3aXRoIGEgcmVmZXJlbmNlIG9mIHRoZVxuICAgKiBvcmlnaW5hbCBjZWxsLlxuICAgKlxuICAgKiBAcGFyYW0gX3Jvd3MgICAgIEFsbCB0YWJsZSByb3dzLlxuICAgKiBAcGFyYW0gcm93SW5kZXggIEN1cnJlbnQgcm93IGluZGV4LlxuICAgKiBAcGFyYW0gY29sSW5kZXggIEN1cnJlbnQgY29sIGluZGV4LlxuICAgKiBAcGFyYW0gcm93U3BhbiAgIEN1cnJlbnQgcm93IHNwYW4uXG4gICAqIEBwYXJhbSBjb2xTcGFuICAgQ3VycmVudCBjb2wgc3Bhbi5cbiAgICovXG4gIHByb3RlY3RlZCBzcGFuUm93cyhcbiAgICBfcm93czogSVJvd1tdLFxuICAgIHJvd0luZGV4ID0gMCxcbiAgICBjb2xJbmRleCA9IDAsXG4gICAgcm93U3BhbjogbnVtYmVyW10gPSBbXSxcbiAgICBjb2xTcGFuID0gMSxcbiAgKTogUm93PENlbGw+W10ge1xuICAgIGNvbnN0IHJvd3M6IFJvdzxDZWxsPltdID0gX3Jvd3MgYXMgUm93PENlbGw+W107XG5cbiAgICBpZiAocm93SW5kZXggPj0gcm93cy5sZW5ndGggJiYgcm93U3Bhbi5ldmVyeSgoc3BhbikgPT4gc3BhbiA9PT0gMSkpIHtcbiAgICAgIHJldHVybiByb3dzO1xuICAgIH0gZWxzZSBpZiAoXG4gICAgICByb3dzW3Jvd0luZGV4XSAmJiBjb2xJbmRleCA+PSByb3dzW3Jvd0luZGV4XS5sZW5ndGggJiZcbiAgICAgIGNvbEluZGV4ID49IHJvd1NwYW4ubGVuZ3RoICYmIGNvbFNwYW4gPT09IDFcbiAgICApIHtcbiAgICAgIHJldHVybiB0aGlzLnNwYW5Sb3dzKHJvd3MsICsrcm93SW5kZXgsIDAsIHJvd1NwYW4sIDEpO1xuICAgIH1cblxuICAgIGlmIChjb2xTcGFuID4gMSkge1xuICAgICAgY29sU3Bhbi0tO1xuICAgICAgcm93U3Bhbltjb2xJbmRleF0gPSByb3dTcGFuW2NvbEluZGV4IC0gMV07XG4gICAgICByb3dzW3Jvd0luZGV4XS5zcGxpY2UoY29sSW5kZXggLSAxLCAwLCByb3dzW3Jvd0luZGV4XVtjb2xJbmRleCAtIDFdKTtcbiAgICAgIHJldHVybiB0aGlzLnNwYW5Sb3dzKHJvd3MsIHJvd0luZGV4LCArK2NvbEluZGV4LCByb3dTcGFuLCBjb2xTcGFuKTtcbiAgICB9XG5cbiAgICBpZiAoY29sSW5kZXggPT09IDApIHtcbiAgICAgIHJvd3Nbcm93SW5kZXhdID0gdGhpcy5jcmVhdGVSb3cocm93c1tyb3dJbmRleF0gfHwgW10pO1xuICAgIH1cblxuICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgIHJvd1NwYW5bY29sSW5kZXhdLS07XG4gICAgICByb3dzW3Jvd0luZGV4XS5zcGxpY2UoY29sSW5kZXgsIDAsIHJvd3Nbcm93SW5kZXggLSAxXVtjb2xJbmRleF0pO1xuICAgICAgcmV0dXJuIHRoaXMuc3BhblJvd3Mocm93cywgcm93SW5kZXgsICsrY29sSW5kZXgsIHJvd1NwYW4sIGNvbFNwYW4pO1xuICAgIH1cblxuICAgIHJvd3Nbcm93SW5kZXhdW2NvbEluZGV4XSA9IHRoaXMuY3JlYXRlQ2VsbChcbiAgICAgIHJvd3Nbcm93SW5kZXhdW2NvbEluZGV4XSB8fCBudWxsLFxuICAgICAgcm93c1tyb3dJbmRleF0sXG4gICAgKTtcblxuICAgIGNvbFNwYW4gPSByb3dzW3Jvd0luZGV4XVtjb2xJbmRleF0uZ2V0Q29sU3BhbigpO1xuICAgIHJvd1NwYW5bY29sSW5kZXhdID0gcm93c1tyb3dJbmRleF1bY29sSW5kZXhdLmdldFJvd1NwYW4oKTtcblxuICAgIHJldHVybiB0aGlzLnNwYW5Sb3dzKHJvd3MsIHJvd0luZGV4LCArK2NvbEluZGV4LCByb3dTcGFuLCBjb2xTcGFuKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDcmVhdGUgYSBuZXcgcm93IGZyb20gZXhpc3Rpbmcgcm93IG9yIGNlbGwgYXJyYXkuXG4gICAqIEBwYXJhbSByb3cgT3JpZ2luYWwgcm93LlxuICAgKi9cbiAgcHJvdGVjdGVkIGNyZWF0ZVJvdyhyb3c6IElSb3cpOiBSb3c8Q2VsbD4ge1xuICAgIHJldHVybiBSb3cuZnJvbShyb3cpXG4gICAgICAuYm9yZGVyKHRoaXMudGFibGUuZ2V0Qm9yZGVyKCksIGZhbHNlKVxuICAgICAgLmFsaWduKHRoaXMudGFibGUuZ2V0QWxpZ24oKSwgZmFsc2UpIGFzIFJvdzxDZWxsPjtcbiAgfVxuXG4gIC8qKlxuICAgKiBDcmVhdGUgYSBuZXcgY2VsbCBmcm9tIGV4aXN0aW5nIGNlbGwgb3IgY2VsbCB2YWx1ZS5cbiAgICogQHBhcmFtIGNlbGwgIE9yaWdpbmFsIGNlbGwuXG4gICAqIEBwYXJhbSByb3cgICBQYXJlbnQgcm93LlxuICAgKi9cbiAgcHJvdGVjdGVkIGNyZWF0ZUNlbGwoY2VsbDogSUNlbGwgfCBudWxsLCByb3c6IFJvdyk6IENlbGwge1xuICAgIHJldHVybiBDZWxsLmZyb20oY2VsbCA/PyBcIlwiKVxuICAgICAgLmJvcmRlcihyb3cuZ2V0Qm9yZGVyKCksIGZhbHNlKVxuICAgICAgLmFsaWduKHJvdy5nZXRBbGlnbigpLCBmYWxzZSk7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIHRhYmxlIGxheW91dC5cbiAgICogQHBhcmFtIG9wdHMgUmVuZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyUm93cyhvcHRzOiBJUmVuZGVyU2V0dGluZ3MpOiBzdHJpbmcge1xuICAgIGxldCByZXN1bHQgPSBcIlwiO1xuICAgIGNvbnN0IHJvd1NwYW46IG51bWJlcltdID0gbmV3IEFycmF5KG9wdHMuY29sdW1ucykuZmlsbCgxKTtcblxuICAgIGZvciAobGV0IHJvd0luZGV4ID0gMDsgcm93SW5kZXggPCBvcHRzLnJvd3MubGVuZ3RoOyByb3dJbmRleCsrKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJSb3cocm93U3Bhbiwgcm93SW5kZXgsIG9wdHMpO1xuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQuc2xpY2UoMCwgLTEpO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciByb3cuXG4gICAqIEBwYXJhbSByb3dTcGFuICAgICBDdXJyZW50IHJvdyBzcGFuLlxuICAgKiBAcGFyYW0gcm93SW5kZXggICAgQ3VycmVudCByb3cgaW5kZXguXG4gICAqIEBwYXJhbSBvcHRzICAgICAgICBSZW5kZXIgb3B0aW9ucy5cbiAgICogQHBhcmFtIGlzTXVsdGlsaW5lIElzIG11bHRpbGluZSByb3cuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyUm93KFxuICAgIHJvd1NwYW46IG51bWJlcltdLFxuICAgIHJvd0luZGV4OiBudW1iZXIsXG4gICAgb3B0czogSVJlbmRlclNldHRpbmdzLFxuICAgIGlzTXVsdGlsaW5lPzogYm9vbGVhbixcbiAgKTogc3RyaW5nIHtcbiAgICBjb25zdCByb3c6IFJvdzxDZWxsPiA9IG9wdHMucm93c1tyb3dJbmRleF07XG4gICAgY29uc3QgcHJldlJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkID0gb3B0cy5yb3dzW3Jvd0luZGV4IC0gMV07XG4gICAgY29uc3QgbmV4dFJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkID0gb3B0cy5yb3dzW3Jvd0luZGV4ICsgMV07XG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG5cbiAgICBsZXQgY29sU3BhbiA9IDE7XG5cbiAgICAvLyBib3JkZXIgdG9wIHJvd1xuICAgIGlmICghaXNNdWx0aWxpbmUgJiYgcm93SW5kZXggPT09IDAgJiYgcm93Lmhhc0JvcmRlcigpKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJSb3codW5kZWZpbmVkLCByb3csIHJvd1NwYW4sIG9wdHMpO1xuICAgIH1cblxuICAgIGxldCBpc011bHRpbGluZVJvdyA9IGZhbHNlO1xuXG4gICAgcmVzdWx0ICs9IFwiIFwiLnJlcGVhdCh0aGlzLm9wdGlvbnMuaW5kZW50IHx8IDApO1xuXG4gICAgZm9yIChsZXQgY29sSW5kZXggPSAwOyBjb2xJbmRleCA8IG9wdHMuY29sdW1uczsgY29sSW5kZXgrKykge1xuICAgICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICAgIGNvbFNwYW4tLTtcbiAgICAgICAgcm93U3Bhbltjb2xJbmRleF0gPSByb3dTcGFuW2NvbEluZGV4IC0gMV07XG4gICAgICAgIGNvbnRpbnVlO1xuICAgICAgfVxuXG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJDZWxsKGNvbEluZGV4LCByb3csIG9wdHMpO1xuXG4gICAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPiAxKSB7XG4gICAgICAgIGlmICghaXNNdWx0aWxpbmUpIHtcbiAgICAgICAgICByb3dTcGFuW2NvbEluZGV4XS0tO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKCFwcmV2Um93IHx8IHByZXZSb3dbY29sSW5kZXhdICE9PSByb3dbY29sSW5kZXhdKSB7XG4gICAgICAgIHJvd1NwYW5bY29sSW5kZXhdID0gcm93W2NvbEluZGV4XS5nZXRSb3dTcGFuKCk7XG4gICAgICB9XG5cbiAgICAgIGNvbFNwYW4gPSByb3dbY29sSW5kZXhdLmdldENvbFNwYW4oKTtcblxuICAgICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID09PSAxICYmIHJvd1tjb2xJbmRleF0ubGVuZ3RoKSB7XG4gICAgICAgIGlzTXVsdGlsaW5lUm93ID0gdHJ1ZTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBpZiAob3B0cy5jb2x1bW5zID4gMCkge1xuICAgICAgaWYgKHJvd1tvcHRzLmNvbHVtbnMgLSAxXS5nZXRCb3JkZXIoKSkge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0O1xuICAgICAgfSBlbHNlIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICB9XG4gICAgfVxuXG4gICAgcmVzdWx0ICs9IFwiXFxuXCI7XG5cbiAgICBpZiAoaXNNdWx0aWxpbmVSb3cpIHsgLy8gc2tpcCBib3JkZXJcbiAgICAgIHJldHVybiByZXN1bHQgKyB0aGlzLnJlbmRlclJvdyhyb3dTcGFuLCByb3dJbmRleCwgb3B0cywgaXNNdWx0aWxpbmVSb3cpO1xuICAgIH1cblxuICAgIC8vIGJvcmRlciBtaWQgcm93XG4gICAgaWYgKFxuICAgICAgKHJvd0luZGV4ID09PSAwICYmIG9wdHMuaGFzSGVhZGVyQm9yZGVyKSB8fFxuICAgICAgKHJvd0luZGV4IDwgb3B0cy5yb3dzLmxlbmd0aCAtIDEgJiYgb3B0cy5oYXNCb2R5Qm9yZGVyKVxuICAgICkge1xuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyQm9yZGVyUm93KHJvdywgbmV4dFJvdywgcm93U3Bhbiwgb3B0cyk7XG4gICAgfVxuXG4gICAgLy8gYm9yZGVyIGJvdHRvbSByb3dcbiAgICBpZiAocm93SW5kZXggPT09IG9wdHMucm93cy5sZW5ndGggLSAxICYmIHJvdy5oYXNCb3JkZXIoKSkge1xuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyQm9yZGVyUm93KHJvdywgdW5kZWZpbmVkLCByb3dTcGFuLCBvcHRzKTtcbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0O1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciBjZWxsLlxuICAgKiBAcGFyYW0gY29sSW5kZXggIEN1cnJlbnQgY29sIGluZGV4LlxuICAgKiBAcGFyYW0gcm93ICAgICAgIEN1cnJlbnQgcm93LlxuICAgKiBAcGFyYW0gb3B0cyAgICAgIFJlbmRlciBvcHRpb25zLlxuICAgKiBAcGFyYW0gbm9Cb3JkZXIgIERpc2FibGUgYm9yZGVyLlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlckNlbGwoXG4gICAgY29sSW5kZXg6IG51bWJlcixcbiAgICByb3c6IFJvdzxDZWxsPixcbiAgICBvcHRzOiBJUmVuZGVyU2V0dGluZ3MsXG4gICAgbm9Cb3JkZXI/OiBib29sZWFuLFxuICApOiBzdHJpbmcge1xuICAgIGxldCByZXN1bHQgPSBcIlwiO1xuICAgIGNvbnN0IHByZXZDZWxsOiBDZWxsIHwgdW5kZWZpbmVkID0gcm93W2NvbEluZGV4IC0gMV07XG5cbiAgICBjb25zdCBjZWxsOiBDZWxsID0gcm93W2NvbEluZGV4XTtcblxuICAgIGlmICghbm9Cb3JkZXIpIHtcbiAgICAgIGlmIChjb2xJbmRleCA9PT0gMCkge1xuICAgICAgICBpZiAoY2VsbC5nZXRCb3JkZXIoKSkge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdDtcbiAgICAgICAgfSBlbHNlIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIHtcbiAgICAgICAgaWYgKGNlbGwuZ2V0Qm9yZGVyKCkgfHwgcHJldkNlbGw/LmdldEJvcmRlcigpKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWRkbGU7XG4gICAgICAgIH0gZWxzZSBpZiAob3B0cy5oYXNCb3JkZXIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG5cbiAgICBsZXQgbWF4TGVuZ3RoOiBudW1iZXIgPSBvcHRzLndpZHRoW2NvbEluZGV4XTtcblxuICAgIGNvbnN0IGNvbFNwYW46IG51bWJlciA9IGNlbGwuZ2V0Q29sU3BhbigpO1xuICAgIGlmIChjb2xTcGFuID4gMSkge1xuICAgICAgZm9yIChsZXQgbyA9IDE7IG8gPCBjb2xTcGFuOyBvKyspIHtcbiAgICAgICAgLy8gYWRkIHBhZGRpbmcgYW5kIHdpdGggb2YgbmV4dCBjZWxsXG4gICAgICAgIG1heExlbmd0aCArPSBvcHRzLndpZHRoW2NvbEluZGV4ICsgb10gKyBvcHRzLnBhZGRpbmdbY29sSW5kZXggKyBvXTtcbiAgICAgICAgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICAgICAgLy8gYWRkIHBhZGRpbmcgYWdhaW4gYW5kIGJvcmRlciB3aXRoXG4gICAgICAgICAgbWF4TGVuZ3RoICs9IG9wdHMucGFkZGluZ1tjb2xJbmRleCArIG9dICsgMTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IHsgY3VycmVudCwgbmV4dCB9ID0gdGhpcy5yZW5kZXJDZWxsVmFsdWUoY2VsbCwgbWF4TGVuZ3RoKTtcblxuICAgIHJvd1tjb2xJbmRleF0uc2V0VmFsdWUobmV4dCk7XG5cbiAgICBpZiAob3B0cy5oYXNCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSBcIiBcIi5yZXBlYXQob3B0cy5wYWRkaW5nW2NvbEluZGV4XSk7XG4gICAgfVxuXG4gICAgcmVzdWx0ICs9IGN1cnJlbnQ7XG5cbiAgICBpZiAob3B0cy5oYXNCb3JkZXIgfHwgY29sSW5kZXggPCBvcHRzLmNvbHVtbnMgLSAxKSB7XG4gICAgICByZXN1bHQgKz0gXCIgXCIucmVwZWF0KG9wdHMucGFkZGluZ1tjb2xJbmRleF0pO1xuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQ7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIHNwZWNpZmllZCBsZW5ndGggb2YgY2VsbC4gUmV0dXJucyB0aGUgcmVuZGVyZWQgdmFsdWUgYW5kIGEgbmV3IGNlbGxcbiAgICogd2l0aCB0aGUgcmVzdCB2YWx1ZS5cbiAgICogQHBhcmFtIGNlbGwgICAgICBDZWxsIHRvIHJlbmRlci5cbiAgICogQHBhcmFtIG1heExlbmd0aCBNYXggbGVuZ3RoIG9mIGNvbnRlbnQgdG8gcmVuZGVyLlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlckNlbGxWYWx1ZShcbiAgICBjZWxsOiBDZWxsLFxuICAgIG1heExlbmd0aDogbnVtYmVyLFxuICApOiB7IGN1cnJlbnQ6IHN0cmluZzsgbmV4dDogQ2VsbCB9IHtcbiAgICBjb25zdCBsZW5ndGg6IG51bWJlciA9IE1hdGgubWluKFxuICAgICAgbWF4TGVuZ3RoLFxuICAgICAgc3RyTGVuZ3RoKGNlbGwudG9TdHJpbmcoKSksXG4gICAgKTtcbiAgICBsZXQgd29yZHM6IHN0cmluZyA9IGNvbnN1bWVXb3JkcyhsZW5ndGgsIGNlbGwudG9TdHJpbmcoKSk7XG5cbiAgICAvLyBicmVhayB3b3JkIGlmIHdvcmQgaXMgbG9uZ2VyIHRoYW4gbWF4IGxlbmd0aFxuICAgIGNvbnN0IGJyZWFrV29yZCA9IHN0ckxlbmd0aCh3b3JkcykgPiBsZW5ndGg7XG4gICAgaWYgKGJyZWFrV29yZCkge1xuICAgICAgd29yZHMgPSB3b3Jkcy5zbGljZSgwLCBsZW5ndGgpO1xuICAgIH1cblxuICAgIC8vIGdldCBuZXh0IGNvbnRlbnQgYW5kIHJlbW92ZSBsZWFkaW5nIHNwYWNlIGlmIGJyZWFrV29yZCBpcyBub3QgdHJ1ZVxuICAgIGNvbnN0IG5leHQgPSBjZWxsLnRvU3RyaW5nKCkuc2xpY2Uod29yZHMubGVuZ3RoICsgKGJyZWFrV29yZCA/IDAgOiAxKSk7XG4gICAgY29uc3QgZmlsbExlbmd0aCA9IG1heExlbmd0aCAtIHN0ckxlbmd0aCh3b3Jkcyk7XG5cbiAgICAvLyBBbGlnbiBjb250ZW50XG4gICAgY29uc3QgYWxpZ246IERpcmVjdGlvbiA9IGNlbGwuZ2V0QWxpZ24oKTtcbiAgICBsZXQgY3VycmVudDogc3RyaW5nO1xuICAgIGlmIChmaWxsTGVuZ3RoID09PSAwKSB7XG4gICAgICBjdXJyZW50ID0gd29yZHM7XG4gICAgfSBlbHNlIGlmIChhbGlnbiA9PT0gXCJsZWZ0XCIpIHtcbiAgICAgIGN1cnJlbnQgPSB3b3JkcyArIFwiIFwiLnJlcGVhdChmaWxsTGVuZ3RoKTtcbiAgICB9IGVsc2UgaWYgKGFsaWduID09PSBcImNlbnRlclwiKSB7XG4gICAgICBjdXJyZW50ID0gXCIgXCIucmVwZWF0KE1hdGguZmxvb3IoZmlsbExlbmd0aCAvIDIpKSArIHdvcmRzICtcbiAgICAgICAgXCIgXCIucmVwZWF0KE1hdGguY2VpbChmaWxsTGVuZ3RoIC8gMikpO1xuICAgIH0gZWxzZSBpZiAoYWxpZ24gPT09IFwicmlnaHRcIikge1xuICAgICAgY3VycmVudCA9IFwiIFwiLnJlcGVhdChmaWxsTGVuZ3RoKSArIHdvcmRzO1xuICAgIH0gZWxzZSB7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoXCJVbmtub3duIGRpcmVjdGlvbjogXCIgKyBhbGlnbik7XG4gICAgfVxuXG4gICAgcmV0dXJuIHtcbiAgICAgIGN1cnJlbnQsXG4gICAgICBuZXh0OiBjZWxsLmNsb25lKG5leHQpLFxuICAgIH07XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIGJvcmRlciByb3cuXG4gICAqIEBwYXJhbSBwcmV2Um93IFByZXZpb3VzIHJvdy5cbiAgICogQHBhcmFtIG5leHRSb3cgTmV4dCByb3cuXG4gICAqIEBwYXJhbSByb3dTcGFuIEN1cnJlbnQgcm93IHNwYW4uXG4gICAqIEBwYXJhbSBvcHRzICAgIFJlbmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlckJvcmRlclJvdyhcbiAgICBwcmV2Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQsXG4gICAgbmV4dFJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkLFxuICAgIHJvd1NwYW46IG51bWJlcltdLFxuICAgIG9wdHM6IElSZW5kZXJTZXR0aW5ncyxcbiAgKTogc3RyaW5nIHtcbiAgICBsZXQgcmVzdWx0ID0gXCJcIjtcblxuICAgIGxldCBjb2xTcGFuID0gMTtcbiAgICBmb3IgKGxldCBjb2xJbmRleCA9IDA7IGNvbEluZGV4IDwgb3B0cy5jb2x1bW5zOyBjb2xJbmRleCsrKSB7XG4gICAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPiAxKSB7XG4gICAgICAgIGlmICghbmV4dFJvdykge1xuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcImludmFsaWQgbGF5b3V0XCIpO1xuICAgICAgICB9XG4gICAgICAgIGlmIChjb2xTcGFuID4gMSkge1xuICAgICAgICAgIGNvbFNwYW4tLTtcbiAgICAgICAgICBjb250aW51ZTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyQm9yZGVyQ2VsbChcbiAgICAgICAgY29sSW5kZXgsXG4gICAgICAgIHByZXZSb3csXG4gICAgICAgIG5leHRSb3csXG4gICAgICAgIHJvd1NwYW4sXG4gICAgICAgIG9wdHMsXG4gICAgICApO1xuICAgICAgY29sU3BhbiA9IG5leHRSb3c/Lltjb2xJbmRleF0uZ2V0Q29sU3BhbigpID8/IDE7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdC5sZW5ndGggPyBcIiBcIi5yZXBlYXQodGhpcy5vcHRpb25zLmluZGVudCkgKyByZXN1bHQgKyBcIlxcblwiIDogXCJcIjtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgYm9yZGVyIGNlbGwuXG4gICAqIEBwYXJhbSBjb2xJbmRleCAgQ3VycmVudCBpbmRleC5cbiAgICogQHBhcmFtIHByZXZSb3cgICBQcmV2aW91cyByb3cuXG4gICAqIEBwYXJhbSBuZXh0Um93ICAgTmV4dCByb3cuXG4gICAqIEBwYXJhbSByb3dTcGFuICAgQ3VycmVudCByb3cgc3Bhbi5cbiAgICogQHBhcmFtIG9wdHMgICAgICBSZW5kZXIgb3B0aW9ucy5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJCb3JkZXJDZWxsKFxuICAgIGNvbEluZGV4OiBudW1iZXIsXG4gICAgcHJldlJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkLFxuICAgIG5leHRSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCxcbiAgICByb3dTcGFuOiBudW1iZXJbXSxcbiAgICBvcHRzOiBJUmVuZGVyU2V0dGluZ3MsXG4gICk6IHN0cmluZyB7XG4gICAgLy8gYTEgfCBiMVxuICAgIC8vIC0tLS0tLS1cbiAgICAvLyBhMiB8IGIyXG5cbiAgICBjb25zdCBhMTogQ2VsbCB8IHVuZGVmaW5lZCA9IHByZXZSb3c/Lltjb2xJbmRleCAtIDFdO1xuICAgIGNvbnN0IGEyOiBDZWxsIHwgdW5kZWZpbmVkID0gbmV4dFJvdz8uW2NvbEluZGV4IC0gMV07XG4gICAgY29uc3QgYjE6IENlbGwgfCB1bmRlZmluZWQgPSBwcmV2Um93Py5bY29sSW5kZXhdO1xuICAgIGNvbnN0IGIyOiBDZWxsIHwgdW5kZWZpbmVkID0gbmV4dFJvdz8uW2NvbEluZGV4XTtcblxuICAgIGNvbnN0IGExQm9yZGVyID0gISFhMT8uZ2V0Qm9yZGVyKCk7XG4gICAgY29uc3QgYTJCb3JkZXIgPSAhIWEyPy5nZXRCb3JkZXIoKTtcbiAgICBjb25zdCBiMUJvcmRlciA9ICEhYjE/LmdldEJvcmRlcigpO1xuICAgIGNvbnN0IGIyQm9yZGVyID0gISFiMj8uZ2V0Qm9yZGVyKCk7XG5cbiAgICBjb25zdCBoYXNDb2xTcGFuID0gKGNlbGw6IENlbGwgfCB1bmRlZmluZWQpOiBib29sZWFuID0+XG4gICAgICAoY2VsbD8uZ2V0Q29sU3BhbigpID8/IDEpID4gMTtcbiAgICBjb25zdCBoYXNSb3dTcGFuID0gKGNlbGw6IENlbGwgfCB1bmRlZmluZWQpOiBib29sZWFuID0+XG4gICAgICAoY2VsbD8uZ2V0Um93U3BhbigpID8/IDEpID4gMTtcblxuICAgIGxldCByZXN1bHQgPSBcIlwiO1xuXG4gICAgaWYgKGNvbEluZGV4ID09PSAwKSB7XG4gICAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPiAxKSB7XG4gICAgICAgIGlmIChiMUJvcmRlcikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIgJiYgYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0TWlkO1xuICAgICAgfSBlbHNlIGlmIChiMUJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbUxlZnQ7XG4gICAgICB9IGVsc2UgaWYgKGIyQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wTGVmdDtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgIH1cbiAgICB9IGVsc2UgaWYgKGNvbEluZGV4IDwgb3B0cy5jb2x1bW5zKSB7XG4gICAgICBpZiAoKGExQm9yZGVyICYmIGIyQm9yZGVyKSB8fCAoYjFCb3JkZXIgJiYgYTJCb3JkZXIpKSB7XG4gICAgICAgIGNvbnN0IGExQ29sU3BhbjogYm9vbGVhbiA9IGhhc0NvbFNwYW4oYTEpO1xuICAgICAgICBjb25zdCBhMkNvbFNwYW46IGJvb2xlYW4gPSBoYXNDb2xTcGFuKGEyKTtcbiAgICAgICAgY29uc3QgYjFDb2xTcGFuOiBib29sZWFuID0gaGFzQ29sU3BhbihiMSk7XG4gICAgICAgIGNvbnN0IGIyQ29sU3BhbjogYm9vbGVhbiA9IGhhc0NvbFNwYW4oYjIpO1xuXG4gICAgICAgIGNvbnN0IGExUm93U3BhbjogYm9vbGVhbiA9IGhhc1Jvd1NwYW4oYTEpO1xuICAgICAgICBjb25zdCBhMlJvd1NwYW46IGJvb2xlYW4gPSBoYXNSb3dTcGFuKGEyKTtcbiAgICAgICAgY29uc3QgYjFSb3dTcGFuOiBib29sZWFuID0gaGFzUm93U3BhbihiMSk7XG4gICAgICAgIGNvbnN0IGIyUm93U3BhbjogYm9vbGVhbiA9IGhhc1Jvd1NwYW4oYjIpO1xuXG4gICAgICAgIGNvbnN0IGhhc0FsbEJvcmRlciA9IGExQm9yZGVyICYmIGIyQm9yZGVyICYmIGIxQm9yZGVyICYmIGEyQm9yZGVyO1xuICAgICAgICBjb25zdCBoYXNBbGxSb3dTcGFuID0gYTFSb3dTcGFuICYmIGIxUm93U3BhbiAmJiBhMlJvd1NwYW4gJiYgYjJSb3dTcGFuO1xuICAgICAgICBjb25zdCBoYXNBbGxDb2xTcGFuID0gYTFDb2xTcGFuICYmIGIxQ29sU3BhbiAmJiBhMkNvbFNwYW4gJiYgYjJDb2xTcGFuO1xuXG4gICAgICAgIGlmIChoYXNBbGxSb3dTcGFuICYmIGhhc0FsbEJvcmRlcikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkZGxlO1xuICAgICAgICB9IGVsc2UgaWYgKGhhc0FsbENvbFNwYW4gJiYgaGFzQWxsQm9yZGVyICYmIGExID09PSBiMSAmJiBhMiA9PT0gYjIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLm1pZDtcbiAgICAgICAgfSBlbHNlIGlmIChhMUNvbFNwYW4gJiYgYjFDb2xTcGFuICYmIGExID09PSBiMSkge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wTWlkO1xuICAgICAgICB9IGVsc2UgaWYgKGEyQ29sU3BhbiAmJiBiMkNvbFNwYW4gJiYgYTIgPT09IGIyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21NaWQ7XG4gICAgICAgIH0gZWxzZSBpZiAoYTFSb3dTcGFuICYmIGEyUm93U3BhbiAmJiBhMSA9PT0gYTIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnRNaWQ7XG4gICAgICAgIH0gZWxzZSBpZiAoYjFSb3dTcGFuICYmIGIyUm93U3BhbiAmJiBiMSA9PT0gYjIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0TWlkO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkTWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGExQm9yZGVyICYmIGIxQm9yZGVyKSB7XG4gICAgICAgIGlmIChoYXNDb2xTcGFuKGExKSAmJiBoYXNDb2xTcGFuKGIxKSAmJiBhMSA9PT0gYjEpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbU1pZDtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChiMUJvcmRlciAmJiBiMkJvcmRlcikge1xuICAgICAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPiAxKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0O1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdE1pZDtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChiMkJvcmRlciAmJiBhMkJvcmRlcikge1xuICAgICAgICBpZiAoaGFzQ29sU3BhbihhMikgJiYgaGFzQ29sU3BhbihiMikgJiYgYTIgPT09IGIyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3A7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BNaWQ7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYTFCb3JkZXIgJiYgYTJCb3JkZXIpIHtcbiAgICAgICAgaWYgKGhhc1Jvd1NwYW4oYTEpICYmIGExID09PSBhMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodE1pZDtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChhMUJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbVJpZ2h0O1xuICAgICAgfSBlbHNlIGlmIChiMUJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbUxlZnQ7XG4gICAgICB9IGVsc2UgaWYgKGEyQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wUmlnaHQ7XG4gICAgICB9IGVsc2UgaWYgKGIyQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wTGVmdDtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBjb25zdCBsZW5ndGggPSBvcHRzLnBhZGRpbmdbY29sSW5kZXhdICsgb3B0cy53aWR0aFtjb2xJbmRleF0gK1xuICAgICAgb3B0cy5wYWRkaW5nW2NvbEluZGV4XTtcblxuICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEgJiYgbmV4dFJvdykge1xuICAgICAgcmVzdWx0ICs9IHRoaXMucmVuZGVyQ2VsbChcbiAgICAgICAgY29sSW5kZXgsXG4gICAgICAgIG5leHRSb3csXG4gICAgICAgIG9wdHMsXG4gICAgICAgIHRydWUsXG4gICAgICApO1xuICAgICAgaWYgKG5leHRSb3dbY29sSW5kZXhdID09PSBuZXh0Um93W25leHRSb3cubGVuZ3RoIC0gMV0pIHtcbiAgICAgICAgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICAgIH1cbiAgICAgICAgcmV0dXJuIHJlc3VsdDtcbiAgICAgIH1cbiAgICB9IGVsc2UgaWYgKGIxQm9yZGVyICYmIGIyQm9yZGVyKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLm1pZC5yZXBlYXQobGVuZ3RoKTtcbiAgICB9IGVsc2UgaWYgKGIxQm9yZGVyKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbS5yZXBlYXQobGVuZ3RoKTtcbiAgICB9IGVsc2UgaWYgKGIyQm9yZGVyKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcC5yZXBlYXQobGVuZ3RoKTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmVzdWx0ICs9IFwiIFwiLnJlcGVhdChsZW5ndGgpO1xuICAgIH1cblxuICAgIGlmIChjb2xJbmRleCA9PT0gb3B0cy5jb2x1bW5zIC0gMSkge1xuICAgICAgaWYgKGIxQm9yZGVyICYmIGIyQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHRNaWQ7XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tUmlnaHQ7XG4gICAgICB9IGVsc2UgaWYgKGIyQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wUmlnaHQ7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXN1bHQgKz0gXCIgXCI7XG4gICAgICB9XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdDtcbiAgfVxufVxuIl19