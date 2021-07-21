import { create_hash as createHash, digest_hash as digestHash, update_hash as updateHash, } from "./wasm.js";
import * as hex from "../../encoding/hex.ts";
import * as base64 from "../../encoding/base64.ts";
export class Hash {
    #hash;
    #digested;
    constructor(algorithm) {
        this.#hash = createHash(algorithm);
        this.#digested = false;
    }
    update(message) {
        let view;
        if (message instanceof Uint8Array) {
            view = message;
        }
        else if (typeof message === "string") {
            view = new TextEncoder().encode(message);
        }
        else if (ArrayBuffer.isView(message)) {
            view = new Uint8Array(message.buffer, message.byteOffset, message.byteLength);
        }
        else if (message instanceof ArrayBuffer) {
            view = new Uint8Array(message);
        }
        else {
            throw new Error("hash: `data` is invalid type");
        }
        const chunkSize = 65_536;
        for (let offset = 0; offset < view.byteLength; offset += chunkSize) {
            updateHash(this.#hash, new Uint8Array(view.buffer, view.byteOffset + offset, Math.min(chunkSize, view.byteLength - offset)));
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaGFzaC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImhhc2gudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBRUEsT0FBTyxFQUNMLFdBQVcsSUFBSSxVQUFVLEVBRXpCLFdBQVcsSUFBSSxVQUFVLEVBQ3pCLFdBQVcsSUFBSSxVQUFVLEdBQzFCLE1BQU0sV0FBVyxDQUFDO0FBRW5CLE9BQU8sS0FBSyxHQUFHLE1BQU0sdUJBQXVCLENBQUM7QUFDN0MsT0FBTyxLQUFLLE1BQU0sTUFBTSwwQkFBMEIsQ0FBQztBQUduRCxNQUFNLE9BQU8sSUFBSTtJQUNmLEtBQUssQ0FBVztJQUNoQixTQUFTLENBQVU7SUFFbkIsWUFBWSxTQUFpQjtRQUMzQixJQUFJLENBQUMsS0FBSyxHQUFHLFVBQVUsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNuQyxJQUFJLENBQUMsU0FBUyxHQUFHLEtBQUssQ0FBQztJQUN6QixDQUFDO0lBRUQsTUFBTSxDQUFDLE9BQWdCO1FBQ3JCLElBQUksSUFBZ0IsQ0FBQztRQUVyQixJQUFJLE9BQU8sWUFBWSxVQUFVLEVBQUU7WUFDakMsSUFBSSxHQUFHLE9BQU8sQ0FBQztTQUNoQjthQUFNLElBQUksT0FBTyxPQUFPLEtBQUssUUFBUSxFQUFFO1lBQ3RDLElBQUksR0FBRyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsQ0FBQztTQUMxQzthQUFNLElBQUksV0FBVyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsRUFBRTtZQUN0QyxJQUFJLEdBQUcsSUFBSSxVQUFVLENBQ25CLE9BQU8sQ0FBQyxNQUFNLEVBQ2QsT0FBTyxDQUFDLFVBQVUsRUFDbEIsT0FBTyxDQUFDLFVBQVUsQ0FDbkIsQ0FBQztTQUNIO2FBQU0sSUFBSSxPQUFPLFlBQVksV0FBVyxFQUFFO1lBQ3pDLElBQUksR0FBRyxJQUFJLFVBQVUsQ0FBQyxPQUFPLENBQUMsQ0FBQztTQUNoQzthQUFNO1lBQ0wsTUFBTSxJQUFJLEtBQUssQ0FBQyw4QkFBOEIsQ0FBQyxDQUFDO1NBQ2pEO1FBS0QsTUFBTSxTQUFTLEdBQUcsTUFBTSxDQUFDO1FBRXpCLEtBQ0UsSUFBSSxNQUFNLEdBQUcsQ0FBQyxFQUNkLE1BQU0sR0FBRyxJQUFJLENBQUMsVUFBVSxFQUN4QixNQUFNLElBQUksU0FBUyxFQUNuQjtZQUNBLFVBQVUsQ0FDUixJQUFJLENBQUMsS0FBSyxFQUNWLElBQUksVUFBVSxDQUNaLElBQUksQ0FBQyxNQUFNLEVBQ1gsSUFBSSxDQUFDLFVBQVUsR0FBRyxNQUFNLEVBQ3hCLElBQUksQ0FBQyxHQUFHLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxVQUFVLEdBQUcsTUFBTSxDQUFDLENBQzlDLENBQ0YsQ0FBQztTQUNIO1FBRUQsT0FBTyxJQUFJLENBQUM7SUFDZCxDQUFDO0lBR0QsTUFBTTtRQUNKLElBQUksSUFBSSxDQUFDLFNBQVM7WUFBRSxNQUFNLElBQUksS0FBSyxDQUFDLHdCQUF3QixDQUFDLENBQUM7UUFFOUQsSUFBSSxDQUFDLFNBQVMsR0FBRyxJQUFJLENBQUM7UUFDdEIsT0FBTyxVQUFVLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDO0lBQ2hDLENBQUM7SUFNRCxRQUFRLENBQUMsU0FBdUIsS0FBSztRQUNuQyxNQUFNLFNBQVMsR0FBRyxJQUFJLFVBQVUsQ0FBQyxJQUFJLENBQUMsTUFBTSxFQUFFLENBQUMsQ0FBQztRQUVoRCxRQUFRLE1BQU0sRUFBRTtZQUNkLEtBQUssS0FBSztnQkFDUixPQUFPLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxNQUFNLENBQUMsU0FBUyxDQUFDLENBQUMsQ0FBQztZQUN6RCxLQUFLLFFBQVE7Z0JBQ1gsT0FBTyxNQUFNLENBQUMsTUFBTSxDQUFDLFNBQVMsQ0FBQyxDQUFDO1lBQ2xDO2dCQUNFLE1BQU0sSUFBSSxLQUFLLENBQUMsc0JBQXNCLENBQUMsQ0FBQztTQUMzQztJQUNILENBQUM7Q0FDRiJ9