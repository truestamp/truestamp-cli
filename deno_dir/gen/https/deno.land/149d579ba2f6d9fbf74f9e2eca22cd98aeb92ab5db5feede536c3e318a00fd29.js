import { Cell } from "./cell.ts";
/**
 * Row representation.
 */ export class Row extends Array {
    options = {};
    /**
   * Create a new row. If cells is a row, all cells and options of the row will
   * be copied to the new row.
   * @param cells Cells or row.
   */ static from(cells) {
        const row = new this(...cells);
        if (cells instanceof Row) {
            row.options = {
                ...cells.options
            };
        }
        return row;
    }
    /** Clone row recursively with all options. */ clone() {
        const row = new Row(...this.map((cell)=>cell instanceof Cell ? cell.clone() : cell));
        row.options = {
            ...this.options
        };
        return row;
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
   * Align row content.
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
   */ /** Check if row has border. */ getBorder() {
        return this.options.border === true;
    }
    /** Check if row or any child cell has border. */ hasBorder() {
        return this.getBorder() || this.some((cell)=>cell instanceof Cell && cell.getBorder());
    }
    /** Get row alignment. */ getAlign() {
        return this.options.align ?? "left";
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvdGFibGUvcm93LnRzIl0sInNvdXJjZXNDb250ZW50IjpbImltcG9ydCB7IENlbGwsIERpcmVjdGlvbiwgSUNlbGwgfSBmcm9tIFwiLi9jZWxsLnRzXCI7XG5cbi8qKiBSb3cgdHlwZSAqL1xuZXhwb3J0IHR5cGUgSVJvdzxUIGV4dGVuZHMgSUNlbGwgPSBJQ2VsbD4gPSBUW10gfCBSb3c8VD47XG4vKiogSnNvbiByb3cuICovXG5leHBvcnQgdHlwZSBJRGF0YVJvdyA9IFJlY29yZDxzdHJpbmcsIHN0cmluZyB8IG51bWJlcj47XG5cbi8qKiBSb3cgb3B0aW9ucy4gKi9cbmV4cG9ydCBpbnRlcmZhY2UgSVJvd09wdGlvbnMge1xuICBpbmRlbnQ/OiBudW1iZXI7XG4gIGJvcmRlcj86IGJvb2xlYW47XG4gIGFsaWduPzogRGlyZWN0aW9uO1xufVxuXG4vKipcbiAqIFJvdyByZXByZXNlbnRhdGlvbi5cbiAqL1xuZXhwb3J0IGNsYXNzIFJvdzxUIGV4dGVuZHMgSUNlbGwgPSBJQ2VsbD4gZXh0ZW5kcyBBcnJheTxUPiB7XG4gIHByb3RlY3RlZCBvcHRpb25zOiBJUm93T3B0aW9ucyA9IHt9O1xuXG4gIC8qKlxuICAgKiBDcmVhdGUgYSBuZXcgcm93LiBJZiBjZWxscyBpcyBhIHJvdywgYWxsIGNlbGxzIGFuZCBvcHRpb25zIG9mIHRoZSByb3cgd2lsbFxuICAgKiBiZSBjb3BpZWQgdG8gdGhlIG5ldyByb3cuXG4gICAqIEBwYXJhbSBjZWxscyBDZWxscyBvciByb3cuXG4gICAqL1xuICBwdWJsaWMgc3RhdGljIGZyb208VCBleHRlbmRzIElDZWxsID0gSUNlbGw+KGNlbGxzOiBJUm93PFQ+KTogUm93PFQ+IHtcbiAgICBjb25zdCByb3cgPSBuZXcgdGhpcyguLi5jZWxscyk7XG4gICAgaWYgKGNlbGxzIGluc3RhbmNlb2YgUm93KSB7XG4gICAgICByb3cub3B0aW9ucyA9IHsgLi4uY2VsbHMub3B0aW9ucyB9O1xuICAgIH1cbiAgICByZXR1cm4gcm93O1xuICB9XG5cbiAgLyoqIENsb25lIHJvdyByZWN1cnNpdmVseSB3aXRoIGFsbCBvcHRpb25zLiAqL1xuICBwdWJsaWMgY2xvbmUoKTogUm93IHtcbiAgICBjb25zdCByb3cgPSBuZXcgUm93KFxuICAgICAgLi4udGhpcy5tYXAoKGNlbGw6IFQpID0+IGNlbGwgaW5zdGFuY2VvZiBDZWxsID8gY2VsbC5jbG9uZSgpIDogY2VsbCksXG4gICAgKTtcbiAgICByb3cub3B0aW9ucyA9IHsgLi4udGhpcy5vcHRpb25zIH07XG4gICAgcmV0dXJuIHJvdztcbiAgfVxuXG4gIC8qKlxuICAgKiBTZXR0ZXI6XG4gICAqL1xuXG4gIC8qKlxuICAgKiBFbmFibGUvZGlzYWJsZSBjZWxsIGJvcmRlci5cbiAgICogQHBhcmFtIGVuYWJsZSAgICBFbmFibGUvZGlzYWJsZSBjZWxsIGJvcmRlci5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBib3JkZXIoZW5hYmxlOiBib29sZWFuLCBvdmVycmlkZSA9IHRydWUpOiB0aGlzIHtcbiAgICBpZiAob3ZlcnJpZGUgfHwgdHlwZW9mIHRoaXMub3B0aW9ucy5ib3JkZXIgPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5ib3JkZXIgPSBlbmFibGU7XG4gICAgfVxuICAgIHJldHVybiB0aGlzO1xuICB9XG5cbiAgLyoqXG4gICAqIEFsaWduIHJvdyBjb250ZW50LlxuICAgKiBAcGFyYW0gZGlyZWN0aW9uIEFsaWduIGRpcmVjdGlvbi5cbiAgICogQHBhcmFtIG92ZXJyaWRlICBPdmVycmlkZSBleGlzdGluZyB2YWx1ZS5cbiAgICovXG4gIHB1YmxpYyBhbGlnbihkaXJlY3Rpb246IERpcmVjdGlvbiwgb3ZlcnJpZGUgPSB0cnVlKTogdGhpcyB7XG4gICAgaWYgKG92ZXJyaWRlIHx8IHR5cGVvZiB0aGlzLm9wdGlvbnMuYWxpZ24gPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHRoaXMub3B0aW9ucy5hbGlnbiA9IGRpcmVjdGlvbjtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXM7XG4gIH1cblxuICAvKipcbiAgICogR2V0dGVyOlxuICAgKi9cblxuICAvKiogQ2hlY2sgaWYgcm93IGhhcyBib3JkZXIuICovXG4gIHB1YmxpYyBnZXRCb3JkZXIoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9ucy5ib3JkZXIgPT09IHRydWU7XG4gIH1cblxuICAvKiogQ2hlY2sgaWYgcm93IG9yIGFueSBjaGlsZCBjZWxsIGhhcyBib3JkZXIuICovXG4gIHB1YmxpYyBoYXNCb3JkZXIoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuZ2V0Qm9yZGVyKCkgfHxcbiAgICAgIHRoaXMuc29tZSgoY2VsbCkgPT4gY2VsbCBpbnN0YW5jZW9mIENlbGwgJiYgY2VsbC5nZXRCb3JkZXIoKSk7XG4gIH1cblxuICAvKiogR2V0IHJvdyBhbGlnbm1lbnQuICovXG4gIHB1YmxpYyBnZXRBbGlnbigpOiBEaXJlY3Rpb24ge1xuICAgIHJldHVybiB0aGlzLm9wdGlvbnMuYWxpZ24gPz8gXCJsZWZ0XCI7XG4gIH1cbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUFTLElBQUksUUFBMEIsV0FBVyxDQUFDO0FBY25EOztHQUVHLENBQ0gsT0FBTyxNQUFNLEdBQUcsU0FBa0MsS0FBSztJQUNyRCxBQUFVLE9BQU8sR0FBZ0IsRUFBRSxDQUFDO0lBRXBDOzs7O0tBSUcsQ0FDSCxPQUFjLElBQUksQ0FBMEIsS0FBYyxFQUFVO1FBQ2xFLE1BQU0sR0FBRyxHQUFHLElBQUksSUFBSSxJQUFJLEtBQUssQ0FBQyxBQUFDO1FBQy9CLElBQUksS0FBSyxZQUFZLEdBQUcsRUFBRTtZQUN4QixHQUFHLENBQUMsT0FBTyxHQUFHO2dCQUFFLEdBQUcsS0FBSyxDQUFDLE9BQU87YUFBRSxDQUFDO1NBQ3BDO1FBQ0QsT0FBTyxHQUFHLENBQUM7S0FDWjtJQUVELDhDQUE4QyxDQUM5QyxBQUFPLEtBQUssR0FBUTtRQUNsQixNQUFNLEdBQUcsR0FBRyxJQUFJLEdBQUcsSUFDZCxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsSUFBTyxHQUFLLElBQUksWUFBWSxJQUFJLEdBQUcsSUFBSSxDQUFDLEtBQUssRUFBRSxHQUFHLElBQUksQ0FBQyxDQUNyRSxBQUFDO1FBQ0YsR0FBRyxDQUFDLE9BQU8sR0FBRztZQUFFLEdBQUcsSUFBSSxDQUFDLE9BQU87U0FBRSxDQUFDO1FBQ2xDLE9BQU8sR0FBRyxDQUFDO0tBQ1o7SUFFRDs7S0FFRyxDQUVIOzs7O0tBSUcsQ0FDSCxBQUFPLE1BQU0sQ0FBQyxNQUFlLEVBQUUsUUFBUSxHQUFHLElBQUksRUFBUTtRQUNwRCxJQUFJLFFBQVEsSUFBSSxPQUFPLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxLQUFLLFdBQVcsRUFBRTtZQUMxRCxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sR0FBRyxNQUFNLENBQUM7U0FDOUI7UUFDRCxPQUFPLElBQUksQ0FBQztLQUNiO0lBRUQ7Ozs7S0FJRyxDQUNILEFBQU8sS0FBSyxDQUFDLFNBQW9CLEVBQUUsUUFBUSxHQUFHLElBQUksRUFBUTtRQUN4RCxJQUFJLFFBQVEsSUFBSSxPQUFPLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxLQUFLLFdBQVcsRUFBRTtZQUN6RCxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssR0FBRyxTQUFTLENBQUM7U0FDaEM7UUFDRCxPQUFPLElBQUksQ0FBQztLQUNiO0lBRUQ7O0tBRUcsQ0FFSCwrQkFBK0IsQ0FDL0IsQUFBTyxTQUFTLEdBQVk7UUFDMUIsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sS0FBSyxJQUFJLENBQUM7S0FDckM7SUFFRCxpREFBaUQsQ0FDakQsQUFBTyxTQUFTLEdBQVk7UUFDMUIsT0FBTyxJQUFJLENBQUMsU0FBUyxFQUFFLElBQ3JCLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxJQUFJLEdBQUssSUFBSSxZQUFZLElBQUksSUFBSSxJQUFJLENBQUMsU0FBUyxFQUFFLENBQUMsQ0FBQztLQUNqRTtJQUVELHlCQUF5QixDQUN6QixBQUFPLFFBQVEsR0FBYztRQUMzQixPQUFPLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxJQUFJLE1BQU0sQ0FBQztLQUNyQztDQUNGIn0=