import Duplex from "./duplex.ts";
import { ERR_METHOD_NOT_IMPLEMENTED } from "../_errors.ts";
const kCallback = Symbol("kCallback");
export default class Transform extends Duplex {
    [kCallback];
    _flush;
    constructor(options) {
        super(options);
        this._readableState.sync = false;
        this[kCallback] = null;
        if (options) {
            if (typeof options.transform === "function") {
                this._transform = options.transform;
            }
            if (typeof options.flush === "function") {
                this._flush = options.flush;
            }
        }
        this.on("prefinish", function () {
            if (typeof this._flush === "function" && !this.destroyed) {
                this._flush((er, data) => {
                    if (er) {
                        this.destroy(er);
                        return;
                    }
                    if (data != null) {
                        this.push(data);
                    }
                    this.push(null);
                });
            }
            else {
                this.push(null);
            }
        });
    }
    _read = () => {
        if (this[kCallback]) {
            const callback = this[kCallback];
            this[kCallback] = null;
            callback();
        }
    };
    _transform(_chunk, _encoding, _callback) {
        throw new ERR_METHOD_NOT_IMPLEMENTED("_transform()");
    }
    _write = (chunk, encoding, callback) => {
        const rState = this._readableState;
        const wState = this._writableState;
        const length = rState.length;
        this._transform(chunk, encoding, (err, val) => {
            if (err) {
                callback(err);
                return;
            }
            if (val != null) {
                this.push(val);
            }
            if (wState.ended ||
                length === rState.length ||
                rState.length < rState.highWaterMark ||
                rState.length === 0) {
                callback();
            }
            else {
                this[kCallback] = callback;
            }
        });
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHJhbnNmb3JtLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidHJhbnNmb3JtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sTUFBTSxNQUFNLGFBQWEsQ0FBQztBQUdqQyxPQUFPLEVBQUUsMEJBQTBCLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFFM0QsTUFBTSxTQUFTLEdBQUcsTUFBTSxDQUFDLFdBQVcsQ0FBQyxDQUFDO0FBbUN0QyxNQUFNLENBQUMsT0FBTyxPQUFPLFNBQVUsU0FBUSxNQUFNO0lBQzNDLENBQUMsU0FBUyxDQUFDLENBQTBDO0lBQ3JELE1BQU0sQ0FBa0I7SUFFeEIsWUFBWSxPQUEwQjtRQUNwQyxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7UUFDZixJQUFJLENBQUMsY0FBYyxDQUFDLElBQUksR0FBRyxLQUFLLENBQUM7UUFFakMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxHQUFHLElBQUksQ0FBQztRQUV2QixJQUFJLE9BQU8sRUFBRTtZQUNYLElBQUksT0FBTyxPQUFPLENBQUMsU0FBUyxLQUFLLFVBQVUsRUFBRTtnQkFDM0MsSUFBSSxDQUFDLFVBQVUsR0FBRyxPQUFPLENBQUMsU0FBUyxDQUFDO2FBQ3JDO1lBRUQsSUFBSSxPQUFPLE9BQU8sQ0FBQyxLQUFLLEtBQUssVUFBVSxFQUFFO2dCQUN2QyxJQUFJLENBQUMsTUFBTSxHQUFHLE9BQU8sQ0FBQyxLQUFLLENBQUM7YUFDN0I7U0FDRjtRQUVELElBQUksQ0FBQyxFQUFFLENBQUMsV0FBVyxFQUFFO1lBQ25CLElBQUksT0FBTyxJQUFJLENBQUMsTUFBTSxLQUFLLFVBQVUsSUFBSSxDQUFDLElBQUksQ0FBQyxTQUFTLEVBQUU7Z0JBQ3hELElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxFQUFFLEVBQUUsSUFBSSxFQUFFLEVBQUU7b0JBQ3ZCLElBQUksRUFBRSxFQUFFO3dCQUNOLElBQUksQ0FBQyxPQUFPLENBQUMsRUFBRSxDQUFDLENBQUM7d0JBQ2pCLE9BQU87cUJBQ1I7b0JBRUQsSUFBSSxJQUFJLElBQUksSUFBSSxFQUFFO3dCQUNoQixJQUFJLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO3FCQUNqQjtvQkFDRCxJQUFJLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO2dCQUNsQixDQUFDLENBQUMsQ0FBQzthQUNKO2lCQUFNO2dCQUNMLElBQUksQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7YUFDakI7UUFDSCxDQUFDLENBQUMsQ0FBQztJQUNMLENBQUM7SUFFRCxLQUFLLEdBQUcsR0FBRyxFQUFFO1FBQ1gsSUFBSSxJQUFJLENBQUMsU0FBUyxDQUFDLEVBQUU7WUFDbkIsTUFBTSxRQUFRLEdBQUcsSUFBSSxDQUFDLFNBQVMsQ0FBbUMsQ0FBQztZQUNuRSxJQUFJLENBQUMsU0FBUyxDQUFDLEdBQUcsSUFBSSxDQUFDO1lBQ3ZCLFFBQVEsRUFBRSxDQUFDO1NBQ1o7SUFDSCxDQUFDLENBQUM7SUFFRixVQUFVLENBRVIsTUFBVyxFQUNYLFNBQWlCLEVBRWpCLFNBQXFEO1FBRXJELE1BQU0sSUFBSSwwQkFBMEIsQ0FBQyxjQUFjLENBQUMsQ0FBQztJQUN2RCxDQUFDO0lBRUQsTUFBTSxHQUFHLENBRVAsS0FBVSxFQUNWLFFBQWdCLEVBQ2hCLFFBQXdDLEVBQ3hDLEVBQUU7UUFDRixNQUFNLE1BQU0sR0FBRyxJQUFJLENBQUMsY0FBYyxDQUFDO1FBQ25DLE1BQU0sTUFBTSxHQUFHLElBQUksQ0FBQyxjQUFjLENBQUM7UUFDbkMsTUFBTSxNQUFNLEdBQUcsTUFBTSxDQUFDLE1BQU0sQ0FBQztRQUU3QixJQUFJLENBQUMsVUFBVSxDQUFDLEtBQUssRUFBRSxRQUFRLEVBQUUsQ0FBQyxHQUFHLEVBQUUsR0FBRyxFQUFFLEVBQUU7WUFDNUMsSUFBSSxHQUFHLEVBQUU7Z0JBQ1AsUUFBUSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2dCQUNkLE9BQU87YUFDUjtZQUVELElBQUksR0FBRyxJQUFJLElBQUksRUFBRTtnQkFDZixJQUFJLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQ2hCO1lBRUQsSUFDRSxNQUFNLENBQUMsS0FBSztnQkFDWixNQUFNLEtBQUssTUFBTSxDQUFDLE1BQU07Z0JBQ3hCLE1BQU0sQ0FBQyxNQUFNLEdBQUcsTUFBTSxDQUFDLGFBQWE7Z0JBQ3BDLE1BQU0sQ0FBQyxNQUFNLEtBQUssQ0FBQyxFQUNuQjtnQkFDQSxRQUFRLEVBQUUsQ0FBQzthQUNaO2lCQUFNO2dCQUNMLElBQUksQ0FBQyxTQUFTLENBQUMsR0FBRyxRQUFRLENBQUM7YUFDNUI7UUFDSCxDQUFDLENBQUMsQ0FBQztJQUNMLENBQUMsQ0FBQztDQUNIIn0=