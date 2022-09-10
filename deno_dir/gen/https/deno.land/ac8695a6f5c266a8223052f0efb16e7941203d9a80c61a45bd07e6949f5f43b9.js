import { border } from "./border.ts";
import { Cell } from "./cell.ts";
import { TableLayout } from "./layout.ts";
import { Row } from "./row.ts";
/** Table representation. */ export class Table extends Array {
    static _chars = {
        ...border
    };
    options = {
        indent: 0,
        border: false,
        maxColWidth: Infinity,
        minColWidth: 0,
        padding: 1,
        chars: {
            ...Table._chars
        }
    };
    headerRow;
    /**
   * Create a new table. If rows is a table, all rows and options of the table
   * will be copied to the new table.
   * @param rows
   */ static from(rows) {
        const table = new this(...rows);
        if (rows instanceof Table) {
            table.options = {
                ...rows.options
            };
            table.headerRow = rows.headerRow ? Row.from(rows.headerRow) : undefined;
        }
        return table;
    }
    /**
   * Create a new table from an array of json objects. An object represents a
   * row and each property a column.
   * @param rows Array of objects.
   */ static fromJson(rows) {
        return new this().fromJson(rows);
    }
    /**
   * Set global default border characters.
   * @param chars Border options.
   */ static chars(chars) {
        Object.assign(this._chars, chars);
        return this;
    }
    /**
   * Write table or rows to stdout.
   * @param rows Table or rows.
   */ static render(rows) {
        Table.from(rows).render();
    }
    /**
   * Read data from an array of json objects. An object represents a
   * row and each property a column.
   * @param rows Array of objects.
   */ fromJson(rows) {
        this.header(Object.keys(rows[0]));
        this.body(rows.map((row)=>Object.values(row)));
        return this;
    }
    /**
   * Set table header.
   * @param header Header row or cells.
   */ header(header) {
        this.headerRow = header instanceof Row ? header : Row.from(header);
        return this;
    }
    /**
   * Set table body.
   * @param rows Table rows.
   */ body(rows) {
        this.length = 0;
        this.push(...rows);
        return this;
    }
    /** Clone table recursively with header and options. */ clone() {
        const table = new Table(...this.map((row)=>row instanceof Row ? row.clone() : Row.from(row).clone()));
        table.options = {
            ...this.options
        };
        table.headerRow = this.headerRow?.clone();
        return table;
    }
    /** Generate table string. */ toString() {
        return new TableLayout(this, this.options).toString();
    }
    /** Write table to stdout. */ render() {
        console.log(this.toString());
        return this;
    }
    /**
   * Set max col with.
   * @param width     Max col width.
   * @param override  Override existing value.
   */ maxColWidth(width, override = true) {
        if (override || typeof this.options.maxColWidth === "undefined") {
            this.options.maxColWidth = width;
        }
        return this;
    }
    /**
   * Set min col width.
   * @param width     Min col width.
   * @param override  Override existing value.
   */ minColWidth(width, override = true) {
        if (override || typeof this.options.minColWidth === "undefined") {
            this.options.minColWidth = width;
        }
        return this;
    }
    /**
   * Set table indentation.
   * @param width     Indent width.
   * @param override  Override existing value.
   */ indent(width, override = true) {
        if (override || typeof this.options.indent === "undefined") {
            this.options.indent = width;
        }
        return this;
    }
    /**
   * Set cell padding.
   * @param padding   Cell padding.
   * @param override  Override existing value.
   */ padding(padding, override = true) {
        if (override || typeof this.options.padding === "undefined") {
            this.options.padding = padding;
        }
        return this;
    }
    /**
   * Enable/disable cell border.
   * @param enable    Enable/disable cell border.
   * @param override  Override existing value.
   */ border(enable, override = true) {
        if (override || typeof this.options.border === "undefined") {
            this.options.border = enable;
        }
        return this;
    }
    /**
   * Align table content.
   * @param direction Align direction.
   * @param override  Override existing value.
   */ align(direction, override = true) {
        if (override || typeof this.options.align === "undefined") {
            this.options.align = direction;
        }
        return this;
    }
    /**
   * Set border characters.
   * @param chars Border options.
   */ chars(chars) {
        Object.assign(this.options.chars, chars);
        return this;
    }
    /** Get table header. */ getHeader() {
        return this.headerRow;
    }
    /** Get table body. */ getBody() {
        return [
            ...this
        ];
    }
    /** Get mac col widrth. */ getMaxColWidth() {
        return this.options.maxColWidth;
    }
    /** Get min col width. */ getMinColWidth() {
        return this.options.minColWidth;
    }
    /** Get table indentation. */ getIndent() {
        return this.options.indent;
    }
    /** Get cell padding. */ getPadding() {
        return this.options.padding;
    }
    /** Check if table has border. */ getBorder() {
        return this.options.border === true;
    }
    /** Check if header row has border. */ hasHeaderBorder() {
        const hasBorder = this.headerRow?.hasBorder();
        return hasBorder === true || this.getBorder() && hasBorder !== false;
    }
    /** Check if table bordy has border. */ hasBodyBorder() {
        return this.getBorder() || this.some((row)=>row instanceof Row ? row.hasBorder() : row.some((cell)=>cell instanceof Cell ? cell.getBorder : false));
    }
    /** Check if table header or body has border. */ hasBorder() {
        return this.hasHeaderBorder() || this.hasBodyBorder();
    }
    /** Get table alignment. */ getAlign() {
        return this.options.align ?? "left";
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvdGFibGUvdGFibGUudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgYm9yZGVyLCBJQm9yZGVyIH0gZnJvbSBcIi4vYm9yZGVyLnRzXCI7XG5pbXBvcnQgeyBDZWxsLCBEaXJlY3Rpb24gfSBmcm9tIFwiLi9jZWxsLnRzXCI7XG5pbXBvcnQgeyBUYWJsZUxheW91dCB9IGZyb20gXCIuL2xheW91dC50c1wiO1xuaW1wb3J0IHsgSURhdGFSb3csIElSb3csIFJvdyB9IGZyb20gXCIuL3Jvdy50c1wiO1xuXG4vKiogQm9yZGVyIGNoYXJhY3RlcnMgc2V0dGluZ3MuICovXG5leHBvcnQgdHlwZSBJQm9yZGVyT3B0aW9ucyA9IFBhcnRpYWw8SUJvcmRlcj47XG5cbi8qKiBUYWJsZSBvcHRpb25zLiAqL1xuZXhwb3J0IGludGVyZmFjZSBJVGFibGVPcHRpb25zIHtcbiAgaW5kZW50PzogbnVtYmVyO1xuICBib3JkZXI/OiBib29sZWFuO1xuICBhbGlnbj86IERpcmVjdGlvbjtcbiAgbWF4Q29sV2lkdGg/OiBudW1iZXIgfCBudW1iZXJbXTtcbiAgbWluQ29sV2lkdGg/OiBudW1iZXIgfCBudW1iZXJbXTtcbiAgcGFkZGluZz86IG51bWJlciB8IG51bWJlcltdO1xuICBjaGFycz86IElCb3JkZXJPcHRpb25zO1xufVxuXG4vKiogVGFibGUgc2V0dGluZ3MuICovXG5leHBvcnQgaW50ZXJmYWNlIElUYWJsZVNldHRpbmdzIGV4dGVuZHMgUmVxdWlyZWQ8T21pdDxJVGFibGVPcHRpb25zLCBcImFsaWduXCI+PiB7XG4gIGNoYXJzOiBJQm9yZGVyO1xuICBhbGlnbj86IERpcmVjdGlvbjtcbn1cblxuLyoqIFRhYmxlIHR5cGUuICovXG5leHBvcnQgdHlwZSBJVGFibGU8VCBleHRlbmRzIElSb3cgPSBJUm93PiA9IFRbXSB8IFRhYmxlPFQ+O1xuXG4vKiogVGFibGUgcmVwcmVzZW50YXRpb24uICovXG5leHBvcnQgY2xhc3MgVGFibGU8VCBleHRlbmRzIElSb3cgPSBJUm93PiBleHRlbmRzIEFycmF5PFQ+IHtcbiAgcHJvdGVjdGVkIHN0YXRpYyBfY2hhcnM6IElCb3JkZXIgPSB7IC4uLmJvcmRlciB9O1xuICBwcm90ZWN0ZWQgb3B0aW9uczogSVRhYmxlU2V0dGluZ3MgPSB7XG4gICAgaW5kZW50OiAwLFxuICAgIGJvcmRlcjogZmFsc2UsXG4gICAgbWF4Q29sV2lkdGg6IEluZmluaXR5LFxuICAgIG1pbkNvbFdpZHRoOiAwLFxuICAgIHBhZGRpbmc6IDEsXG4gICAgY2hhcnM6IHsgLi4uVGFibGUuX2NoYXJzIH0sXG4gIH07XG4gIHByaXZhdGUgaGVhZGVyUm93PzogUm93O1xuXG4gIC8qKlxuICAgKiBDcmVhdGUgYSBuZXcgdGFibGUuIElmIHJvd3MgaXMgYSB0YWJsZSwgYWxsIHJvd3MgYW5kIG9wdGlvbnMgb2YgdGhlIHRhYmxlXG4gICAqIHdpbGwgYmUgY29waWVkIHRvIHRoZSBuZXcgdGFibGUuXG4gICAqIEBwYXJhbSByb3dzXG4gICAqL1xuICBwdWJsaWMgc3RhdGljIGZyb208VCBleHRlbmRzIElSb3c+KHJvd3M6IElUYWJsZTxUPik6IFRhYmxlPFQ+IHtcbiAgICBjb25zdCB0YWJsZSA9IG5ldyB0aGlzKC4uLnJvd3MpO1xuICAgIGlmIChyb3dzIGluc3RhbmNlb2YgVGFibGUpIHtcbiAgICAgIHRhYmxlLm9wdGlvbnMgPSB7IC4uLnJvd3Mub3B0aW9ucyB9O1xuICAgICAgdGFibGUuaGVhZGVyUm93ID0gcm93cy5oZWFkZXJSb3cgPyBSb3cuZnJvbShyb3dzLmhlYWRlclJvdykgOiB1bmRlZmluZWQ7XG4gICAgfVxuICAgIHJldHVybiB0YWJsZTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDcmVhdGUgYSBuZXcgdGFibGUgZnJvbSBhbiBhcnJheSBvZiBqc29uIG9iamVjdHMuIEFuIG9iamVjdCByZXByZXNlbnRzIGFcbiAgICogcm93IGFuZCBlYWNoIHByb3BlcnR5IGEgY29sdW1uLlxuICAgKiBAcGFyYW0gcm93cyBBcnJheSBvZiBvYmplY3RzLlxuICAgKi9cbiAgcHVibGljIHN0YXRpYyBmcm9tSnNvbihyb3dzOiBJRGF0YVJvd1tdKTogVGFibGUge1xuICAgIHJldHVybiBuZXcgdGhpcygpLmZyb21Kc29uKHJvd3MpO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBnbG9iYWwgZGVmYXVsdCBib3JkZXIgY2hhcmFjdGVycy5cbiAgICogQHBhcmFtIGNoYXJzIEJvcmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHVibGljIHN0YXRpYyBjaGFycyhjaGFyczogSUJvcmRlck9wdGlvbnMpOiB0eXBlb2YgVGFibGUge1xuICAgIE9iamVjdC5hc3NpZ24odGhpcy5fY2hhcnMsIGNoYXJzKTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBXcml0ZSB0YWJsZSBvciByb3dzIHRvIHN0ZG91dC5cbiAgICogQHBhcmFtIHJvd3MgVGFibGUgb3Igcm93cy5cbiAgICovXG4gIHB1YmxpYyBzdGF0aWMgcmVuZGVyPFQgZXh0ZW5kcyBJUm93Pihyb3dzOiBJVGFibGU8VD4pOiB2b2lkIHtcbiAgICBUYWJsZS5mcm9tKHJvd3MpLnJlbmRlcigpO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlYWQgZGF0YSBmcm9tIGFuIGFycmF5IG9mIGpzb24gb2JqZWN0cy4gQW4gb2JqZWN0IHJlcHJlc2VudHMgYVxuICAgKiByb3cgYW5kIGVhY2ggcHJvcGVydHkgYSBjb2x1bW4uXG4gICAqIEBwYXJhbSByb3dzIEFycmF5IG9mIG9iamVjdHMuXG4gICAqL1xuICBwdWJsaWMgZnJvbUpzb24ocm93czogSURhdGFSb3dbXSk6IHRoaXMge1xuICAgIHRoaXMuaGVhZGVyKE9iamVjdC5rZXlzKHJvd3NbMF0pKTtcbiAgICB0aGlzLmJvZHkocm93cy5tYXAoKHJvdykgPT4gT2JqZWN0LnZhbHVlcyhyb3cpIGFzIFQpKTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgdGFibGUgaGVhZGVyLlxuICAgKiBAcGFyYW0gaGVhZGVyIEhlYWRlciByb3cgb3IgY2VsbHMuXG4gICAqL1xuICBwdWJsaWMgaGVhZGVyKGhlYWRlcjogSVJvdyk6IHRoaXMge1xuICAgIHRoaXMuaGVhZGVyUm93ID0gaGVhZGVyIGluc3RhbmNlb2YgUm93ID8gaGVhZGVyIDogUm93LmZyb20oaGVhZGVyKTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXQgdGFibGUgYm9keS5cbiAgICogQHBhcmFtIHJvd3MgVGFibGUgcm93cy5cbiAgICovXG4gIHB1YmxpYyBib2R5KHJvd3M6IFRbXSk6IHRoaXMge1xuICAgIHRoaXMubGVuZ3RoID0gMDtcbiAgICB0aGlzLnB1c2goLi4ucm93cyk7XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKiogQ2xvbmUgdGFibGUgcmVjdXJzaXZlbHkgd2l0aCBoZWFkZXIgYW5kIG9wdGlvbnMuICovXG4gIHB1YmxpYyBjbG9uZSgpOiBUYWJsZSB7XG4gICAgY29uc3QgdGFibGUgPSBuZXcgVGFibGUoXG4gICAgICAuLi50aGlzLm1hcCgocm93OiBUKSA9PlxuICAgICAgICByb3cgaW5zdGFuY2VvZiBSb3cgPyByb3cuY2xvbmUoKSA6IFJvdy5mcm9tKHJvdykuY2xvbmUoKVxuICAgICAgKSxcbiAgICApO1xuICAgIHRhYmxlLm9wdGlvbnMgPSB7IC4uLnRoaXMub3B0aW9ucyB9O1xuICAgIHRhYmxlLmhlYWRlclJvdyA9IHRoaXMuaGVhZGVyUm93Py5jbG9uZSgpO1xuICAgIHJldHVybiB0YWJsZTtcbiAgfVxuXG4gIC8qKiBHZW5lcmF0ZSB0YWJsZSBzdHJpbmcuICovXG4gIHB1YmxpYyB0b1N0cmluZygpOiBzdHJpbmcge1xuICAgIHJldHVybiBuZXcgVGFibGVMYXlvdXQodGhpcywgdGhpcy5vcHRpb25zKS50b1N0cmluZygpO1xuICB9XG5cbiAgLyoqIFdyaXRlIHRhYmxlIHRvIHN0ZG91dC4gKi9cbiAgcHVibGljIHJlbmRlcigpOiB0aGlzIHtcbiAgICBjb25zb2xlLmxvZyh0aGlzLnRvU3RyaW5nKCkpO1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBtYXggY29sIHdpdGguXG4gICAqIEBwYXJhbSB3aWR0aCAgICAgTWF4IGNvbCB3aWR0aC5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBtYXhDb2xXaWR0aCh3aWR0aDogbnVtYmVyIHwgbnVtYmVyW10sIG92ZXJyaWRlID0gdHJ1ZSk6IHRoaXMge1xuICAgIGlmIChvdmVycmlkZSB8fCB0eXBlb2YgdGhpcy5vcHRpb25zLm1heENvbFdpZHRoID09PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICB0aGlzLm9wdGlvbnMubWF4Q29sV2lkdGggPSB3aWR0aDtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IG1pbiBjb2wgd2lkdGguXG4gICAqIEBwYXJhbSB3aWR0aCAgICAgTWluIGNvbCB3aWR0aC5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBtaW5Db2xXaWR0aCh3aWR0aDogbnVtYmVyIHwgbnVtYmVyW10sIG92ZXJyaWRlID0gdHJ1ZSk6IHRoaXMge1xuICAgIGlmIChvdmVycmlkZSB8fCB0eXBlb2YgdGhpcy5vcHRpb25zLm1pbkNvbFdpZHRoID09PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICB0aGlzLm9wdGlvbnMubWluQ29sV2lkdGggPSB3aWR0aDtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IHRhYmxlIGluZGVudGF0aW9uLlxuICAgKiBAcGFyYW0gd2lkdGggICAgIEluZGVudCB3aWR0aC5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBpbmRlbnQod2lkdGg6IG51bWJlciwgb3ZlcnJpZGUgPSB0cnVlKTogdGhpcyB7XG4gICAgaWYgKG92ZXJyaWRlIHx8IHR5cGVvZiB0aGlzLm9wdGlvbnMuaW5kZW50ID09PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICB0aGlzLm9wdGlvbnMuaW5kZW50ID0gd2lkdGg7XG4gICAgfVxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBjZWxsIHBhZGRpbmcuXG4gICAqIEBwYXJhbSBwYWRkaW5nICAgQ2VsbCBwYWRkaW5nLlxuICAgKiBAcGFyYW0gb3ZlcnJpZGUgIE92ZXJyaWRlIGV4aXN0aW5nIHZhbHVlLlxuICAgKi9cbiAgcHVibGljIHBhZGRpbmcocGFkZGluZzogbnVtYmVyIHwgbnVtYmVyW10sIG92ZXJyaWRlID0gdHJ1ZSk6IHRoaXMge1xuICAgIGlmIChvdmVycmlkZSB8fCB0eXBlb2YgdGhpcy5vcHRpb25zLnBhZGRpbmcgPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5wYWRkaW5nID0gcGFkZGluZztcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogRW5hYmxlL2Rpc2FibGUgY2VsbCBib3JkZXIuXG4gICAqIEBwYXJhbSBlbmFibGUgICAgRW5hYmxlL2Rpc2FibGUgY2VsbCBib3JkZXIuXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgT3ZlcnJpZGUgZXhpc3RpbmcgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgYm9yZGVyKGVuYWJsZTogYm9vbGVhbiwgb3ZlcnJpZGUgPSB0cnVlKTogdGhpcyB7XG4gICAgaWYgKG92ZXJyaWRlIHx8IHR5cGVvZiB0aGlzLm9wdGlvbnMuYm9yZGVyID09PSBcInVuZGVmaW5lZFwiKSB7XG4gICAgICB0aGlzLm9wdGlvbnMuYm9yZGVyID0gZW5hYmxlO1xuICAgIH1cbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBBbGlnbiB0YWJsZSBjb250ZW50LlxuICAgKiBAcGFyYW0gZGlyZWN0aW9uIEFsaWduIGRpcmVjdGlvbi5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBhbGlnbihkaXJlY3Rpb246IERpcmVjdGlvbiwgb3ZlcnJpZGUgPSB0cnVlKTogdGhpcyB7XG4gICAgaWYgKG92ZXJyaWRlIHx8IHR5cGVvZiB0aGlzLm9wdGlvbnMuYWxpZ24gPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5hbGlnbiA9IGRpcmVjdGlvbjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IGJvcmRlciBjaGFyYWN0ZXJzLlxuICAgKiBAcGFyYW0gY2hhcnMgQm9yZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgY2hhcnMoY2hhcnM6IElCb3JkZXJPcHRpb25zKTogdGhpcyB7XG4gICAgT2JqZWN0LmFzc2lnbih0aGlzLm9wdGlvbnMuY2hhcnMsIGNoYXJzKTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKiBHZXQgdGFibGUgaGVhZGVyLiAqL1xuICBwdWJsaWMgZ2V0SGVhZGVyKCk6IFJvdyB8IHVuZGVmaW5lZCB7XG4gICAgcmV0dXJuIHRoaXMuaGVhZGVyUm93O1xuICB9XG5cbiAgLyoqIEdldCB0YWJsZSBib2R5LiAqL1xuICBwdWJsaWMgZ2V0Qm9keSgpOiBUW10ge1xuICAgIHJldHVybiBbLi4udGhpc107XG4gIH1cblxuICAvKiogR2V0IG1hYyBjb2wgd2lkcnRoLiAqL1xuICBwdWJsaWMgZ2V0TWF4Q29sV2lkdGgoKTogbnVtYmVyIHwgbnVtYmVyW10ge1xuICAgIHJldHVybiB0aGlzLm9wdGlvbnMubWF4Q29sV2lkdGg7XG4gIH1cblxuICAvKiogR2V0IG1pbiBjb2wgd2lkdGguICovXG4gIHB1YmxpYyBnZXRNaW5Db2xXaWR0aCgpOiBudW1iZXIgfCBudW1iZXJbXSB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5taW5Db2xXaWR0aDtcbiAgfVxuXG4gIC8qKiBHZXQgdGFibGUgaW5kZW50YXRpb24uICovXG4gIHB1YmxpYyBnZXRJbmRlbnQoKTogbnVtYmVyIHtcbiAgICByZXR1cm4gdGhpcy5vcHRpb25zLmluZGVudDtcbiAgfVxuXG4gIC8qKiBHZXQgY2VsbCBwYWRkaW5nLiAqL1xuICBwdWJsaWMgZ2V0UGFkZGluZygpOiBudW1iZXIgfCBudW1iZXJbXSB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5wYWRkaW5nO1xuICB9XG5cbiAgLyoqIENoZWNrIGlmIHRhYmxlIGhhcyBib3JkZXIuICovXG4gIHB1YmxpYyBnZXRCb3JkZXIoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5ib3JkZXIgPT09IHRydWU7XG4gIH1cblxuICAvKiogQ2hlY2sgaWYgaGVhZGVyIHJvdyBoYXMgYm9yZGVyLiAqL1xuICBwdWJsaWMgaGFzSGVhZGVyQm9yZGVyKCk6IGJvb2xlYW4ge1xuICAgIGNvbnN0IGhhc0JvcmRlciA9IHRoaXMuaGVhZGVyUm93Py5oYXNCb3JkZXIoKTtcbiAgICByZXR1cm4gaGFzQm9yZGVyID09PSB0cnVlIHx8ICh0aGlzLmdldEJvcmRlcigpICYmIGhhc0JvcmRlciAhPT0gZmFsc2UpO1xuICB9XG5cbiAgLyoqIENoZWNrIGlmIHRhYmxlIGJvcmR5IGhhcyBib3JkZXIuICovXG4gIHB1YmxpYyBoYXNCb2R5Qm9yZGVyKCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLmdldEJvcmRlcigpIHx8XG4gICAgICB0aGlzLnNvbWUoKHJvdykgPT5cbiAgICAgICAgcm93IGluc3RhbmNlb2YgUm93XG4gICAgICAgICAgPyByb3cuaGFzQm9yZGVyKClcbiAgICAgICAgICA6IHJvdy5zb21lKChjZWxsKSA9PiBjZWxsIGluc3RhbmNlb2YgQ2VsbCA/IGNlbGwuZ2V0Qm9yZGVyIDogZmFsc2UpXG4gICAgICApO1xuICB9XG5cbiAgLyoqIENoZWNrIGlmIHRhYmxlIGhlYWRlciBvciBib2R5IGhhcyBib3JkZXIuICovXG4gIHB1YmxpYyBoYXNCb3JkZXIoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuaGFzSGVhZGVyQm9yZGVyKCkgfHwgdGhpcy5oYXNCb2R5Qm9yZGVyKCk7XG4gIH1cblxuICAvKiogR2V0IHRhYmxlIGFsaWdubWVudC4gKi9cbiAgcHVibGljIGdldEFsaWduKCk6IERpcmVjdGlvbiB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5hbGlnbiA/PyBcImxlZnRcIjtcbiAgfVxufVxuIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLFNBQVMsTUFBTSxRQUFpQixhQUFhLENBQUM7QUFDOUMsU0FBUyxJQUFJLFFBQW1CLFdBQVcsQ0FBQztBQUM1QyxTQUFTLFdBQVcsUUFBUSxhQUFhLENBQUM7QUFDMUMsU0FBeUIsR0FBRyxRQUFRLFVBQVUsQ0FBQztBQXlCL0MsMEJBQTBCLEdBQzFCLE9BQU8sTUFBTSxLQUFLLFNBQWdDLEtBQUs7SUFDckQsT0FBaUIsTUFBTSxHQUFZO1FBQUUsR0FBRyxNQUFNO0tBQUUsQ0FBQztJQUNqRCxBQUFVLE9BQU8sR0FBbUI7UUFDbEMsTUFBTSxFQUFFLENBQUM7UUFDVCxNQUFNLEVBQUUsS0FBSztRQUNiLFdBQVcsRUFBRSxRQUFRO1FBQ3JCLFdBQVcsRUFBRSxDQUFDO1FBQ2QsT0FBTyxFQUFFLENBQUM7UUFDVixLQUFLLEVBQUU7WUFBRSxHQUFHLEtBQUssQ0FBQyxNQUFNO1NBQUU7S0FDM0IsQ0FBQztJQUNGLEFBQVEsU0FBUyxDQUFPO0lBRXhCOzs7O0dBSUMsVUFDYSxJQUFJLENBQWlCLElBQWUsRUFBWTtRQUM1RCxNQUFNLEtBQUssR0FBRyxJQUFJLElBQUksSUFBSSxJQUFJLENBQUMsQUFBQztRQUNoQyxJQUFJLElBQUksWUFBWSxLQUFLLEVBQUU7WUFDekIsS0FBSyxDQUFDLE9BQU8sR0FBRztnQkFBRSxHQUFHLElBQUksQ0FBQyxPQUFPO2FBQUUsQ0FBQztZQUNwQyxLQUFLLENBQUMsU0FBUyxHQUFHLElBQUksQ0FBQyxTQUFTLEdB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