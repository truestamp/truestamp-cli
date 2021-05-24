import { default as randomBytes } from "./_crypto/randomBytes.ts";
import { createHash as stdCreateHash, supportedAlgorithms, } from "../hash/mod.ts";
import { pbkdf2, pbkdf2Sync } from "./_crypto/pbkdf2.ts";
import { Buffer } from "./buffer.ts";
import { Transform } from "./stream.ts";
import { encodeToString as encodeToHexString } from "../encoding/hex.ts";
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
                return encodeToHexString(new Uint8Array(digest));
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY3J5cHRvLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiY3J5cHRvLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxPQUFPLElBQUksV0FBVyxFQUFFLE1BQU0sMEJBQTBCLENBQUM7QUFDbEUsT0FBTyxFQUNMLFVBQVUsSUFBSSxhQUFhLEVBRzNCLG1CQUFtQixHQUNwQixNQUFNLGdCQUFnQixDQUFDO0FBQ3hCLE9BQU8sRUFBRSxNQUFNLEVBQUUsVUFBVSxFQUFFLE1BQU0scUJBQXFCLENBQUM7QUFDekQsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUNyQyxPQUFPLEVBQUUsU0FBUyxFQUFFLE1BQU0sYUFBYSxDQUFDO0FBRXhDLE9BQU8sRUFBRSxjQUFjLElBQUksaUJBQWlCLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQVV6RSxNQUFNLE9BQU8sSUFBSyxTQUFRLFNBQVM7SUFDMUIsSUFBSSxDQUFTO0lBQ3BCLFlBQVksU0FBNkIsRUFBRSxLQUF3QjtRQUNqRSxLQUFLLENBQUM7WUFDSixTQUFTLENBQUMsS0FBYSxFQUFFLFNBQWlCLEVBQUUsUUFBb0I7Z0JBQzlELElBQUksQ0FBQyxNQUFNLENBQUMsS0FBSyxDQUFDLENBQUM7Z0JBQ25CLFFBQVEsRUFBRSxDQUFDO1lBQ2IsQ0FBQztZQUNELEtBQUssQ0FBQyxRQUFvQjtnQkFFeEIsSUFBSSxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsTUFBTSxFQUFFLENBQUMsQ0FBQztnQkFDekIsUUFBUSxFQUFFLENBQUM7WUFDYixDQUFDO1NBQ0YsQ0FBQyxDQUFDO1FBQ0gsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLElBQUksR0FBRyxhQUFhLENBQUMsU0FBUyxDQUFDLENBQUM7SUFDcEQsQ0FBQztJQVFELE1BQU0sQ0FBQyxJQUEwQixFQUFFLFNBQWtCO1FBQ25ELElBQUksT0FBTyxJQUFJLEtBQUssUUFBUSxFQUFFO1lBQzVCLElBQUksR0FBRyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsQ0FBQztZQUN0QyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsQ0FBQztTQUN4QjthQUFNO1lBQ0wsSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLENBQUM7U0FDeEI7UUFDRCxPQUFPLElBQUksQ0FBQztJQUNkLENBQUM7SUFTRCxNQUFNLENBQUMsUUFBaUI7UUFDdEIsTUFBTSxNQUFNLEdBQUcsSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLEVBQUUsQ0FBQztRQUNsQyxJQUFJLFFBQVEsS0FBSyxTQUFTLEVBQUU7WUFDMUIsT0FBTyxNQUFNLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDO1NBQzVCO1FBRUQsUUFBUSxRQUFRLEVBQUU7WUFDaEIsS0FBSyxLQUFLLENBQUMsQ0FBQztnQkFDVixPQUFPLGlCQUFpQixDQUFDLElBQUksVUFBVSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7YUFDbEQ7WUFFRCxPQUFPLENBQUMsQ0FBQztnQkFDUCxNQUFNLElBQUksS0FBSyxDQUNiLDREQUE0RCxRQUFRLEVBQUUsQ0FDdkUsQ0FBQzthQUNIO1NBQ0Y7SUFDSCxDQUFDO0NBQ0Y7QUFNRCxNQUFNLFVBQVUsVUFBVSxDQUN4QixTQUE2QixFQUM3QixJQUF1QjtJQUV2QixPQUFPLElBQUksSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLENBQUMsQ0FBQztBQUNuQyxDQUFDO0FBS0QsTUFBTSxVQUFVLFNBQVM7SUFDdkIsT0FBTyxtQkFBbUIsQ0FBQyxLQUFLLEVBQUUsQ0FBQztBQUNyQyxDQUFDO0FBRUQsZUFBZSxFQUFFLElBQUksRUFBRSxVQUFVLEVBQUUsU0FBUyxFQUFFLE1BQU0sRUFBRSxVQUFVLEVBQUUsV0FBVyxFQUFFLENBQUM7QUFDaEYsT0FBTyxFQUFFLE1BQU0sRUFBRSxVQUFVLEVBQUUsV0FBVyxFQUFFLENBQUMifQ==