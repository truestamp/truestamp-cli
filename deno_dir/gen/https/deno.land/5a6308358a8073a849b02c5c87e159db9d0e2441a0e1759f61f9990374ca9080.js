/** Cell type */ // deno-lint-ignore ban-types
/** Cell representation. */ export class Cell {
    options;
    /** Get cell length. */ get length() {
        return this.toString().length;
    }
    /**
   * Create a new cell. If value is a cell, the value and all options of the cell
   * will be copied to the new cell.
   * @param value Cell or cell value.
   */ static from(value) {
        const cell = new this(value);
        if (value instanceof Cell) {
            cell.options = {
                ...value.options
            };
        }
        return cell;
    }
    /**
   * Cell constructor.
   * @param value Cell value.
   */ constructor(value){
        this.value = value;
        this.options = {};
    }
    /** Get cell value. */ toString() {
        return this.value.toString();
    }
    /**
   * Set cell value.
   * @param value Cell or cell value.
   */ setValue(value) {
        this.value = value;
        return this;
    }
    /**
   * Clone cell with all options.
   * @param value Cell or cell value.
   */ clone(value) {
        const cell = new Cell(value ?? this);
        cell.options = {
            ...this.options
        };
        return cell;
    }
    /**
   * Setter:
   */ /**
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
   * Set col span.
   * @param span      Number of cols to span.
   * @param override  Override existing value.
   */ colSpan(span, override = true) {
        if (override || typeof this.options.colSpan === "undefined") {
            this.options.colSpan = span;
        }
        return this;
    }
    /**
   * Set row span.
   * @param span      Number of rows to span.
   * @param override  Override existing value.
   */ rowSpan(span, override = true) {
        if (override || typeof this.options.rowSpan === "undefined") {
            this.options.rowSpan = span;
        }
        return this;
    }
    /**
   * Align cell content.
   * @param direction Align direction.
   * @param override  Override existing value.
   */ align(direction, override = true) {
        if (override || typeof this.options.align === "undefined") {
            this.options.align = direction;
        }
        return this;
    }
    /**
   * Getter:
   */ /** Check if cell has border. */ getBorder() {
        return this.options.border === true;
    }
    /** Get col span. */ getColSpan() {
        return typeof this.options.colSpan === "number" && this.options.colSpan > 0 ? this.options.colSpan : 1;
    }
    /** Get row span. */ getRowSpan() {
        return typeof this.options.rowSpan === "number" && this.options.rowSpan > 0 ? this.options.rowSpan : 1;
    }
    /** Get row span. */ getAlign() {
        return this.options.align ?? "left";
    }
    value;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI1LjAvdGFibGUvY2VsbC50cyJdLCJzb3VyY2VzQ29udGVudCI6WyIvKiogQ2VsbCB0eXBlICovXG4vLyBkZW5vLWxpbnQtaWdub3JlIGJhbi10eXBlc1xuZXhwb3J0IHR5cGUgSUNlbGwgPSBudW1iZXIgfCBzdHJpbmcgfCBTdHJpbmcgfCBDZWxsO1xuXG5leHBvcnQgdHlwZSBEaXJlY3Rpb24gPSBcImxlZnRcIiB8IFwicmlnaHRcIiB8IFwiY2VudGVyXCI7XG5cbi8qKiBDZWxsIG9wdGlvbnMuICovXG5leHBvcnQgaW50ZXJmYWNlIElDZWxsT3B0aW9ucyB7XG4gIGJvcmRlcj86IGJvb2xlYW47XG4gIGNvbFNwYW4/OiBudW1iZXI7XG4gIHJvd1NwYW4/OiBudW1iZXI7XG4gIGFsaWduPzogRGlyZWN0aW9uO1xufVxuXG4vKiogQ2VsbCByZXByZXNlbnRhdGlvbi4gKi9cbmV4cG9ydCBjbGFzcyBDZWxsIHtcbiAgcHJvdGVjdGVkIG9wdGlvbnM6IElDZWxsT3B0aW9ucyA9IHt9O1xuXG4gIC8qKiBHZXQgY2VsbCBsZW5ndGguICovXG4gIHB1YmxpYyBnZXQgbGVuZ3RoKCk6IG51bWJlciB7XG4gICAgcmV0dXJuIHRoaXMudG9TdHJpbmcoKS5sZW5ndGg7XG4gIH1cblxuICAvKipcbiAgICogQ3JlYXRlIGEgbmV3IGNlbGwuIElmIHZhbHVlIGlzIGEgY2VsbCwgdGhlIHZhbHVlIGFuZCBhbGwgb3B0aW9ucyBvZiB0aGUgY2VsbFxuICAgKiB3aWxsIGJlIGNvcGllZCB0byB0aGUgbmV3IGNlbGwuXG4gICAqIEBwYXJhbSB2YWx1ZSBDZWxsIG9yIGNlbGwgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgc3RhdGljIGZyb20odmFsdWU6IElDZWxsKTogQ2VsbCB7XG4gICAgY29uc3QgY2VsbCA9IG5ldyB0aGlzKHZhbHVlKTtcbiAgICBpZiAodmFsdWUgaW5zdGFuY2VvZiBDZWxsKSB7XG4gICAgICBjZWxsLm9wdGlvbnMgPSB7IC4uLnZhbHVlLm9wdGlvbnMgfTtcbiAgICB9XG4gICAgcmV0dXJuIGNlbGw7XG4gIH1cblxuICAvKipcbiAgICogQ2VsbCBjb25zdHJ1Y3Rvci5cbiAgICogQHBhcmFtIHZhbHVlIENlbGwgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgY29uc3RydWN0b3IocHJpdmF0ZSB2YWx1ZTogSUNlbGwpIHt9XG5cbiAgLyoqIEdldCBjZWxsIHZhbHVlLiAqL1xuICBwdWJsaWMgdG9TdHJpbmcoKTogc3RyaW5nIHtcbiAgICByZXR1cm4gdGhpcy52YWx1ZS50b1N0cmluZygpO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldCBjZWxsIHZhbHVlLlxuICAgKiBAcGFyYW0gdmFsdWUgQ2VsbCBvciBjZWxsIHZhbHVlLlxuICAgKi9cbiAgcHVibGljIHNldFZhbHVlKHZhbHVlOiBJQ2VsbCk6IHRoaXMge1xuICAgIHRoaXMudmFsdWUgPSB2YWx1ZTtcbiAgICByZXR1cm4gdGhpcztcbiAgfVxuXG4gIC8qKlxuICAgKiBDbG9uZSBjZWxsIHdpdGggYWxsIG9wdGlvbnMuXG4gICAqIEBwYXJhbSB2YWx1ZSBDZWxsIG9yIGNlbGwgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgY2xvbmUodmFsdWU/OiBJQ2VsbCk6IENlbGwge1xuICAgIGNvbnN0IGNlbGwgPSBuZXcgQ2VsbCh2YWx1ZSA/PyB0aGlzKTtcbiAgICBjZWxsLm9wdGlvbnMgPSB7IC4uLnRoaXMub3B0aW9ucyB9O1xuICAgIHJldHVybiBjZWxsO1xuICB9XG5cbiAgLyoqXG4gICAqIFNldHRlcjpcbiAgICovXG5cbiAgLyoqXG4gICAqIEVuYWJsZS9kaXNhYmxlIGNlbGwgYm9yZGVyLlxuICAgKiBAcGFyYW0gZW5hYmxlICAgIEVuYWJsZS9kaXNhYmxlIGNlbGwgYm9yZGVyLlxuICAgKiBAcGFyYW0gb3ZlcnJpZGUgIE92ZXJyaWRlIGV4aXN0aW5nIHZhbHVlLlxuICAgKi9cbiAgcHVibGljIGJvcmRlcihlbmFibGU6IGJvb2xlYW4sIG92ZXJyaWRlID0gdHJ1ZSk6IHRoaXMge1xuICAgIGlmIChvdmVycmlkZSB8fCB0eXBlb2YgdGhpcy5vcHRpb25zLmJvcmRlciA9PT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgdGhpcy5vcHRpb25zLmJvcmRlciA9IGVuYWJsZTtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IGNvbCBzcGFuLlxuICAgKiBAcGFyYW0gc3BhbiAgICAgIE51bWJlciBvZiBjb2xzIHRvIHNwYW4uXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgT3ZlcnJpZGUgZXhpc3RpbmcgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgY29sU3BhbihzcGFuOiBudW1iZXIsIG92ZXJyaWRlID0gdHJ1ZSk6IHRoaXMge1xuICAgIGlmIChvdmVycmlkZSB8fCB0eXBlb2YgdGhpcy5vcHRpb25zLmNvbFNwYW4gPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5jb2xTcGFuID0gc3BhbjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogU2V0IHJvdyBzcGFuLlxuICAgKiBAcGFyYW0gc3BhbiAgICAgIE51bWJlciBvZiByb3dzIHRvIHNwYW4uXG4gICAqIEBwYXJhbSBvdmVycmlkZSAgT3ZlcnJpZGUgZXhpc3RpbmcgdmFsdWUuXG4gICAqL1xuICBwdWJsaWMgcm93U3BhbihzcGFuOiBudW1iZXIsIG92ZXJyaWRlID0gdHJ1ZSk6IHRoaXMge1xuICAgIGlmIChvdmVycmlkZSB8fCB0eXBlb2YgdGhpcy5vcHRpb25zLnJvd1NwYW4gPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5yb3dTcGFuID0gc3BhbjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogQWxpZ24gY2VsbCBjb250ZW50LlxuICAgKiBAcGFyYW0gZGlyZWN0aW9uIEFsaWduIGRpcmVjdGlvbi5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBhbGlnbihkaXJlY3Rpb246IERpcmVjdGlvbiwgb3ZlcnJpZGUgPSB0cnVlKTogdGhpcyB7XG4gICAgaWYgKG92ZXJyaWRlIHx8IHR5cGVvZiB0aGlzLm9wdGlvbnMuYWxpZ24gPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5hbGlnbiA9IGRpcmVjdGlvbjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogR2V0dGVyOlxuICAgKi9cblxuICAvKiogQ2hlY2sgaWYgY2VsbCBoYXMgYm9yZGVyLiAqL1xuICBwdWJsaWMgZ2V0Qm9yZGVyKCk6IGJvb2xlYW4ge1xuICAgIHJldHVybiB0aGlzLm9wdGlvbnMuYm9yZGVyID09PSB0cnVlO1xuICB9XG5cbiAgLyoqIEdldCBjb2wgc3Bhbi4gKi9cbiAgcHVibGljIGdldENvbFNwYW4oKTogbnVtYmVyIHtcbiAgICByZXR1cm4gdHlwZW9mIHRoaXMub3B0aW9ucy5jb2xTcGFuID09PSBcIm51bWJlclwiICYmIHRoaXMub3B0aW9ucy5jb2xTcGFuID4gMFxuICAgICAgPyB0aGlzLm9wdGlvbnMuY29sU3BhblxuICAgICAgOiAxO1xuICB9XG5cbiAgLyoqIEdldCByb3cgc3Bhbi4gKi9cbiAgcHVibGljIGdldFJvd1NwYW4oKTogbnVtYmVyIHtcbiAgICByZXR1cm4gdHlwZW9mIHRoaXMub3B0aW9ucy5yb3dTcGFuID09PSBcIm51bWJlclwiICYmIHRoaXMub3B0aW9ucy5yb3dTcGFuID4gMFxuICAgICAgPyB0aGlzLm9wdGlvbnMucm93U3BhblxuICAgICAgOiAxO1xuICB9XG5cbiAgLyoqIEdldCByb3cgc3Bhbi4gKi9cbiAgcHVibGljIGdldEFsaWduKCk6IERpcmVjdGlvbiB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5hbGlnbiA/PyBcImxlZnRcIjtcbiAgfVxufVxuIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJB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