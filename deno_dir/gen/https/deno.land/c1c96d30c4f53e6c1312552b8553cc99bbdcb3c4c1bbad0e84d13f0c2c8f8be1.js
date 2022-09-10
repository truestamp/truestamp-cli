import { getStr, setArr, setStr } from "./wasm.ts";
import { Status, Types, Values } from "./constants.ts";
import { SqliteError } from "./error.ts";
/**
 * A prepared query which can be executed many
 * times.
 */ export class PreparedQuery {
    _wasm;
    _stmt;
    _openStatements;
    _status;
    _iterKv;
    _rowKeys;
    _finalized;
    /**
   * A prepared query which can be executed many
   * times.
   *
   * The constructor should never be used directly.
   * Instead a prepared query can be obtained by
   * calling `DB.prepareQuery`.
   */ constructor(wasm, stmt, openStatements){
        this._wasm = wasm;
        this._stmt = stmt;
        this._openStatements = openStatements;
        this._status = Status.Unknown;
        this._iterKv = false;
        this._finalized = false;
    }
    startQuery(params) {
        if (this._finalized) {
            throw new SqliteError("Query is finalized.");
        }
        // Reset query
        this._wasm.reset(this._stmt);
        this._wasm.clear_bindings(this._stmt);
        // Prepare parameter array
        let parameters = [];
        if (Array.isArray(params)) {
            parameters = params;
        } else if (typeof params === "object") {
            // Resolve parameter index for named parameter
            for (const key of Object.keys(params)){
                let name = key;
                // blank names default to ':'
                if (name[0] !== ":" && name[0] !== "@" && name[0] !== "$") {
                    name = `:${name}`;
                }
                const idx = setStr(this._wasm, name, (ptr)=>this._wasm.bind_parameter_index(this._stmt, ptr));
                if (idx === Values.Error) {
                    throw new SqliteError(`No parameter named '${name}'.`);
                }
                parameters[idx - 1] = params[key];
            }
        }
        // Bind parameters
        for(let i = 0; i < parameters.length; i++){
            let value = parameters[i];
            let status;
            switch(typeof value){
                case "boolean":
                    value = value ? 1 : 0;
                // fall through
                case "number":
                    if (Number.isSafeInteger(value)) {
                        status = this._wasm.bind_int(this._stmt, i + 1, value);
                    } else {
                        status = this._wasm.bind_double(this._stmt, i + 1, value);
                    }
                    break;
                case "bigint":
                    // bigint is bound as two 32bit integers and reassembled on the C side
                    if (value > 9223372036854775807n || value < -9223372036854775808n) {
                        throw new SqliteError(`BigInt value ${value} overflows 64 bit integer.`);
                    } else {
                        const posVal = value >= 0n ? value : -value;
                        const sign = value >= 0n ? 1 : -1;
                        const upper = Number(BigInt.asUintN(32, posVal >> 32n));
                        const lower = Number(BigInt.asUintN(32, posVal));
                        status = this._wasm.bind_big_int(this._stmt, i + 1, sign, upper, lower);
                    }
                    break;
                case "string":
                    status = setStr(this._wasm, value, (ptr)=>this._wasm.bind_text(this._stmt, i + 1, ptr));
                    break;
                default:
                    if (value instanceof Date) {
                        // Dates are allowed and bound to TEXT, formatted `YYYY-MM-DDTHH:MM:SS.SSSZ`
                        status = setStr(this._wasm, value.toISOString(), (ptr)=>this._wasm.bind_text(this._stmt, i + 1, ptr));
                    } else if (value instanceof Uint8Array) {
                        // Uint8Arrays are allowed and bound to BLOB
                        const size = value.length;
                        status = setArr(this._wasm, value, (ptr)=>this._wasm.bind_blob(this._stmt, i + 1, ptr, size));
                    } else if (value === null || value === undefined) {
                        // Both null and undefined result in a NULL entry
                        status = this._wasm.bind_null(this._stmt, i + 1);
                    } else {
                        throw new SqliteError(`Can not bind ${typeof value}.`);
                    }
                    break;
            }
            if (status !== Status.SqliteOk) {
                throw new SqliteError(this._wasm, status);
            }
        }
    }
    getQueryRow() {
        if (this._finalized) {
            throw new SqliteError("Query is finalized.");
        }
        const columnCount = this._wasm.column_count(this._stmt);
        const row = [];
        for(let i = 0; i < columnCount; i++){
            switch(this._wasm.column_type(this._stmt, i)){
                case Types.Integer:
                    row.push(this._wasm.column_int(this._stmt, i));
                    break;
                case Types.Float:
                    row.push(this._wasm.column_double(this._stmt, i));
                    break;
                case Types.Text:
                    row.push(getStr(this._wasm, this._wasm.column_text(this._stmt, i)));
                    break;
                case Types.Blob:
                    {
                        const ptr = this._wasm.column_blob(this._stmt, i);
                        if (ptr === 0) {
                            // Zero pointer results in null
                            row.push(null);
                        } else {
                            const length = this._wasm.column_bytes(this._stmt, i);
                            // Slice should copy the bytes, as it makes a shallow copy
                            row.push(new Uint8Array(this._wasm.memory.buffer, ptr, length).slice());
                        }
                        break;
                    }
                case Types.BigInteger:
                    {
                        const ptr1 = this._wasm.column_text(this._stmt, i);
                        row.push(BigInt(getStr(this._wasm, ptr1)));
                        break;
                    }
                default:
                    // TODO(dyedgreen): Differentiate between NULL and not-recognized?
                    row.push(null);
                    break;
            }
        }
        return row;
    }
    makeRowObject(row) {
        if (this._rowKeys == null) {
            const rowCount = this._wasm.column_count(this._stmt);
            this._rowKeys = [];
            for(let i = 0; i < rowCount; i++){
                this._rowKeys.push(getStr(this._wasm, this._wasm.column_name(this._stmt, i)));
            }
        }
        const obj = row.reduce((obj, val, idx)=>{
            obj[this._rowKeys[idx]] = val;
            return obj;
        }, {});
        return obj;
    }
    /**
   * Binds the given parameters to the query
   * and returns an iterator over rows.
   *
   * Using an iterator avoids loading all returned
   * rows into memory and hence allows to process a large
   * number of rows.
   *
   * # Example:
   * ```typescript
   * const query = db.prepareQuery<[number, string]>("SELECT id, name FROM people");
   * for (const [id, name] of query.iter()) {
   *   // ...
   * }
   * ```
   *
   * Calling `iter` invalidates any iterators previously returned
   * from this prepared query. Using an invalidated iterator is a bug.
   *
   * To avoid SQL injection, user-provided values
   * should always be passed to the database through
   * a query parameter.
   *
   * See `QueryParameterSet` for documentation on
   * how values can be bound to SQL statements.
   *
   * See `QueryParameter` for documentation on how
   * values are returned from the database.
   */ iter(params) {
        this.startQuery(params);
        this._status = this._wasm.step(this._stmt);
        if (this._status !== Status.SqliteRow && this._status !== Status.SqliteDone) {
            throw new SqliteError(this._wasm, this._status);
        }
        this._iterKv = false;
        return this;
    }
    /**
   * Like `iter` except each row is returned
   * as an object containing key-value pairs.
   */ iterEntries(params) {
        this.iter(params);
        this._iterKv = true;
        return this;
    }
    /**
   * @ignore
   *
   * Implements the iterable protocol. It is
   * a bug to call this method directly.
   */ [Symbol.iterator]() {
        return this;
    }
    /**
   * @ignore
   *
   * Implements the iterator protocol. It is
   * a bug to call this method directly.
   */ next() {
        if (this._status === Status.SqliteRow) {
            const value = this.getQueryRow();
            this._status = this._wasm.step(this._stmt);
            if (this._iterKv) {
                return {
                    value: this.makeRowObject(value),
                    done: false
                };
            } else {
                return {
                    value,
                    done: false
                };
            }
        } else if (this._status === Status.SqliteDone) {
            return {
                value: null,
                done: true
            };
        } else {
            throw new SqliteError(this._wasm, this._status);
        }
    }
    /**
   * Binds the given parameters to the query
   * and returns an array containing all resulting
   * rows.
   *
   * Calling `all` invalidates any iterators
   * previously returned by calls to `iter`.
   * Using an invalidated iterator is a bug.
   *
   * To avoid SQL injection, user-provided values
   * should always be passed to the database through
   * a query parameter.
   *
   * See `QueryParameterSet` for documentation on
   * how values can be bound to SQL statements.
   *
   * See `QueryParameter` for documentation on how
   * values are returned from the database.
   */ all(params) {
        this.startQuery(params);
        const rows = [];
        this._status = this._wasm.step(this._stmt);
        while(this._status === Status.SqliteRow){
            rows.push(this.getQueryRow());
            this._status = this._wasm.step(this._stmt);
        }
        if (this._status !== Status.SqliteDone) {
            throw new SqliteError(this._wasm, this._status);
        }
        return rows;
    }
    /**
   * Like `all` except each row is returned
   * as an object containing key-value pairs.
   */ allEntries(params) {
        return this.all(params).map((row)=>this.makeRowObject(row));
    }
    /**
   * Binds the given parameters to the query and
   * returns exactly one row.
   *
   * If the query does not return exactly one row,
   * this throws an error.
   *
   * Calling `one` invalidates any iterators
   * previously returned by calls to `iter`.
   * Using an invalidated iterator is a bug.
   *
   * To avoid SQL injection, user-provided values
   * should always be passed to the database through
   * a query parameter.
   *
   * See `QueryParameterSet` for documentation on
   * how values can be bound to SQL statements.
   *
   * See `QueryParameter` for documentation on how
   * values are returned from the database.
   */ one(params) {
        this.startQuery(params);
        // Get first row
        this._status = this._wasm.step(this._stmt);
        if (this._status !== Status.SqliteRow) {
            if (this._status === Status.SqliteDone) {
                throw new SqliteError("The query did not return any rows.");
            } else {
                throw new SqliteError(this._wasm, this._status);
            }
        }
        const row = this.getQueryRow();
        // Ensure the query only returns one row
        this._status = this._wasm.step(this._stmt);
        if (this._status !== Status.SqliteDone) {
            if (this._status === Status.SqliteRow) {
                throw new SqliteError("The query returned more than one row.");
            } else {
                throw new SqliteError(this._wasm, this._status);
            }
        }
        return row;
    }
    /**
   * Like `one` except the row is returned
   * as an object containing key-value pairs.
   */ oneEntry(params) {
        return this.makeRowObject(this.one(params));
    }
    /**
   * Binds the given parameters to the query and
   * executes the query, ignoring any rows which
   * might be returned.
   *
   * Using this method is more efficient when the
   * rows returned by a query are not needed or
   * the query does not return any rows.
   *
   * Calling `execute` invalidates any iterators
   * previously returned by calls to `iter`.
   * Using an invalidated iterator is a bug.
   *
   * To avoid SQL injection, user-provided values
   * should always be passed to the database through
   * a query parameter.
   *
   * See `QueryParameterSet` for documentation on
   * how values can be bound to SQL statements.
   */ execute(params) {
        this.startQuery(params);
        this._status = this._wasm.step(this._stmt);
        while(this._status === Status.SqliteRow){
            this._status = this._wasm.step(this._stmt);
        }
        if (this._status !== Status.SqliteDone) {
            throw new SqliteError(this._wasm, this._status);
        }
    }
    /**
   * Closes the prepared query. This must be
   * called once the query is no longer needed
   * to avoid leaking resources.
   *
   * After a prepared query has been finalized,
   * trying to call `iter`, `all`, `one`,
   * `execute`, or `columns`, or using iterators which where
   * previously obtained from the finalized query
   * is a bug.
   *
   * `finalize` may safely be called multiple
   * times.
   */ finalize() {
        if (!this._finalized) {
            this._wasm.finalize(this._stmt);
            this._openStatements.delete(this._stmt);
            this._finalized = true;
        }
    }
    /**
   * Returns the column names for the query
   * results.
   *
   * This method returns an array of objects,
   * where each object has the following properties:
   *
   * | Property     | Value                                      |
   * |--------------|--------------------------------------------|
   * | `name`       | the result of `sqlite3_column_name`        |
   * | `originName` | the result of `sqlite3_column_origin_name` |
   * | `tableName`  | the result of `sqlite3_column_table_name`  |
   */ columns() {
        if (this._finalized) {
            throw new SqliteError("Unable to retrieve column names from finalized transaction.");
        }
        const columnCount = this._wasm.column_count(this._stmt);
        const columns = [];
        for(let i = 0; i < columnCount; i++){
            const name = getStr(this._wasm, this._wasm.column_name(this._stmt, i));
            const originName = getStr(this._wasm, this._wasm.column_origin_name(this._stmt, i));
            const tableName = getStr(this._wasm, this._wasm.column_table_name(this._stmt, i));
            columns.push({
                name,
                originName,
                tableName
            });
        }
        return columns;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvc3FsaXRlQHYzLjQuMS9zcmMvcXVlcnkudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgU3RhdGVtZW50UHRyLCBXYXNtIH0gZnJvbSBcIi4uL2J1aWxkL3NxbGl0ZS5qc1wiO1xuaW1wb3J0IHsgZ2V0U3RyLCBzZXRBcnIsIHNldFN0ciB9IGZyb20gXCIuL3dhc20udHNcIjtcbmltcG9ydCB7IFN0YXR1cywgVHlwZXMsIFZhbHVlcyB9IGZyb20gXCIuL2NvbnN0YW50cy50c1wiO1xuaW1wb3J0IHsgU3FsaXRlRXJyb3IgfSBmcm9tIFwiLi9lcnJvci50c1wiO1xuXG4vKipcbiAqIFRoZSBkZWZhdWx0IHR5cGUgZm9yIHJldHVybmVkIHJvd3MuXG4gKi9cbmV4cG9ydCB0eXBlIFJvdyA9IEFycmF5PHVua25vd24+O1xuXG4vKipcbiAqIFRoZSBkZWZhdWx0IHR5cGUgZm9yIHJvdyByZXR1cm5lZFxuICogYXMgb2JqZWN0cy5cbiAqL1xuZXhwb3J0IHR5cGUgUm93T2JqZWN0ID0gUmVjb3JkPHN0cmluZywgdW5rbm93bj47XG5cbi8qKlxuICogUG9zc2libGUgcGFyYW1ldGVyIHZhbHVlcyB0byBiZSBib3VuZCB0byBhIHF1ZXJ5LlxuICpcbiAqIFdoZW4gdmFsdWVzIGFyZSBib3VuZCB0byBhIHF1ZXJ5LCB0aGV5IGFyZVxuICogY29udmVydGVkIGJldHdlZW4gSmF2YVNjcmlwdCBhbmQgU1FMaXRlIHR5cGVzXG4gKiBpbiB0aGUgZm9sbG93aW5nIHdheTpcbiAqXG4gKiB8IEpTIHR5cGUgaW4gfCBTUUwgdHlwZSAgICAgICAgfCBKUyB0eXBlIG91dCAgICAgIHxcbiAqIHwtLS0tLS0tLS0tLS18LS0tLS0tLS0tLS0tLS0tLS18LS0tLS0tLS0tLS0tLS0tLS0tfFxuICogfCBudW1iZXIgICAgIHwgSU5URUdFUiBvciBSRUFMIHwgbnVtYmVyIG9yIGJpZ2ludCB8XG4gKiB8IGJpZ2ludCAgICAgfCBJTlRFR0VSICAgICAgICAgfCBudW1iZXIgb3IgYmlnaW50IHxcbiAqIHwgYm9vbGVhbiAgICB8IElOVEVHRVIgICAgICAgICB8IG51bWJlciAgICAgICAgICAgfFxuICogfCBzdHJpbmcgICAgIHwgVEVYVCAgICAgICAgICAgIHwgc3RyaW5nICAgICAgICAgICB8XG4gKiB8IERhdGUgICAgICAgfCBURVhUICAgICAgICAgICAgfCBzdHJpbmcgICAgICAgICAgIHxcbiAqIHwgVWludDhBcnJheSB8IEJMT0IgICAgICAgICAgICB8IFVpbnQ4QXJyYXkgICAgICAgfFxuICogfCBudWxsICAgICAgIHwgTlVMTCAgICAgICAgICAgIHwgbnVsbCAgICAgICAgICAgICB8XG4gKiB8IHVuZGVmaW5lZCAgfCBOVUxMICAgICAgICAgICAgfCBudWxsICAgICAgICAgICAgIHxcbiAqXG4gKiBJZiBubyB2YWx1ZSBpcyBwcm92aWRlZCBmb3IgYSBnaXZlbiBwYXJhbWV0ZXIsXG4gKiBTUUxpdGUgd2lsbCBkZWZhdWx0IHRvIE5VTEwuXG4gKlxuICogSWYgYSBgYmlnaW50YCBpcyBib3VuZCwgaXQgaXMgY29udmVydGVkIHRvIGFcbiAqIHNpZ25lZCA2NCBiaXQgaW50ZWdlciwgd2hpY2ggbWF5IG92ZXJmbG93LlxuICpcbiAqIElmIGFuIGludGVnZXIgdmFsdWUgaXMgcmVhZCBmcm9tIHRoZSBkYXRhYmFzZSwgd2hpY2hcbiAqIGlzIHRvbyBiaWcgdG8gc2FmZWx5IGJlIGNvbnRhaW5lZCBpbiBhIGBudW1iZXJgLCBpdFxuICogaXMgYXV0b21hdGljYWxseSByZXR1cm5lZCBhcyBhIGBiaWdpbnRgLlxuICpcbiAqIElmIGEgYERhdGVgIGlzIGJvdW5kLCBpdCB3aWxsIGJlIGNvbnZlcnRlZCB0b1xuICogYW4gSVNPIDg2MDEgc3RyaW5nOiBgWVlZWS1NTS1ERFRISDpNTTpTUy5TU1NaYC5cbiAqIFRoaXMgZm9ybWF0IGlzIHVuZGVyc3Rvb2QgYnkgYnVpbHQtaW4gU1FMaXRlXG4gKiBkYXRlLXRpbWUgZnVuY3Rpb25zLiBBbHNvIHNlZSBodHRwczovL3NxbGl0ZS5vcmcvbGFuZ19kYXRlZnVuYy5odG1sLlxuICovXG5leHBvcnQgdHlwZSBRdWVyeVBhcmFtZXRlciA9XG4gIHwgYm9vbGVhblxuICB8IG51bWJlclxuICB8IGJpZ2ludFxuICB8IHN0cmluZ1xuICB8IG51bGxcbiAgfCB1bmRlZmluZWRcbiAgfCBEYXRlXG4gIHwgVWludDhBcnJheTtcblxuLyoqXG4gKiBBIHNldCBvZiBxdWVyeSBwYXJhbWV0ZXJzLlxuICpcbiAqIFdoZW4gYSBxdWVyeSBpcyBjb25zdHJ1Y3RlZCwgaXQgY2FuIGNvbnRhaW5cbiAqIGVpdGhlciBwb3NpdGlvbmFsIG9yIG5hbWVkIHBhcmFtZXRlcnMuIEZvclxuICogbW9yZSBpbmZvcm1hdGlvbiBzZWUgaHR0cHM6Ly93d3cuc3FsaXRlLm9yZy9sYW5nX2V4cHIuaHRtbCNwYXJhbWV0ZXJzLlxuICpcbiAqIEEgc2V0IG9mIHBhcmFtZXRlcnMgY2FuIGJlIHBhc3NlZCB0b1xuICogYSBxdWVyeSBtZXRob2QgZWl0aGVyIGFzIGFuIGFycmF5IG9mXG4gKiBwYXJhbWV0ZXJzIChpbiBwb3NpdGlvbmFsIG9yZGVyKSwgb3JcbiAqIGFzIGFuIG9iamVjdCB3aGljaCBtYXBzIHBhcmFtZXRlciBuYW1lc1xuICogdG8gdGhlaXIgdmFsdWVzOlxuICpcbiAqIHwgU1FMIFBhcmFtZXRlciB8IFF1ZXJ5UGFyYW1ldGVyU2V0ICAgICAgIHxcbiAqIHwtLS0tLS0tLS0tLS0tLS18LS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLXxcbiAqIHwgYD9OTk5gIG9yIGA/YCB8IE5OTi10aCB2YWx1ZSBpbiBhcnJheSAgIHxcbiAqIHwgYDpBQUFBYCAgICAgICB8IHZhbHVlIGBBQUFBYCBvciBgOkFBQUFgIHxcbiAqIHwgYEBBQUFBYCAgICAgICB8IHZhbHVlIGBAQUFBQWAgICAgICAgICAgIHxcbiAqIHwgYCRBQUFBYCAgICAgICB8IHZhbHVlIGAkQUFBQWAgICAgICAgICAgIHxcbiAqXG4gKiBTZWUgYFF1ZXJ5UGFyYW1ldGVyYCBmb3IgZG9jdW1lbnRhdGlvbiBvblxuICogaG93IHZhbHVlcyBhcmUgY29udmVydGVkIGJldHdlZW4gU1FMXG4gKiBhbmQgSmF2YVNjcmlwdCB0eXBlcy5cbiAqL1xuZXhwb3J0IHR5cGUgUXVlcnlQYXJhbWV0ZXJTZXQgPVxuICB8IFJlY29yZDxzdHJpbmcsIFF1ZXJ5UGFyYW1ldGVyPlxuICB8IEFycmF5PFF1ZXJ5UGFyYW1ldGVyPjtcblxuLyoqXG4gKiBOYW1lIG9mIGEgY29sdW1uIGluIGEgZGF0YWJhc2UgcXVlcnkuXG4gKi9cbmV4cG9ydCBpbnRlcmZhY2UgQ29sdW1uTmFtZSB7XG4gIG5hbWU6IHN0cmluZztcbiAgb3JpZ2luTmFtZTogc3RyaW5nO1xuICB0YWJsZU5hbWU6IHN0cmluZztcbn1cblxuaW50ZXJmYWNlIFJvd3NJdGVyYXRvcjxSPiB7XG4gIG5leHQ6ICgpID0+IEl0ZXJhdG9yUmVzdWx0PFI+O1xuICBbU3ltYm9sLml0ZXJhdG9yXTogKCkgPT4gUm93c0l0ZXJhdG9yPFI+O1xufVxuXG4vKipcbiAqIEEgcHJlcGFyZWQgcXVlcnkgd2hpY2ggY2FuIGJlIGV4ZWN1dGVkIG1hbnlcbiAqIHRpbWVzLlxuICovXG5leHBvcnQgY2xhc3MgUHJlcGFyZWRRdWVyeTxcbiAgUiBleHRlbmRzIFJvdyA9IFJvdyxcbiAgTyBleHRlbmRzIFJvd09iamVjdCA9IFJvd09iamVjdCxcbiAgUCBleHRlbmRzIFF1ZXJ5UGFyYW1ldGVyU2V0ID0gUXVlcnlQYXJhbWV0ZXJTZXQsXG4+IHtcbiAgcHJpdmF0ZSBfd2FzbTogV2FzbTtcbiAgcHJpdmF0ZSBfc3RtdDogU3RhdGVtZW50UHRyO1xuICBwcml2YXRlIF9vcGVuU3RhdGVtZW50czogU2V0PFN0YXRlbWVudFB0cj47XG5cbiAgcHJpdmF0ZSBfc3RhdHVzOiBudW1iZXI7XG4gIHByaXZhdGUgX2l0ZXJLdjogYm9vbGVhbjtcbiAgcHJpdmF0ZSBfcm93S2V5cz86IEFycmF5PHN0cmluZz47XG4gIHByaXZhdGUgX2ZpbmFsaXplZDogYm9vbGVhbjtcblxuICAvKipcbiAgICogQSBwcmVwYXJlZCBxdWVyeSB3aGljaCBjYW4gYmUgZXhlY3V0ZWQgbWFueVxuICAgKiB0aW1lcy5cbiAgICpcbiAgICogVGhlIGNvbnN0cnVjdG9yIHNob3VsZCBuZXZlciBiZSB1c2VkIGRpcmVjdGx5LlxuICAgKiBJbnN0ZWFkIGEgcHJlcGFyZWQgcXVlcnkgY2FuIGJlIG9idGFpbmVkIGJ5XG4gICAqIGNhbGxpbmcgYERCLnByZXBhcmVRdWVyeWAuXG4gICAqL1xuICBjb25zdHJ1Y3RvcihcbiAgICB3YXNtOiBXYXNtLFxuICAgIHN0bXQ6IFN0YXRlbWVudFB0cixcbiAgICBvcGVuU3RhdGVtZW50czogU2V0PFN0YXRlbWVudFB0cj4sXG4gICkge1xuICAgIHRoaXMuX3dhc20gPSB3YXNtO1xuICAgIHRoaXMuX3N0bXQgPSBzdG10O1xuICAgIHRoaXMuX29wZW5TdGF0ZW1lbnRzID0gb3BlblN0YXRlbWVudHM7XG5cbiAgICB0aGlzLl9zdGF0dXMgPSBTdGF0dXMuVW5rbm93bjtcbiAgICB0aGlzLl9pdGVyS3YgPSBmYWxzZTtcbiAgICB0aGlzLl9maW5hbGl6ZWQgPSBmYWxzZTtcbiAgfVxuXG4gIHByaXZhdGUgc3RhcnRRdWVyeShwYXJhbXM/OiBQKSB7XG4gICAgaWYgKHRoaXMuX2ZpbmFsaXplZCkge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKFwiUXVlcnkgaXMgZmluYWxpemVkLlwiKTtcbiAgICB9XG5cbiAgICAvLyBSZXNldCBxdWVyeVxuICAgIHRoaXMuX3dhc20ucmVzZXQodGhpcy5fc3RtdCk7XG4gICAgdGhpcy5fd2FzbS5jbGVhcl9iaW5kaW5ncyh0aGlzLl9zdG10KTtcblxuICAgIC8vIFByZXBhcmUgcGFyYW1ldGVyIGFycmF5XG4gICAgbGV0IHBhcmFtZXRlcnMgPSBbXTtcbiAgICBpZiAoQXJyYXkuaXNBcnJheShwYXJhbXMpKSB7XG4gICAgICBwYXJhbWV0ZXJzID0gcGFyYW1zO1xuICAgIH0gZWxzZSBpZiAodHlwZW9mIHBhcmFtcyA9PT0gXCJvYmplY3RcIikge1xuICAgICAgLy8gUmVzb2x2ZSBwYXJhbWV0ZXIgaW5kZXggZm9yIG5hbWVkIHBhcmFtZXRlclxuICAgICAgZm9yIChjb25zdCBrZXkgb2YgT2JqZWN0LmtleXMocGFyYW1zKSkge1xuICAgICAgICBsZXQgbmFtZSA9IGtleTtcbiAgICAgICAgLy8gYmxhbmsgbmFtZXMgZGVmYXVsdCB0byAnOidcbiAgICAgICAgaWYgKG5hbWVbMF0gIT09IFwiOlwiICYmIG5hbWVbMF0gIT09IFwiQFwiICYmIG5hbWVbMF0gIT09IFwiJFwiKSB7XG4gICAgICAgICAgbmFtZSA9IGA6JHtuYW1lfWA7XG4gICAgICAgIH1cbiAgICAgICAgY29uc3QgaWR4ID0gc2V0U3RyKFxuICAgICAgICAgIHRoaXMuX3dhc20sXG4gICAgICAgICAgbmFtZSxcbiAgICAgICAgICAocHRyKSA9PiB0aGlzLl93YXNtLmJpbmRfcGFyYW1ldGVyX2luZGV4KHRoaXMuX3N0bXQsIHB0ciksXG4gICAgICAgICk7XG4gICAgICAgIGlmIChpZHggPT09IFZhbHVlcy5FcnJvcikge1xuICAgICAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcihgTm8gcGFyYW1ldGVyIG5hbWVkICcke25hbWV9Jy5gKTtcbiAgICAgICAgfVxuICAgICAgICBwYXJhbWV0ZXJzW2lkeCAtIDFdID0gcGFyYW1zW2tleV07XG4gICAgICB9XG4gICAgfVxuXG4gICAgLy8gQmluZCBwYXJhbWV0ZXJzXG4gICAgZm9yIChsZXQgaSA9IDA7IGkgPCBwYXJhbWV0ZXJzLmxlbmd0aDsgaSsrKSB7XG4gICAgICBsZXQgdmFsdWUgPSBwYXJhbWV0ZXJzW2ldO1xuICAgICAgbGV0IHN0YXR1cztcbiAgICAgIHN3aXRjaCAodHlwZW9mIHZhbHVlKSB7XG4gICAgICAgIGNhc2UgXCJib29sZWFuXCI6XG4gICAgICAgICAgdmFsdWUgPSB2YWx1ZSA/IDEgOiAwO1xuICAgICAgICAgIC8vIGZhbGwgdGhyb3VnaFxuICAgICAgICBjYXNlIFwibnVtYmVyXCI6XG4gICAgICAgICAgaWYgKE51bWJlci5pc1NhZmVJbnRlZ2VyKHZhbHVlKSkge1xuICAgICAgICAgICAgc3RhdHVzID0gdGhpcy5fd2FzbS5iaW5kX2ludCh0aGlzLl9zdG10LCBpICsgMSwgdmFsdWUpO1xuICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICBzdGF0dXMgPSB0aGlzLl93YXNtLmJpbmRfZG91YmxlKHRoaXMuX3N0bXQsIGkgKyAxLCB2YWx1ZSk7XG4gICAgICAgICAgfVxuICAgICAgICAgIGJyZWFrO1xuICAgICAgICBjYXNlIFwiYmlnaW50XCI6XG4gICAgICAgICAgLy8gYmlnaW50IGlzIGJvdW5kIGFzIHR3byAzMmJpdCBpbnRlZ2VycyBhbmQgcmVhc3NlbWJsZWQgb24gdGhlIEMgc2lkZVxuICAgICAgICAgIGlmICh2YWx1ZSA+IDkyMjMzNzIwMzY4NTQ3NzU4MDduIHx8IHZhbHVlIDwgLTkyMjMzNzIwMzY4NTQ3NzU4MDhuKSB7XG4gICAgICAgICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IoXG4gICAgICAgICAgICAgIGBCaWdJbnQgdmFsdWUgJHt2YWx1ZX0gb3ZlcmZsb3dzIDY0IGJpdCBpbnRlZ2VyLmAsXG4gICAgICAgICAgICApO1xuICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICBjb25zdCBwb3NWYWwgPSB2YWx1ZSA+PSAwbiA/IHZhbHVlIDogLXZhbHVlO1xuICAgICAgICAgICAgY29uc3Qgc2lnbiA9IHZhbHVlID49IDBuID8gMSA6IC0xO1xuICAgICAgICAgICAgY29uc3QgdXBwZXIgPSBOdW1iZXIoQmlnSW50LmFzVWludE4oMzIsIHBvc1ZhbCA+PiAzMm4pKTtcbiAgICAgICAgICAgIGNvbnN0IGxvd2VyID0gTnVtYmVyKEJpZ0ludC5hc1VpbnROKDMyLCBwb3NWYWwpKTtcbiAgICAgICAgICAgIHN0YXR1cyA9IHRoaXMuX3dhc20uYmluZF9iaWdfaW50KFxuICAgICAgICAgICAgICB0aGlzLl9zdG10LFxuICAgICAgICAgICAgICBpICsgMSxcbiAgICAgICAgICAgICAgc2lnbixcbiAgICAgICAgICAgICAgdXBwZXIsXG4gICAgICAgICAgICAgIGxvd2VyLFxuICAgICAgICAgICAgKTtcbiAgICAgICAgICB9XG4gICAgICAgICAgYnJlYWs7XG4gICAgICAgIGNhc2UgXCJzdHJpbmdcIjpcbiAgICAgICAgICBzdGF0dXMgPSBzZXRTdHIoXG4gICAgICAgICAgICB0aGlzLl93YXNtLFxuICAgICAgICAgICAgdmFsdWUsXG4gICAgICAgICAgICAocHRyKSA9PiB0aGlzLl93YXNtLmJpbmRfdGV4dCh0aGlzLl9zdG10LCBpICsgMSwgcHRyKSxcbiAgICAgICAgICApO1xuICAgICAgICAgIGJyZWFrO1xuICAgICAgICBkZWZhdWx0OlxuICAgICAgICAgIGlmICh2YWx1ZSBpbnN0YW5jZW9mIERhdGUpIHtcbiAgICAgICAgICAgIC8vIERhdGVzIGFyZSBhbGxvd2VkIGFuZCBib3VuZCB0byBURVhULCBmb3JtYXR0ZWQgYFlZWVktTU0tRERUSEg6TU06U1MuU1NTWmBcbiAgICAgICAgICAgIHN0YXR1cyA9IHNldFN0cihcbiAgICAgICAgICAgICAgdGhpcy5fd2FzbSxcbiAgICAgICAgICAgICAgdmFsdWUudG9JU09TdHJpbmcoKSxcbiAgICAgICAgICAgICAgKHB0cikgPT4gdGhpcy5fd2FzbS5iaW5kX3RleHQodGhpcy5fc3RtdCwgaSArIDEsIHB0ciksXG4gICAgICAgICAgICApO1xuICAgICAgICAgIH0gZWxzZSBpZiAodmFsdWUgaW5zdGFuY2VvZiBVaW50OEFycmF5KSB7XG4gICAgICAgICAgICAvLyBVaW50OEFycmF5cyBhcmUgYWxsb3dlZCBhbmQgYm91bmQgdG8gQkxPQlxuICAgICAgICAgICAgY29uc3Qgc2l6ZSA9IHZhbHVlLmxlbmd0aDtcbiAgICAgICAgICAgIHN0YXR1cyA9IHNldEFycihcbiAgICAgICAgICAgICAgdGhpcy5fd2FzbSxcbiAgICAgICAgICAgICAgdmFsdWUsXG4gICAgICAgICAgICAgIChwdHIpID0+IHRoaXMuX3dhc20uYmluZF9ibG9iKHRoaXMuX3N0bXQsIGkgKyAxLCBwdHIsIHNpemUpLFxuICAgICAgICAgICAgKTtcbiAgICAgICAgICB9IGVsc2UgaWYgKHZhbHVlID09PSBudWxsIHx8IHZhbHVlID09PSB1bmRlZmluZWQpIHtcbiAgICAgICAgICAgIC8vIEJvdGggbnVsbCBhbmQgdW5kZWZpbmVkIHJlc3VsdCBpbiBhIE5VTEwgZW50cnlcbiAgICAgICAgICAgIHN0YXR1cyA9IHRoaXMuX3dhc20uYmluZF9udWxsKHRoaXMuX3N0bXQsIGkgKyAxKTtcbiAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKGBDYW4gbm90IGJpbmQgJHt0eXBlb2YgdmFsdWV9LmApO1xuICAgICAgICAgIH1cbiAgICAgICAgICBicmVhaztcbiAgICAgIH1cbiAgICAgIGlmIChzdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVPaykge1xuICAgICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSwgc3RhdHVzKTtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICBwcml2YXRlIGdldFF1ZXJ5Um93KCk6IFIge1xuICAgIGlmICh0aGlzLl9maW5hbGl6ZWQpIHtcbiAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcihcIlF1ZXJ5IGlzIGZpbmFsaXplZC5cIik7XG4gICAgfVxuXG4gICAgY29uc3QgY29sdW1uQ291bnQgPSB0aGlzLl93YXNtLmNvbHVtbl9jb3VudCh0aGlzLl9zdG10KTtcbiAgICBjb25zdCByb3c6IFJvdyA9IFtdO1xuICAgIGZvciAobGV0IGkgPSAwOyBpIDwgY29sdW1uQ291bnQ7IGkrKykge1xuICAgICAgc3dpdGNoICh0aGlzLl93YXNtLmNvbHVtbl90eXBlKHRoaXMuX3N0bXQsIGkpKSB7XG4gICAgICAgIGNhc2UgVHlwZXMuSW50ZWdlcjpcbiAgICAgICAgICByb3cucHVzaCh0aGlzLl93YXNtLmNvbHVtbl9pbnQodGhpcy5fc3RtdCwgaSkpO1xuICAgICAgICAgIGJyZWFrO1xuICAgICAgICBjYXNlIFR5cGVzLkZsb2F0OlxuICAgICAgICAgIHJvdy5wdXNoKHRoaXMuX3dhc20uY29sdW1uX2RvdWJsZSh0aGlzLl9zdG10LCBpKSk7XG4gICAgICAgICAgYnJlYWs7XG4gICAgICAgIGNhc2UgVHlwZXMuVGV4dDpcbiAgICAgICAgICByb3cucHVzaChcbiAgICAgICAgICAgIGdldFN0cihcbiAgICAgICAgICAgICAgdGhpcy5fd2FzbSxcbiAgICAgICAgICAgICAgdGhpcy5fd2FzbS5jb2x1bW5fdGV4dCh0aGlzLl9zdG10LCBpKSxcbiAgICAgICAgICAgICksXG4gICAgICAgICAgKTtcbiAgICAgICAgICBicmVhaztcbiAgICAgICAgY2FzZSBUeXBlcy5CbG9iOiB7XG4gICAgICAgICAgY29uc3QgcHRyID0gdGhpcy5fd2FzbS5jb2x1bW5fYmxvYih0aGlzLl9zdG10LCBpKTtcbiAgICAgICAgICBpZiAocHRyID09PSAwKSB7XG4gICAgICAgICAgICAvLyBaZXJvIHBvaW50ZXIgcmVzdWx0cyBpbiBudWxsXG4gICAgICAgICAgICByb3cucHVzaChudWxsKTtcbiAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgY29uc3QgbGVuZ3RoID0gdGhpcy5fd2FzbS5jb2x1bW5fYnl0ZXModGhpcy5fc3RtdCwgaSk7XG4gICAgICAgICAgICAvLyBTbGljZSBzaG91bGQgY29weSB0aGUgYnl0ZXMsIGFzIGl0IG1ha2VzIGEgc2hhbGxvdyBjb3B5XG4gICAgICAgICAgICByb3cucHVzaChcbiAgICAgICAgICAgICAgbmV3IFVpbnQ4QXJyYXkodGhpcy5fd2FzbS5tZW1vcnkuYnVmZmVyLCBwdHIsIGxlbmd0aCkuc2xpY2UoKSxcbiAgICAgICAgICAgICk7XG4gICAgICAgICAgfVxuICAgICAgICAgIGJyZWFrO1xuICAgICAgICB9XG4gICAgICAgIGNhc2UgVHlwZXMuQmlnSW50ZWdlcjoge1xuICAgICAgICAgIGNvbnN0IHB0ciA9IHRoaXMuX3dhc20uY29sdW1uX3RleHQodGhpcy5fc3RtdCwgaSk7XG4gICAgICAgICAgcm93LnB1c2goQmlnSW50KGdldFN0cih0aGlzLl93YXNtLCBwdHIpKSk7XG4gICAgICAgICAgYnJlYWs7XG4gICAgICAgIH1cbiAgICAgICAgZGVmYXVsdDpcbiAgICAgICAgICAvLyBUT0RPKGR5ZWRncmVlbik6IERpZmZlcmVudGlhdGUgYmV0d2VlbiBOVUxMIGFuZCBub3QtcmVjb2duaXplZD9cbiAgICAgICAgICByb3cucHVzaChudWxsKTtcbiAgICAgICAgICBicmVhaztcbiAgICAgIH1cbiAgICB9XG4gICAgcmV0dXJuIHJvdyBhcyBSO1xuICB9XG5cbiAgcHJpdmF0ZSBtYWtlUm93T2JqZWN0KHJvdzogUm93KTogTyB7XG4gICAgaWYgKHRoaXMuX3Jvd0tleXMgPT0gbnVsbCkge1xuICAgICAgY29uc3Qgcm93Q291bnQgPSB0aGlzLl93YXNtLmNvbHVtbl9jb3VudCh0aGlzLl9zdG10KTtcbiAgICAgIHRoaXMuX3Jvd0tleXMgPSBbXTtcbiAgICAgIGZvciAobGV0IGkgPSAwOyBpIDwgcm93Q291bnQ7IGkrKykge1xuICAgICAgICB0aGlzLl9yb3dLZXlzLnB1c2goXG4gICAgICAgICAgZ2V0U3RyKHRoaXMuX3dhc20sIHRoaXMuX3dhc20uY29sdW1uX25hbWUodGhpcy5fc3RtdCwgaSkpLFxuICAgICAgICApO1xuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IG9iaiA9IHJvdy5yZWR1Y2U8Um93T2JqZWN0Pigob2JqLCB2YWwsIGlkeCkgPT4ge1xuICAgICAgb2JqW3RoaXMuX3Jvd0tleXMhW2lkeF1dID0gdmFsO1xuICAgICAgcmV0dXJuIG9iajtcbiAgICB9LCB7fSk7XG4gICAgcmV0dXJuIG9iaiBhcyBPO1xuICB9XG5cbiAgLyoqXG4gICAqIEJpbmRzIHRoZSBnaXZlbiBwYXJhbWV0ZXJzIHRvIHRoZSBxdWVyeVxuICAgKiBhbmQgcmV0dXJucyBhbiBpdGVyYXRvciBvdmVyIHJvd3MuXG4gICAqXG4gICAqIFVzaW5nIGFuIGl0ZXJhdG9yIGF2b2lkcyBsb2FkaW5nIGFsbCByZXR1cm5lZFxuICAgKiByb3dzIGludG8gbWVtb3J5IGFuZCBoZW5jZSBhbGxvd3MgdG8gcHJvY2VzcyBhIGxhcmdlXG4gICAqIG51bWJlciBvZiByb3dzLlxuICAgKlxuICAgKiAjIEV4YW1wbGU6XG4gICAqIGBgYHR5cGVzY3JpcHRcbiAgICogY29uc3QgcXVlcnkgPSBkYi5wcmVwYXJlUXVlcnk8W251bWJlciwgc3RyaW5nXT4oXCJTRUxFQ1QgaWQsIG5hbWUgRlJPTSBwZW9wbGVcIik7XG4gICAqIGZvciAoY29uc3QgW2lkLCBuYW1lXSBvZiBxdWVyeS5pdGVyKCkpIHtcbiAgICogICAvLyAuLi5cbiAgICogfVxuICAgKiBgYGBcbiAgICpcbiAgICogQ2FsbGluZyBgaXRlcmAgaW52YWxpZGF0ZXMgYW55IGl0ZXJhdG9ycyBwcmV2aW91c2x5IHJldHVybmVkXG4gICAqIGZyb20gdGhpcyBwcmVwYXJlZCBxdWVyeS4gVXNpbmcgYW4gaW52YWxpZGF0ZWQgaXRlcmF0b3IgaXMgYSBidWcuXG4gICAqXG4gICAqIFRvIGF2b2lkIFNRTCBpbmplY3Rpb24sIHVzZXItcHJvdmlkZWQgdmFsdWVzXG4gICAqIHNob3VsZCBhbHdheXMgYmUgcGFzc2VkIHRvIHRoZSBkYXRhYmFzZSB0aHJvdWdoXG4gICAqIGEgcXVlcnkgcGFyYW1ldGVyLlxuICAgKlxuICAgKiBTZWUgYFF1ZXJ5UGFyYW1ldGVyU2V0YCBmb3IgZG9jdW1lbnRhdGlvbiBvblxuICAgKiBob3cgdmFsdWVzIGNhbiBiZSBib3VuZCB0byBTUUwgc3RhdGVtZW50cy5cbiAgICpcbiAgICogU2VlIGBRdWVyeVBhcmFtZXRlcmAgZm9yIGRvY3VtZW50YXRpb24gb24gaG93XG4gICAqIHZhbHVlcyBhcmUgcmV0dXJuZWQgZnJvbSB0aGUgZGF0YWJhc2UuXG4gICAqL1xuICBpdGVyKHBhcmFtcz86IFApOiBSb3dzSXRlcmF0b3I8Uj4ge1xuICAgIHRoaXMuc3RhcnRRdWVyeShwYXJhbXMpO1xuICAgIHRoaXMuX3N0YXR1cyA9IHRoaXMuX3dhc20uc3RlcCh0aGlzLl9zdG10KTtcbiAgICBpZiAoXG4gICAgICB0aGlzLl9zdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVSb3cgJiYgdGhpcy5fc3RhdHVzICE9PSBTdGF0dXMuU3FsaXRlRG9uZVxuICAgICkge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKHRoaXMuX3dhc20sIHRoaXMuX3N0YXR1cyk7XG4gICAgfVxuICAgIHRoaXMuX2l0ZXJLdiA9IGZhbHNlO1xuICAgIHJldHVybiB0aGlzIGFzIFJvd3NJdGVyYXRvcjxSPjtcbiAgfVxuXG4gIC8qKlxuICAgKiBMaWtlIGBpdGVyYCBleGNlcHQgZWFjaCByb3cgaXMgcmV0dXJuZWRcbiAgICogYXMgYW4gb2JqZWN0IGNvbnRhaW5pbmcga2V5LXZhbHVlIHBhaXJzLlxuICAgKi9cbiAgaXRlckVudHJpZXMocGFyYW1zPzogUCk6IFJvd3NJdGVyYXRvcjxPPiB7XG4gICAgdGhpcy5pdGVyKHBhcmFtcyk7XG4gICAgdGhpcy5faXRlckt2ID0gdHJ1ZTtcbiAgICByZXR1cm4gdGhpcyBhcyBSb3dzSXRlcmF0b3I8Tz47XG4gIH1cblxuICAvKipcbiAgICogQGlnbm9yZVxuICAgKlxuICAgKiBJbXBsZW1lbnRzIHRoZSBpdGVyYWJsZSBwcm90b2NvbC4gSXQgaXNcbiAgICogYSBidWcgdG8gY2FsbCB0aGlzIG1ldGhvZCBkaXJlY3RseS5cbiAgICovXG4gIFtTeW1ib2wuaXRlcmF0b3JdKCk6IFJvd3NJdGVyYXRvcjxSIHwgTz4ge1xuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEBpZ25vcmVcbiAgICpcbiAgICogSW1wbGVtZW50cyB0aGUgaXRlcmF0b3IgcHJvdG9jb2wuIEl0IGlzXG4gICAqIGEgYnVnIHRvIGNhbGwgdGhpcyBtZXRob2QgZGlyZWN0bHkuXG4gICAqL1xuICBuZXh0KCk6IEl0ZXJhdG9yUmVzdWx0PFIgfCBPPiB7XG4gICAgaWYgKHRoaXMuX3N0YXR1cyA9PT0gU3RhdHVzLlNxbGl0ZVJvdykge1xuICAgICAgY29uc3QgdmFsdWUgPSB0aGlzLmdldFF1ZXJ5Um93KCk7XG4gICAgICB0aGlzLl9zdGF0dXMgPSB0aGlzLl93YXNtLnN0ZXAodGhpcy5fc3RtdCk7XG4gICAgICBpZiAodGhpcy5faXRlckt2KSB7XG4gICAgICAgIHJldHVybiB7IHZhbHVlOiB0aGlzLm1ha2VSb3dPYmplY3QodmFsdWUpLCBkb25lOiBmYWxzZSB9O1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmV0dXJuIHsgdmFsdWUsIGRvbmU6IGZhbHNlIH07XG4gICAgICB9XG4gICAgfSBlbHNlIGlmICh0aGlzLl9zdGF0dXMgPT09IFN0YXR1cy5TcWxpdGVEb25lKSB7XG4gICAgICByZXR1cm4geyB2YWx1ZTogbnVsbCwgZG9uZTogdHJ1ZSB9O1xuICAgIH0gZWxzZSB7XG4gICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSwgdGhpcy5fc3RhdHVzKTtcbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogQmluZHMgdGhlIGdpdmVuIHBhcmFtZXRlcnMgdG8gdGhlIHF1ZXJ5XG4gICAqIGFuZCByZXR1cm5zIGFuIGFycmF5IGNvbnRhaW5pbmcgYWxsIHJlc3VsdGluZ1xuICAgKiByb3dzLlxuICAgKlxuICAgKiBDYWxsaW5nIGBhbGxgIGludmFsaWRhdGVzIGFueSBpdGVyYXRvcnNcbiAgICogcHJldmlvdXNseSByZXR1cm5lZCBieSBjYWxscyB0byBgaXRlcmAuXG4gICAqIFVzaW5nIGFuIGludmFsaWRhdGVkIGl0ZXJhdG9yIGlzIGEgYnVnLlxuICAgKlxuICAgKiBUbyBhdm9pZCBTUUwgaW5qZWN0aW9uLCB1c2VyLXByb3ZpZGVkIHZhbHVlc1xuICAgKiBzaG91bGQgYWx3YXlzIGJlIHBhc3NlZCB0byB0aGUgZGF0YWJhc2UgdGhyb3VnaFxuICAgKiBhIHF1ZXJ5IHBhcmFtZXRlci5cbiAgICpcbiAgICogU2VlIGBRdWVyeVBhcmFtZXRlclNldGAgZm9yIGRvY3VtZW50YXRpb24gb25cbiAgICogaG93IHZhbHVlcyBjYW4gYmUgYm91bmQgdG8gU1FMIHN0YXRlbWVudHMuXG4gICAqXG4gICAqIFNlZSBgUXVlcnlQYXJhbWV0ZXJgIGZvciBkb2N1bWVudGF0aW9uIG9uIGhvd1xuICAgKiB2YWx1ZXMgYXJlIHJldHVybmVkIGZyb20gdGhlIGRhdGFiYXNlLlxuICAgKi9cbiAgYWxsKHBhcmFtcz86IFApOiBBcnJheTxSPiB7XG4gICAgdGhpcy5zdGFydFF1ZXJ5KHBhcmFtcyk7XG4gICAgY29uc3Qgcm93czogQXJyYXk8Uj4gPSBbXTtcbiAgICB0aGlzLl9zdGF0dXMgPSB0aGlzLl93YXNtLnN0ZXAodGhpcy5fc3RtdCk7XG4gICAgd2hpbGUgKHRoaXMuX3N0YXR1cyA9PT0gU3RhdHVzLlNxbGl0ZVJvdykge1xuICAgICAgcm93cy5wdXNoKHRoaXMuZ2V0UXVlcnlSb3coKSk7XG4gICAgICB0aGlzLl9zdGF0dXMgPSB0aGlzLl93YXNtLnN0ZXAodGhpcy5fc3RtdCk7XG4gICAgfVxuICAgIGlmICh0aGlzLl9zdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVEb25lKSB7XG4gICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSwgdGhpcy5fc3RhdHVzKTtcbiAgICB9XG4gICAgcmV0dXJuIHJvd3M7XG4gIH1cblxuICAvKipcbiAgICogTGlrZSBgYWxsYCBleGNlcHQgZWFjaCByb3cgaXMgcmV0dXJuZWRcbiAgICogYXMgYW4gb2JqZWN0IGNvbnRhaW5pbmcga2V5LXZhbHVlIHBhaXJzLlxuICAgKi9cbiAgYWxsRW50cmllcyhwYXJhbXM/OiBQKTogQXJyYXk8Tz4ge1xuICAgIHJldHVybiB0aGlzLmFsbChwYXJhbXMpLm1hcCgocm93KSA9PiB0aGlzLm1ha2VSb3dPYmplY3Qocm93KSk7XG4gIH1cblxuICAvKipcbiAgICogQmluZHMgdGhlIGdpdmVuIHBhcmFtZXRlcnMgdG8gdGhlIHF1ZXJ5IGFuZFxuICAgKiByZXR1cm5zIGV4YWN0bHkgb25lIHJvdy5cbiAgICpcbiAgICogSWYgdGhlIHF1ZXJ5IGRvZXMgbm90IHJldHVybiBleGFjdGx5IG9uZSByb3csXG4gICAqIHRoaXMgdGhyb3dzIGFuIGVycm9yLlxuICAgKlxuICAgKiBDYWxsaW5nIGBvbmVgIGludmFsaWRhdGVzIGFueSBpdGVyYXRvcnNcbiAgICogcHJldmlvdXNseSByZXR1cm5lZCBieSBjYWxscyB0byBgaXRlcmAuXG4gICAqIFVzaW5nIGFuIGludmFsaWRhdGVkIGl0ZXJhdG9yIGlzIGEgYnVnLlxuICAgKlxuICAgKiBUbyBhdm9pZCBTUUwgaW5qZWN0aW9uLCB1c2VyLXByb3ZpZGVkIHZhbHVlc1xuICAgKiBzaG91bGQgYWx3YXlzIGJlIHBhc3NlZCB0byB0aGUgZGF0YWJhc2UgdGhyb3VnaFxuICAgKiBhIHF1ZXJ5IHBhcmFtZXRlci5cbiAgICpcbiAgICogU2VlIGBRdWVyeVBhcmFtZXRlclNldGAgZm9yIGRvY3VtZW50YXRpb24gb25cbiAgICogaG93IHZhbHVlcyBjYW4gYmUgYm91bmQgdG8gU1FMIHN0YXRlbWVudHMuXG4gICAqXG4gICAqIFNlZSBgUXVlcnlQYXJhbWV0ZXJgIGZvciBkb2N1bWVudGF0aW9uIG9uIGhvd1xuICAgKiB2YWx1ZXMgYXJlIHJldHVybmVkIGZyb20gdGhlIGRhdGFiYXNlLlxuICAgKi9cbiAgb25lKHBhcmFtcz86IFApOiBSIHtcbiAgICB0aGlzLnN0YXJ0UXVlcnkocGFyYW1zKTtcblxuICAgIC8vIEdldCBmaXJzdCByb3dcbiAgICB0aGlzLl9zdGF0dXMgPSB0aGlzLl93YXNtLnN0ZXAodGhpcy5fc3RtdCk7XG4gICAgaWYgKHRoaXMuX3N0YXR1cyAhPT0gU3RhdHVzLlNxbGl0ZVJvdykge1xuICAgICAgaWYgKHRoaXMuX3N0YXR1cyA9PT0gU3RhdHVzLlNxbGl0ZURvbmUpIHtcbiAgICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKFwiVGhlIHF1ZXJ5IGRpZCBub3QgcmV0dXJuIGFueSByb3dzLlwiKTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcih0aGlzLl93YXNtLCB0aGlzLl9zdGF0dXMpO1xuICAgICAgfVxuICAgIH1cbiAgICBjb25zdCByb3cgPSB0aGlzLmdldFF1ZXJ5Um93KCk7XG5cbiAgICAvLyBFbnN1cmUgdGhlIHF1ZXJ5IG9ubHkgcmV0dXJucyBvbmUgcm93XG4gICAgdGhpcy5fc3RhdHVzID0gdGhpcy5fd2FzbS5zdGVwKHRoaXMuX3N0bXQpO1xuICAgIGlmICh0aGlzLl9zdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVEb25lKSB7XG4gICAgICBpZiAodGhpcy5fc3RhdHVzID09PSBTdGF0dXMuU3FsaXRlUm93KSB7XG4gICAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcihcIlRoZSBxdWVyeSByZXR1cm5lZCBtb3JlIHRoYW4gb25lIHJvdy5cIik7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSwgdGhpcy5fc3RhdHVzKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4gcm93O1xuICB9XG5cbiAgLyoqXG4gICAqIExpa2UgYG9uZWAgZXhjZXB0IHRoZSByb3cgaXMgcmV0dXJuZWRcbiAgICogYXMgYW4gb2JqZWN0IGNvbnRhaW5pbmcga2V5LXZhbHVlIHBhaXJzLlxuICAgKi9cbiAgb25lRW50cnkocGFyYW1zPzogUCk6IE8ge1xuICAgIHJldHVybiB0aGlzLm1ha2VSb3dPYmplY3QodGhpcy5vbmUocGFyYW1zKSk7XG4gIH1cblxuICAvKipcbiAgICogQmluZHMgdGhlIGdpdmVuIHBhcmFtZXRlcnMgdG8gdGhlIHF1ZXJ5IGFuZFxuICAgKiBleGVjdXRlcyB0aGUgcXVlcnksIGlnbm9yaW5nIGFueSByb3dzIHdoaWNoXG4gICAqIG1pZ2h0IGJlIHJldHVybmVkLlxuICAgKlxuICAgKiBVc2luZyB0aGlzIG1ldGhvZCBpcyBtb3JlIGVmZmljaWVudCB3aGVuIHRoZVxuICAgKiByb3dzIHJldHVybmVkIGJ5IGEgcXVlcnkgYXJlIG5vdCBuZWVkZWQgb3JcbiAgICogdGhlIHF1ZXJ5IGRvZXMgbm90IHJldHVybiBhbnkgcm93cy5cbiAgICpcbiAgICogQ2FsbGluZyBgZXhlY3V0ZWAgaW52YWxpZGF0ZXMgYW55IGl0ZXJhdG9yc1xuICAgKiBwcmV2aW91c2x5IHJldHVybmVkIGJ5IGNhbGxzIHRvIGBpdGVyYC5cbiAgICogVXNpbmcgYW4gaW52YWxpZGF0ZWQgaXRlcmF0b3IgaXMgYSBidWcuXG4gICAqXG4gICAqIFRvIGF2b2lkIFNRTCBpbmplY3Rpb24sIHVzZXItcHJvdmlkZWQgdmFsdWVzXG4gICAqIHNob3VsZCBhbHdheXMgYmUgcGFzc2VkIHRvIHRoZSBkYXRhYmFzZSB0aHJvdWdoXG4gICAqIGEgcXVlcnkgcGFyYW1ldGVyLlxuICAgKlxuICAgKiBTZWUgYFF1ZXJ5UGFyYW1ldGVyU2V0YCBmb3IgZG9jdW1lbnRhdGlvbiBvblxuICAgKiBob3cgdmFsdWVzIGNhbiBiZSBib3VuZCB0byBTUUwgc3RhdGVtZW50cy5cbiAgICovXG4gIGV4ZWN1dGUocGFyYW1zPzogUCkge1xuICAgIHRoaXMuc3RhcnRRdWVyeShwYXJhbXMpO1xuICAgIHRoaXMuX3N0YXR1cyA9IHRoaXMuX3dhc20uc3RlcCh0aGlzLl9zdG10KTtcbiAgICB3aGlsZSAodGhpcy5fc3RhdHVzID09PSBTdGF0dXMuU3FsaXRlUm93KSB7XG4gICAgICB0aGlzLl9zdGF0dXMgPSB0aGlzLl93YXNtLnN0ZXAodGhpcy5fc3RtdCk7XG4gICAgfVxuICAgIGlmICh0aGlzLl9zdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVEb25lKSB7XG4gICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSwgdGhpcy5fc3RhdHVzKTtcbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogQ2xvc2VzIHRoZSBwcmVwYXJlZCBxdWVyeS4gVGhpcyBtdXN0IGJlXG4gICAqIGNhbGxlZCBvbmNlIHRoZSBxdWVyeSBpcyBubyBsb25nZXIgbmVlZGVkXG4gICAqIHRvIGF2b2lkIGxlYWtpbmcgcmVzb3VyY2VzLlxuICAgKlxuICAgKiBBZnRlciBhIHByZXBhcmVkIHF1ZXJ5IGhhcyBiZWVuIGZpbmFsaXplZCxcbiAgICogdHJ5aW5nIHRvIGNhbGwgYGl0ZXJgLCBgYWxsYCwgYG9uZWAsXG4gICAqIGBleGVjdXRlYCwgb3IgYGNvbHVtbnNgLCBvciB1c2luZyBpdGVyYXRvcnMgd2hpY2ggd2hlcmVcbiAgICogcHJldmlvdXNseSBvYnRhaW5lZCBmcm9tIHRoZSBmaW5hbGl6ZWQgcXVlcnlcbiAgICogaXMgYSBidWcuXG4gICAqXG4gICAqIGBmaW5hbGl6ZWAgbWF5IHNhZmVseSBiZSBjYWxsZWQgbXVsdGlwbGVcbiAgICogdGltZXMuXG4gICAqL1xuICBmaW5hbGl6ZSgpIHtcbiAgICBpZiAoIXRoaXMuX2ZpbmFsaXplZCkge1xuICAgICAgdGhpcy5fd2FzbS5maW5hbGl6ZSh0aGlzLl9zdG10KTtcbiAgICAgIHRoaXMuX29wZW5TdGF0ZW1lbnRzLmRlbGV0ZSh0aGlzLl9zdG10KTtcbiAgICAgIHRoaXMuX2ZpbmFsaXplZCA9IHRydWU7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIFJldHVybnMgdGhlIGNvbHVtbiBuYW1lcyBmb3IgdGhlIHF1ZXJ5XG4gICAqIHJlc3VsdHMuXG4gICAqXG4gICAqIFRoaXMgbWV0aG9kIHJldHVybnMgYW4gYXJyYXkgb2Ygb2JqZWN0cyxcbiAgICogd2hlcmUgZWFjaCBvYmplY3QgaGFzIHRoZSBmb2xsb3dpbmcgcHJvcGVydGllczpcbiAgICpcbiAgICogfCBQcm9wZXJ0eSAgICAgfCBWYWx1ZSAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfFxuICAgKiB8LS0tLS0tLS0tLS0tLS18LS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS18XG4gICAqIHwgYG5hbWVgICAgICAgIHwgdGhlIHJlc3VsdCBvZiBgc3FsaXRlM19jb2x1bW5fbmFtZWAgICAgICAgIHxcbiAgICogfCBgb3JpZ2luTmFtZWAgfCB0aGUgcmVzdWx0IG9mIGBzcWxpdGUzX2NvbHVtbl9vcmlnaW5fbmFtZWAgfFxuICAgKiB8IGB0YWJsZU5hbWVgICB8IHRoZSByZXN1bHQgb2YgYHNxbGl0ZTNfY29sdW1uX3RhYmxlX25hbWVgICB8XG4gICAqL1xuICBjb2x1bW5zKCk6IEFycmF5PENvbHVtbk5hbWU+IHtcbiAgICBpZiAodGhpcy5fZmluYWxpemVkKSB7XG4gICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IoXG4gICAgICAgIFwiVW5hYmxlIHRvIHJldHJpZXZlIGNvbHVtbiBuYW1lcyBmcm9tIGZpbmFsaXplZCB0cmFuc2FjdGlvbi5cIixcbiAgICAgICk7XG4gICAgfVxuXG4gICAgY29uc3QgY29sdW1uQ291bnQgPSB0aGlzLl93YXNtLmNvbHVtbl9jb3VudCh0aGlzLl9zdG10KTtcbiAgICBjb25zdCBjb2x1bW5zOiBBcnJheTxDb2x1bW5OYW1lPiA9IFtdO1xuICAgIGZvciAobGV0IGkgPSAwOyBpIDwgY29sdW1uQ291bnQ7IGkrKykge1xuICAgICAgY29uc3QgbmFtZSA9IGdldFN0cihcbiAgICAgICAgdGhpcy5fd2FzbSxcbiAgICAgICAgdGhpcy5fd2FzbS5jb2x1bW5fbmFtZSh0aGlzLl9zdG10LCBpKSxcbiAgICAgICk7XG4gICAgICBjb25zdCBvcmlnaW5OYW1lID0gZ2V0U3RyKFxuICAgICAgICB0aGlzLl93YXNtLFxuICAgICAgICB0aGlzLl93YXNtLmNvbHVtbl9vcmlnaW5fbmFtZSh0aGlzLl9zdG10LCBpKSxcbiAgICAgICk7XG4gICAgICBjb25zdCB0YWJsZU5hbWUgPSBnZXRTdHIoXG4gICAgICAgIHRoaXMuX3dhc20sXG4gICAgICAgIHRoaXMuX3dhc20uY29sdW1uX3RhYmxlX25hbWUodGhpcy5fc3RtdCwgaSksXG4gICAgICApO1xuICAgICAgY29sdW1ucy5wdXNoKHsgbmFtZSwgb3JpZ2luTmFtZSwgdGFibGVOYW1lIH0pO1xuICAgIH1cbiAgICByZXR1cm4gY29sdW1ucztcbiAgfVxufVxuIl0sIm5h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