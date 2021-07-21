import { isArrayBuffer } from "../is-array-buffer/mod.ts";
import { Buffer } from "https://deno.land/std@0.97.0/node/buffer.ts";
export const fromArrayBuffer = (input, offset = 0, length = input.byteLength - offset) => {
    if (!isArrayBuffer(input)) {
        throw new TypeError(`The "input" argument must be ArrayBuffer. Received type ${typeof input} (${input})`);
    }
    return Buffer.from(input, offset, length);
};
export const fromString = (input, encoding) => {
    if (typeof input !== "string") {
        throw new TypeError(`The "input" argument must be of type string. Received type ${typeof input} (${input})`);
    }
    return encoding ? Buffer.from(input, encoding) : Buffer.from(input);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUMxRCxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sNkNBQTZDLENBQUM7QUFFckUsTUFBTSxDQUFDLE1BQU0sZUFBZSxHQUFHLENBQUMsS0FBa0IsRUFBRSxNQUFNLEdBQUcsQ0FBQyxFQUFFLFNBQWlCLEtBQUssQ0FBQyxVQUFVLEdBQUcsTUFBTSxFQUFVLEVBQUU7SUFDcEgsSUFBSSxDQUFDLGFBQWEsQ0FBQyxLQUFLLENBQUMsRUFBRTtRQUN6QixNQUFNLElBQUksU0FBUyxDQUFDLDJEQUEyRCxPQUFPLEtBQUssS0FBSyxLQUFLLEdBQUcsQ0FBQyxDQUFDO0tBQzNHO0lBRUQsT0FBTyxNQUFNLENBQUMsSUFBSSxDQUFDLEtBQUssRUFBRSxNQUFNLEVBQUUsTUFBTSxDQUFDLENBQUM7QUFDNUMsQ0FBQyxDQUFDO0FBSUYsTUFBTSxDQUFDLE1BQU0sVUFBVSxHQUFHLENBQUMsS0FBYSxFQUFFLFFBQXlCLEVBQVUsRUFBRTtJQUM3RSxJQUFJLE9BQU8sS0FBSyxLQUFLLFFBQVEsRUFBRTtRQUM3QixNQUFNLElBQUksU0FBUyxDQUFDLDhEQUE4RCxPQUFPLEtBQUssS0FBSyxLQUFLLEdBQUcsQ0FBQyxDQUFDO0tBQzlHO0lBRUQsT0FBTyxRQUFRLENBQUMsQ0FBQyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsS0FBSyxFQUFFLFFBQVEsQ0FBQyxDQUFDLENBQUMsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDO0FBQ3RFLENBQUMsQ0FBQyJ9