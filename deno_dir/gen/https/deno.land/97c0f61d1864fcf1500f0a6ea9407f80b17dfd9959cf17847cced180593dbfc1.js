import { default as randomBytes } from "./_crypto/randomBytes.ts";
import { createHash as stdCreateHash, supportedAlgorithms, } from "../hash/mod.ts";
import { pbkdf2, pbkdf2Sync } from "./_crypto/pbkdf2.ts";
import { Buffer } from "./buffer.ts";
import { Transform } from "./stream.ts";
import { encode as encodeToHex } from "../encoding/hex.ts";
export class Hash extends Transform {
    hash;
    constructor(algorithm, _opts) {
        super({
            transform(chunk, _encoding, callback) {
                hash.update(chunk);
                callback();
            },
            flush(callback) {
                this.push(hash.digest());
                callback();
            },
        });
        const hash = this.hash = stdCreateHash(algorithm);
    }
    update(data, _encoding) {
        if (typeof data === "string") {
            data = new TextEncoder().encode(data);
            this.hash.update(data);
        }
        else {
            this.hash.update(data);
        }
        return this;
    }
    digest(encoding) {
        const digest = this.hash.digest();
        if (encoding === undefined) {
            return Buffer.from(digest);
        }
        switch (encoding) {
            case "hex": {
                return new TextDecoder().decode(encodeToHex(new Uint8Array(digest)));
            }
            default: {
                throw new Error(`The output encoding for hash digest is not impelemented: ${encoding}`);
            }
        }
    }
}
export function createHash(algorithm, opts) {
    return new Hash(algorithm, opts);
}
export function getHashes() {
    return supportedAlgorithms.slice();
}
export default { Hash, createHash, getHashes, pbkdf2, pbkdf2Sync, randomBytes };
export { pbkdf2, pbkdf2Sync, randomBytes };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY3J5cHRvLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiY3J5cHRvLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxPQUFPLElBQUksV0FBVyxFQUFFLE1BQU0sMEJBQTBCLENBQUM7QUFDbEUsT0FBTyxFQUNMLFVBQVUsSUFBSSxhQUFhLEVBRzNCLG1CQUFtQixHQUNwQixNQUFNLGdCQUFnQixDQUFDO0FBQ3hCLE9BQU8sRUFBRSxNQUFNLEVBQUUsVUFBVSxFQUFFLE1BQU0scUJBQXFCLENBQUM7QUFDekQsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUNyQyxPQUFPLEVBQUUsU0FBUyxFQUFFLE1BQU0sYUFBYSxDQUFDO0FBRXhDLE9BQU8sRUFBRSxNQUFNLElBQUksV0FBVyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFVM0QsTUFBTSxPQUFPLElBQUssU0FBUSxTQUFTO0lBQzFCLElBQUksQ0FBUztJQUNwQixZQUFZLFNBQTZCLEVBQUUsS0FBd0I7UUFDakUsS0FBSyxDQUFDO1lBQ0osU0FBUyxDQUFDLEtBQWEsRUFBRSxTQUFpQixFQUFFLFFBQW9CO2dCQUM5RCxJQUFJLENBQUMsTUFBTSxDQUFDLEtBQUssQ0FBQyxDQUFDO2dCQUNuQixRQUFRLEVBQUUsQ0FBQztZQUNiLENBQUM7WUFDRCxLQUFLLENBQUMsUUFBb0I7Z0JBQ3hCLElBQUksQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sRUFBRSxDQUFDLENBQUM7Z0JBQ3pCLFFBQVEsRUFBRSxDQUFDO1lBQ2IsQ0FBQztTQUNGLENBQUMsQ0FBQztRQUNILE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxJQUFJLEdBQUcsYUFBYSxDQUFDLFNBQVMsQ0FBQyxDQUFDO0lBQ3BELENBQUM7SUFRRCxNQUFNLENBQUMsSUFBMEIsRUFBRSxTQUFrQjtRQUNuRCxJQUFJLE9BQU8sSUFBSSxLQUFLLFFBQVEsRUFBRTtZQUM1QixJQUFJLEdBQUcsSUFBSSxXQUFXLEVBQUUsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLENBQUM7WUFDdEMsSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLENBQUM7U0FDeEI7YUFBTTtZQUNMLElBQUksQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxDQUFDO1NBQ3hCO1FBQ0QsT0FBTyxJQUFJLENBQUM7SUFDZCxDQUFDO0lBU0QsTUFBTSxDQUFDLFFBQWlCO1FBQ3RCLE1BQU0sTUFBTSxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUMsTUFBTSxFQUFFLENBQUM7UUFDbEMsSUFBSSxRQUFRLEtBQUssU0FBUyxFQUFFO1lBQzFCLE9BQU8sTUFBTSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQztTQUM1QjtRQUVELFFBQVEsUUFBUSxFQUFFO1lBQ2hCLEtBQUssS0FBSyxDQUFDLENBQUM7Z0JBQ1YsT0FBTyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxXQUFXLENBQUMsSUFBSSxVQUFVLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQyxDQUFDO2FBQ3RFO1lBRUQsT0FBTyxDQUFDLENBQUM7Z0JBQ1AsTUFBTSxJQUFJLEtBQUssQ0FDYiw0REFBNEQsUUFBUSxFQUFFLENBQ3ZFLENBQUM7YUFDSDtTQUNGO0lBQ0gsQ0FBQztDQUNGO0FBTUQsTUFBTSxVQUFVLFVBQVUsQ0FDeEIsU0FBNkIsRUFDN0IsSUFBdUI7SUFFdkIsT0FBTyxJQUFJLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxDQUFDLENBQUM7QUFDbkMsQ0FBQztBQUtELE1BQU0sVUFBVSxTQUFTO0lBQ3ZCLE9BQU8sbUJBQW1CLENBQUMsS0FBSyxFQUFFLENBQUM7QUFDckMsQ0FBQztBQUVELGVBQWUsRUFBRSxJQUFJLEVBQUUsVUFBVSxFQUFFLFNBQVMsRUFBRSxNQUFNLEVBQUUsVUFBVSxFQUFFLFdBQVcsRUFBRSxDQUFDO0FBQ2hGLE9BQU8sRUFBRSxNQUFNLEVBQUUsVUFBVSxFQUFFLFdBQVcsRUFBRSxDQUFDIn0=