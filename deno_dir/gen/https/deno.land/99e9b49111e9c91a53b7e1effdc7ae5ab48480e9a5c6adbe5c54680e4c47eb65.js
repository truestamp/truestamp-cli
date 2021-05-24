import { Buffer } from "../buffer.ts";
import Readable from "./readable.ts";
import { ERR_INVALID_ARG_TYPE, ERR_STREAM_NULL_VALUES } from "../_errors.ts";
export default function from(iterable, opts) {
    let iterator;
    if (typeof iterable === "string" || iterable instanceof Buffer) {
        return new Readable({
            objectMode: true,
            ...opts,
            read() {
                this.push(iterable);
                this.push(null);
            },
        });
    }
    if (Symbol.asyncIterator in iterable) {
        iterator = iterable[Symbol.asyncIterator]();
    }
    else if (Symbol.iterator in iterable) {
        iterator = iterable[Symbol.iterator]();
    }
    else {
        throw new ERR_INVALID_ARG_TYPE("iterable", ["Iterable"], iterable);
    }
    const readable = new Readable({
        objectMode: true,
        highWaterMark: 1,
        ...opts,
    });
    let reading = false;
    let needToClose = false;
    readable._read = function () {
        if (!reading) {
            reading = true;
            next();
        }
    };
    readable._destroy = function (error, cb) {
        if (needToClose) {
            needToClose = false;
            close().then(() => queueMicrotask(() => cb(error)), (e) => queueMicrotask(() => cb(error || e)));
        }
        else {
            cb(error);
        }
    };
    async function close() {
        if (typeof iterator.return === "function") {
            const { value } = await iterator.return();
            await value;
        }
    }
    async function next() {
        try {
            needToClose = false;
            const { value, done } = await iterator.next();
            needToClose = !done;
            if (done) {
                readable.push(null);
            }
            else if (readable.destroyed) {
                await close();
            }
            else {
                const res = await value;
                if (res === null) {
                    reading = false;
                    throw new ERR_STREAM_NULL_VALUES();
                }
                else if (readable.push(res)) {
                    next();
                }
                else {
                    reading = false;
                }
            }
        }
        catch (err) {
            readable.destroy(err);
        }
    }
    return readable;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImZyb20udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLGNBQWMsQ0FBQztBQUN0QyxPQUFPLFFBQVEsTUFBTSxlQUFlLENBQUM7QUFFckMsT0FBTyxFQUFFLG9CQUFvQixFQUFFLHNCQUFzQixFQUFFLE1BQU0sZUFBZSxDQUFDO0FBRTdFLE1BQU0sQ0FBQyxPQUFPLFVBQVUsSUFBSSxDQUUxQixRQUE0QyxFQUM1QyxJQUFzQjtJQUV0QixJQUFJLFFBSWtDLENBQUM7SUFDdkMsSUFBSSxPQUFPLFFBQVEsS0FBSyxRQUFRLElBQUksUUFBUSxZQUFZLE1BQU0sRUFBRTtRQUM5RCxPQUFPLElBQUksUUFBUSxDQUFDO1lBQ2xCLFVBQVUsRUFBRSxJQUFJO1lBQ2hCLEdBQUcsSUFBSTtZQUNQLElBQUk7Z0JBQ0YsSUFBSSxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsQ0FBQztnQkFDcEIsSUFBSSxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztZQUNsQixDQUFDO1NBQ0YsQ0FBQyxDQUFDO0tBQ0o7SUFFRCxJQUFJLE1BQU0sQ0FBQyxhQUFhLElBQUksUUFBUSxFQUFFO1FBRXBDLFFBQVEsR0FBSSxRQUErQixDQUFDLE1BQU0sQ0FBQyxhQUFhLENBQUMsRUFBRSxDQUFDO0tBQ3JFO1NBQU0sSUFBSSxNQUFNLENBQUMsUUFBUSxJQUFJLFFBQVEsRUFBRTtRQUV0QyxRQUFRLEdBQUksUUFBMEIsQ0FBQyxNQUFNLENBQUMsUUFBUSxDQUFDLEVBQUUsQ0FBQztLQUMzRDtTQUFNO1FBQ0wsTUFBTSxJQUFJLG9CQUFvQixDQUFDLFVBQVUsRUFBRSxDQUFDLFVBQVUsQ0FBQyxFQUFFLFFBQVEsQ0FBQyxDQUFDO0tBQ3BFO0lBRUQsTUFBTSxRQUFRLEdBQUcsSUFBSSxRQUFRLENBQUM7UUFDNUIsVUFBVSxFQUFFLElBQUk7UUFDaEIsYUFBYSxFQUFFLENBQUM7UUFDaEIsR0FBRyxJQUFJO0tBQ1IsQ0FBQyxDQUFDO0lBSUgsSUFBSSxPQUFPLEdBQUcsS0FBSyxDQUFDO0lBR3BCLElBQUksV0FBVyxHQUFHLEtBQUssQ0FBQztJQUV4QixRQUFRLENBQUMsS0FBSyxHQUFHO1FBQ2YsSUFBSSxDQUFDLE9BQU8sRUFBRTtZQUNaLE9BQU8sR0FBRyxJQUFJLENBQUM7WUFDZixJQUFJLEVBQUUsQ0FBQztTQUNSO0lBQ0gsQ0FBQyxDQUFDO0lBRUYsUUFBUSxDQUFDLFFBQVEsR0FBRyxVQUFVLEtBQUssRUFBRSxFQUFFO1FBQ3JDLElBQUksV0FBVyxFQUFFO1lBQ2YsV0FBVyxHQUFHLEtBQUssQ0FBQztZQUNwQixLQUFLLEVBQUUsQ0FBQyxJQUFJLENBQ1YsR0FBRyxFQUFFLENBQUMsY0FBYyxDQUFDLEdBQUcsRUFBRSxDQUFDLEVBQUUsQ0FBQyxLQUFLLENBQUMsQ0FBQyxFQUNyQyxDQUFDLENBQUMsRUFBRSxFQUFFLENBQUMsY0FBYyxDQUFDLEdBQUcsRUFBRSxDQUFDLEVBQUUsQ0FBQyxLQUFLLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FDNUMsQ0FBQztTQUNIO2FBQU07WUFDTCxFQUFFLENBQUMsS0FBSyxDQUFDLENBQUM7U0FDWDtJQUNILENBQUMsQ0FBQztJQUVGLEtBQUssVUFBVSxLQUFLO1FBQ2xCLElBQUksT0FBTyxRQUFRLENBQUMsTUFBTSxLQUFLLFVBQVUsRUFBRTtZQUN6QyxNQUFNLEVBQUUsS0FBSyxFQUFFLEdBQUcsTUFBTSxRQUFRLENBQUMsTUFBTSxFQUFFLENBQUM7WUFDMUMsTUFBTSxLQUFLLENBQUM7U0FDYjtJQUNILENBQUM7SUFFRCxLQUFLLFVBQVUsSUFBSTtRQUNqQixJQUFJO1lBQ0YsV0FBVyxHQUFHLEtBQUssQ0FBQztZQUNwQixNQUFNLEVBQUUsS0FBSyxFQUFFLElBQUksRUFBRSxHQUFHLE1BQU0sUUFBUSxDQUFDLElBQUksRUFBRSxDQUFDO1lBQzlDLFdBQVcsR0FBRyxDQUFDLElBQUksQ0FBQztZQUNwQixJQUFJLElBQUksRUFBRTtnQkFDUixRQUFRLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO2FBQ3JCO2lCQUFNLElBQUksUUFBUSxDQUFDLFNBQVMsRUFBRTtnQkFDN0IsTUFBTSxLQUFLLEVBQUUsQ0FBQzthQUNmO2lCQUFNO2dCQUNMLE1BQU0sR0FBRyxHQUFHLE1BQU0sS0FBSyxDQUFDO2dCQUN4QixJQUFJLEdBQUcsS0FBSyxJQUFJLEVBQUU7b0JBQ2hCLE9BQU8sR0FBRyxLQUFLLENBQUM7b0JBQ2hCLE1BQU0sSUFBSSxzQkFBc0IsRUFBRSxDQUFDO2lCQUNwQztxQkFBTSxJQUFJLFFBQVEsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLEVBQUU7b0JBQzdCLElBQUksRUFBRSxDQUFDO2lCQUNSO3FCQUFNO29CQUNMLE9BQU8sR0FBRyxLQUFLLENBQUM7aUJBQ2pCO2FBQ0Y7U0FDRjtRQUFDLE9BQU8sR0FBRyxFQUFFO1lBQ1osUUFBUSxDQUFDLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQztTQUN2QjtJQUNILENBQUM7SUFDRCxPQUFPLFFBQVEsQ0FBQztBQUNsQixDQUFDIn0=