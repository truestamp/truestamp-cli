import { Buffer } from "../buffer.ts";
import EventEmitter from "../events.ts";
import { types } from "../util.ts";
class Stream extends EventEmitter {
    constructor() {
        super();
    }
    static _isUint8Array = types.isUint8Array;
    static _uint8ArrayToBuffer = (chunk) => Buffer.from(chunk);
    pipe(dest, options) {
        const source = this;
        if (options?.end ?? true) {
            source.on("end", onend);
            source.on("close", onclose);
        }
        let didOnEnd = false;
        function onend() {
            if (didOnEnd)
                return;
            didOnEnd = true;
            dest.end();
        }
        function onclose() {
            if (didOnEnd)
                return;
            didOnEnd = true;
            if (typeof dest.destroy === "function")
                dest.destroy();
        }
        function onerror(er) {
            cleanup();
            if (this.listenerCount("error") === 0) {
                throw er;
            }
        }
        source.on("error", onerror);
        dest.on("error", onerror);
        function cleanup() {
            source.removeListener("end", onend);
            source.removeListener("close", onclose);
            source.removeListener("error", onerror);
            dest.removeListener("error", onerror);
            source.removeListener("end", cleanup);
            source.removeListener("close", cleanup);
            dest.removeListener("close", cleanup);
        }
        source.on("end", cleanup);
        source.on("close", cleanup);
        dest.on("close", cleanup);
        dest.emit("pipe", source);
        return dest;
    }
}
export default Stream;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic3RyZWFtLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsic3RyZWFtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxNQUFNLEVBQUUsTUFBTSxjQUFjLENBQUM7QUFDdEMsT0FBTyxZQUFZLE1BQU0sY0FBYyxDQUFDO0FBR3hDLE9BQU8sRUFBRSxLQUFLLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFFbkMsTUFBTSxNQUFPLFNBQVEsWUFBWTtJQUMvQjtRQUNFLEtBQUssRUFBRSxDQUFDO0lBQ1YsQ0FBQztJQUVELE1BQU0sQ0FBQyxhQUFhLEdBQUcsS0FBSyxDQUFDLFlBQVksQ0FBQztJQUMxQyxNQUFNLENBQUMsbUJBQW1CLEdBQUcsQ0FBQyxLQUFpQixFQUFFLEVBQUUsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDO0lBRXZFLElBQUksQ0FBQyxJQUF5QixFQUFFLE9BQTJCO1FBRXpELE1BQU0sTUFBTSxHQUFHLElBQUksQ0FBQztRQU9wQixJQUFJLE9BQU8sRUFBRSxHQUFHLElBQUksSUFBSSxFQUFFO1lBQ3hCLE1BQU0sQ0FBQyxFQUFFLENBQUMsS0FBSyxFQUFFLEtBQUssQ0FBQyxDQUFDO1lBQ3hCLE1BQU0sQ0FBQyxFQUFFLENBQUMsT0FBTyxFQUFFLE9BQU8sQ0FBQyxDQUFDO1NBQzdCO1FBRUQsSUFBSSxRQUFRLEdBQUcsS0FBSyxDQUFDO1FBQ3JCLFNBQVMsS0FBSztZQUNaLElBQUksUUFBUTtnQkFBRSxPQUFPO1lBQ3JCLFFBQVEsR0FBRyxJQUFJLENBQUM7WUFHZixJQUFpQixDQUFDLEdBQUcsRUFBRSxDQUFDO1FBQzNCLENBQUM7UUFFRCxTQUFTLE9BQU87WUFDZCxJQUFJLFFBQVE7Z0JBQUUsT0FBTztZQUNyQixRQUFRLEdBQUcsSUFBSSxDQUFDO1lBRWhCLElBQUksT0FBTyxJQUFJLENBQUMsT0FBTyxLQUFLLFVBQVU7Z0JBQUUsSUFBSSxDQUFDLE9BQU8sRUFBRSxDQUFDO1FBQ3pELENBQUM7UUFHRCxTQUFTLE9BQU8sQ0FBZSxFQUFTO1lBQ3RDLE9BQU8sRUFBRSxDQUFDO1lBQ1YsSUFBSSxJQUFJLENBQUMsYUFBYSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsRUFBRTtnQkFDckMsTUFBTSxFQUFFLENBQUM7YUFDVjtRQUNILENBQUM7UUFFRCxNQUFNLENBQUMsRUFBRSxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztRQUM1QixJQUFJLENBQUMsRUFBRSxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztRQUcxQixTQUFTLE9BQU87WUFDZCxNQUFNLENBQUMsY0FBYyxDQUFDLEtBQUssRUFBRSxLQUFLLENBQUMsQ0FBQztZQUNwQyxNQUFNLENBQUMsY0FBYyxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztZQUV4QyxNQUFNLENBQUMsY0FBYyxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztZQUN4QyxJQUFJLENBQUMsY0FBYyxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztZQUV0QyxNQUFNLENBQUMsY0FBYyxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztZQUN0QyxNQUFNLENBQUMsY0FBYyxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztZQUV4QyxJQUFJLENBQUMsY0FBYyxDQUFDLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztRQUN4QyxDQUFDO1FBRUQsTUFBTSxDQUFDLEVBQUUsQ0FBQyxLQUFLLEVBQUUsT0FBTyxDQUFDLENBQUM7UUFDMUIsTUFBTSxDQUFDLEVBQUUsQ0FBQyxPQUFPLEVBQUUsT0FBTyxDQUFDLENBQUM7UUFFNUIsSUFBSSxDQUFDLEVBQUUsQ0FBQyxPQUFPLEVBQUUsT0FBTyxDQUFDLENBQUM7UUFDMUIsSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLEVBQUUsTUFBTSxDQUFDLENBQUM7UUFFMUIsT0FBTyxJQUFJLENBQUM7SUFDZCxDQUFDOztBQUdILGVBQWUsTUFBTSxDQUFDIn0=