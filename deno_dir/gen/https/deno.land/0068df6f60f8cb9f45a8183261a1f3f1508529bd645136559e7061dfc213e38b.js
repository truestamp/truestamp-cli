import { create_hash as createHash, digest_hash as digestHash, update_hash as updateHash, } from "./wasm.js";
import * as hex from "../../encoding/hex.ts";
import * as base64 from "../../encoding/base64.ts";
const TYPE_ERROR_MSG = "hash: `data` is invalid type";
export class Hash {
    #hash;
    #digested;
    constructor(algorithm) {
        this.#hash = createHash(algorithm);
        this.#digested = false;
    }
    update(data) {
        let msg;
        if (typeof data === "string") {
            msg = new TextEncoder().encode(data);
        }
        else if (typeof data === "object") {
            if (data instanceof ArrayBuffer || ArrayBuffer.isView(data)) {
                msg = new Uint8Array(data);
            }
            else {
                throw new Error(TYPE_ERROR_MSG);
            }
        }
        else {
            throw new Error(TYPE_ERROR_MSG);
        }
        const CHUNK_SIZE = 65_536;
        for (let offset = 0; offset < msg.length; offset += CHUNK_SIZE) {
            updateHash(this.#hash, new Uint8Array(msg.buffer, offset, Math.min(CHUNK_SIZE, msg.length - offset)));
        }
        return this;
    }
    digest() {
        if (this.#digested)
            throw new Error("hash: already digested");
        this.#digested = true;
        return digestHash(this.#hash);
    }
    toString(format = "hex") {
        const finalized = new Uint8Array(this.digest());
        switch (format) {
            case "hex":
                return new TextDecoder().decode(hex.encode(finalized));
            case "base64":
                return base64.encode(finalized);
            default:
                throw new Error("hash: invalid format");
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaGFzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImhhc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBRUEsT0FBTyxFQUNMLFdBQVcsSUFBSSxVQUFVLEVBRXpCLFdBQVcsSUFBSSxVQUFVLEVBQ3pCLFdBQVcsSUFBSSxVQUFVLEdBQzFCLE1BQU0sV0FBVyxDQUFDO0FBRW5CLE9BQU8sS0FBSyxHQUFHLE1BQU0sdUJBQXVCLENBQUM7QUFDN0MsT0FBTyxLQUFLLE1BQU0sTUFBTSwwQkFBMEIsQ0FBQztBQUduRCxNQUFNLGNBQWMsR0FBRyw4QkFBOEIsQ0FBQztBQUV0RCxNQUFNLE9BQU8sSUFBSTtJQUNmLEtBQUssQ0FBVztJQUNoQixTQUFTLENBQVU7SUFFbkIsWUFBWSxTQUFpQjtRQUMzQixJQUFJLENBQUMsS0FBSyxHQUFHLFVBQVUsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNuQyxJQUFJLENBQUMsU0FBUyxHQUFHLEtBQUssQ0FBQztJQUN6QixDQUFDO0lBTUQsTUFBTSxDQUFDLElBQWE7UUFDbEIsSUFBSSxHQUFlLENBQUM7UUFFcEIsSUFBSSxPQUFPLElBQUksS0FBSyxRQUFRLEVBQUU7WUFDNUIsR0FBRyxHQUFHLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUFDLElBQWMsQ0FBQyxDQUFDO1NBQ2hEO2FBQU0sSUFBSSxPQUFPLElBQUksS0FBSyxRQUFRLEVBQUU7WUFDbkMsSUFBSSxJQUFJLFlBQVksV0FBVyxJQUFJLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLEVBQUU7Z0JBQzNELEdBQUcsR0FBRyxJQUFJLFVBQVUsQ0FBQyxJQUFJLENBQUMsQ0FBQzthQUM1QjtpQkFBTTtnQkFDTCxNQUFNLElBQUksS0FBSyxDQUFDLGNBQWMsQ0FBQyxDQUFDO2FBQ2pDO1NBQ0Y7YUFBTTtZQUNMLE1BQU0sSUFBSSxLQUFLLENBQUMsY0FBYyxDQUFDLENBQUM7U0FDakM7UUFJRCxNQUFNLFVBQVUsR0FBRyxNQUFNLENBQUM7UUFFMUIsS0FBSyxJQUFJLE1BQU0sR0FBRyxDQUFDLEVBQUUsTUFBTSxHQUFHLEdBQUcsQ0FBQyxNQUFNLEVBQUUsTUFBTSxJQUFJLFVBQVUsRUFBRTtZQUM5RCxVQUFVLENBQ1IsSUFBSSxDQUFDLEtBQUssRUFDVixJQUFJLFVBQVUsQ0FDWixHQUFHLENBQUMsTUFBTSxFQUNWLE1BQU0sRUFDTixJQUFJLENBQUMsR0FBRyxDQUFDLFVBQVUsRUFBRSxHQUFHLENBQUMsTUFBTSxHQUFHLE1BQU0sQ0FBQyxDQUMxQyxDQUNGLENBQUM7U0FDSDtRQUVELE9BQU8sSUFBSSxDQUFDO0lBQ2QsQ0FBQztJQUdELE1BQU07UUFDSixJQUFJLElBQUksQ0FBQyxTQUFTO1lBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQyx3QkFBd0IsQ0FBQyxDQUFDO1FBRTlELElBQUksQ0FBQyxTQUFTLEdBQUcsSUFBSSxDQUFDO1FBQ3RCLE9BQU8sVUFBVSxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQztJQUNoQyxDQUFDO0lBTUQsUUFBUSxDQUFDLFNBQXVCLEtBQUs7UUFDbkMsTUFBTSxTQUFTLEdBQUcsSUFBSSxVQUFVLENBQUMsSUFBSSxDQUFDLE1BQU0sRUFBRSxDQUFDLENBQUM7UUFFaEQsUUFBUSxNQUFNLEVBQUU7WUFDZCxLQUFLLEtBQUs7Z0JBQ1IsT0FBTyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxHQUFHLENBQUMsTUFBTSxDQUFDLFNBQVMsQ0FBQyxDQUFDLENBQUM7WUFDekQsS0FBSyxRQUFRO2dCQUNYLE9BQU8sTUFBTSxDQUFDLE1BQU0sQ0FBQyxTQUFTLENBQUMsQ0FBQztZQUNsQztnQkFDRSxNQUFNLElBQUksS0FBSyxDQUFDLHNCQUFzQixDQUFDLENBQUM7U0FDM0M7SUFDSCxDQUFDO0NBQ0YifQ==