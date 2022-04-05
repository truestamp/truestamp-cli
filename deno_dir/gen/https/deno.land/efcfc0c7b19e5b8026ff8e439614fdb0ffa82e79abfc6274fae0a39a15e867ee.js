import { SqliteError } from "./error.ts";
export function setStr(wasm, str, closure) {
    const bytes = new TextEncoder().encode(str);
    const ptr = wasm.malloc(bytes.length + 1);
    if (ptr === 0) {
        throw new SqliteError("Out of memory.");
    }
    const mem = new Uint8Array(wasm.memory.buffer, ptr, bytes.length + 1);
    mem.set(bytes);
    mem[bytes.length] = 0;
    try {
        const result = closure(ptr);
        wasm.free(ptr);
        return result;
    }
    catch (error) {
        wasm.free(ptr);
        throw error;
    }
}
export function setArr(wasm, arr, closure) {
    const ptr = wasm.malloc(arr.length);
    if (ptr === 0) {
        throw new SqliteError("Out of memory.");
    }
    const mem = new Uint8Array(wasm.memory.buffer, ptr, arr.length);
    mem.set(arr);
    try {
        const result = closure(ptr);
        wasm.free(ptr);
        return result;
    }
    catch (error) {
        wasm.free(ptr);
        throw error;
    }
}
export function getStr(wasm, ptr) {
    const len = wasm.str_len(ptr);
    const bytes = new Uint8Array(wasm.memory.buffer, ptr, len);
    if (len > 16) {
        return new TextDecoder().decode(bytes);
    }
    else {
        let str = "";
        let idx = 0;
        while (idx < len) {
            let u0 = bytes[idx++];
            if (!(u0 & 0x80)) {
                str += String.fromCharCode(u0);
                continue;
            }
            const u1 = bytes[idx++] & 63;
            if ((u0 & 0xE0) == 0xC0) {
                str += String.fromCharCode(((u0 & 31) << 6) | u1);
                continue;
            }
            const u2 = bytes[idx++] & 63;
            if ((u0 & 0xF0) == 0xE0) {
                u0 = ((u0 & 15) << 12) | (u1 << 6) | u2;
            }
            else {
                u0 = ((u0 & 7) << 18) | (u1 << 12) | (u2 << 6) | (bytes[idx++] & 63);
            }
            if (u0 < 0x10000) {
                str += String.fromCharCode(u0);
            }
            else {
                const ch = u0 - 0x10000;
                str += String.fromCharCode(0xD800 | (ch >> 10), 0xDC00 | (ch & 0x3FF));
            }
        }
        return str;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid2FzbS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIndhc20udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLFlBQVksQ0FBQztBQUd6QyxNQUFNLFVBQVUsTUFBTSxDQUNwQixJQUFVLEVBQ1YsR0FBVyxFQUNYLE9BQTJCO0lBRTNCLE1BQU0sS0FBSyxHQUFHLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBQzVDLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxNQUFNLENBQUMsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUMsQ0FBQztJQUMxQyxJQUFJLEdBQUcsS0FBSyxDQUFDLEVBQUU7UUFDYixNQUFNLElBQUksV0FBVyxDQUFDLGdCQUFnQixDQUFDLENBQUM7S0FDekM7SUFDRCxNQUFNLEdBQUcsR0FBRyxJQUFJLFVBQVUsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLE1BQU0sRUFBRSxHQUFHLEVBQUUsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLENBQUMsQ0FBQztJQUN0RSxHQUFHLENBQUMsR0FBRyxDQUFDLEtBQUssQ0FBQyxDQUFDO0lBQ2YsR0FBRyxDQUFDLEtBQUssQ0FBQyxNQUFNLENBQUMsR0FBRyxDQUFDLENBQUM7SUFDdEIsSUFBSTtRQUNGLE1BQU0sTUFBTSxHQUFHLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQztRQUM1QixJQUFJLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ2YsT0FBTyxNQUFNLENBQUM7S0FDZjtJQUFDLE9BQU8sS0FBSyxFQUFFO1FBQ2QsSUFBSSxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQztRQUNmLE1BQU0sS0FBSyxDQUFDO0tBQ2I7QUFDSCxDQUFDO0FBR0QsTUFBTSxVQUFVLE1BQU0sQ0FDcEIsSUFBVSxFQUNWLEdBQWUsRUFDZixPQUEyQjtJQUUzQixNQUFNLEdBQUcsR0FBRyxJQUFJ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LEdBQUcsSUFBSSxNQUFNLENBQUMsWUFBWSxDQUFDLE1BQU0sR0FBRyxDQUFDLEVBQUUsSUFBSSxFQUFFLENBQUMsRUFBRSxNQUFNLEdBQUcsQ0FBQyxFQUFFLEdBQUcsS0FBSyxDQUFDLENBQUMsQ0FBQzthQUN4RTtTQUNGO1FBQ0QsT0FBTyxHQUFHLENBQUM7S0FDWjtBQUNILENBQUMiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBXYXNtIH0gZnJvbSBcIi4uL2J1aWxkL3NxbGl0ZS5qc1wiO1xuaW1wb3J0IHsgU3FsaXRlRXJyb3IgfSBmcm9tIFwiLi9lcnJvci50c1wiO1xuXG4vLyBNb3ZlIHN0cmluZyB0byBDXG5leHBvcnQgZnVuY3Rpb24gc2V0U3RyPFQ+KFxuICB3YXNtOiBXYXNtLFxuICBzdHI6IHN0cmluZyxcbiAgY2xvc3VyZTogKHB0cjogbnVtYmVyKSA9PiBULFxuKTogVCB7XG4gIGNvbnN0IGJ5dGVzID0gbmV3IFRleHRFbmNvZGVyKCkuZW5jb2RlKHN0cik7XG4gIGNvbnN0IHB0ciA9IHdhc20ubWFsbG9jKGJ5dGVzLmxlbmd0aCArIDEpO1xuICBpZiAocHRyID09PSAwKSB7XG4gICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKFwiT3V0IG9mIG1lbW9yeS5cIik7XG4gIH1cbiAgY29uc3QgbWVtID0gbmV3IFVpbnQ4QXJyYXkod2FzbS5tZW1vcnkuYnVmZmVyLCBwdHIsIGJ5dGVzLmxlbmd0aCArIDEpO1xuICBtZW0uc2V0KGJ5dGVzKTtcbiAgbWVtW2J5dGVzLmxlbmd0aF0gPSAwOyAvLyBcXDAgdGVybWluYXRvclxuICB0cnkge1xuICAgIGNvbnN0IHJlc3VsdCA9IGNsb3N1cmUocHRyKTtcbiAgICB3YXNtLmZyZWUocHRyKTtcbiAgICByZXR1cm4gcmVzdWx0O1xuICB9IGNhdGNoIChlcnJvcikge1xuICAgIHdhc20uZnJlZShwdHIpO1xuICAgIHRocm93IGVycm9yO1xuICB9XG59XG5cbi8vIE1vdmUgVWludDhBcnJheSB0byBDXG5leHBvcnQgZnVuY3Rpb24gc2V0QXJyPFQ+KFxuICB3YXNtOiBXYXNtLFxuICBhcnI6IFVpbnQ4QXJyYXksXG4gIGNsb3N1cmU6IChwdHI6IG51bWJlcikgPT4gVCxcbik6IFQge1xuICBjb25zdCBwdHIgPSB3YXNtLm1hbGxvYyhhcnIubGVuZ3RoKTtcbiAgaWYgKHB0ciA9PT0gMCkge1xuICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcihcIk91dCBvZiBtZW1vcnkuXCIpO1xuICB9XG4gIGNvbnN0IG1lbSA9IG5ldyBVaW50OEFycmF5KHdhc20ubWVtb3J5LmJ1ZmZlciwgcHRyLCBhcnIubGVuZ3RoKTtcbiAgbWVtLnNldChhcnIpO1xuICB0cnkge1xuICAgIGNvbnN0IHJlc3VsdCA9IGNsb3N1cmUocHRyKTtcbiAgICB3YXNtLmZyZWUocHRyKTtcbiAgICByZXR1cm4gcmVzdWx0O1xuICB9IGNhdGNoIChlcnJvcikge1xuICAgIHdhc20uZnJlZShwdHIpO1xuICAgIHRocm93IGVycm9yO1xuICB9XG59XG5cbi8vIFJlYWQgc3RyaW5nIGZyb20gQ1xuZXhwb3J0IGZ1bmN0aW9uIGdldFN0cih3YXNtOiBXYXNtLCBwdHI6IG51bWJlcik6IHN0cmluZyB7XG4gIGNvbnN0IGxlbiA9IHdhc20uc3RyX2xlbihwdHIpO1xuICBjb25zdCBieXRlcyA9IG5ldyBVaW50OEFycmF5KHdhc20ubWVtb3J5LmJ1ZmZlciwgcHRyLCBsZW4pO1xuICBpZiAobGVuID4gMTYpIHtcbiAgICByZXR1cm4gbmV3IFRleHREZWNvZGVyKCkuZGVjb2RlKGJ5dGVzKTtcbiAgfSBlbHNlIHtcbiAgICAvLyBUaGlzIG9wdGltaXphdGlvbiBpcyBsaWZ0ZWQgZnJvbSBFTVNDUklQVEVOJ3MgZ2x1ZSBjb2RlXG4gICAgbGV0IHN0ciA9IFwiXCI7XG4gICAgbGV0IGlkeCA9IDA7XG4gICAgd2hpbGUgKGlkeCA8IGxlbikge1xuICAgICAgbGV0IHUwID0gYnl0ZXNbaWR4KytdO1xuICAgICAgaWYgKCEodTAgJiAweDgwKSkge1xuICAgICAgICBzdHIgKz0gU3RyaW5nLmZyb21DaGFyQ29kZSh1MCk7XG4gICAgICAgIGNvbnRpbnVlO1xuICAgICAgfVxuICAgICAgY29uc3QgdTEgPSBieXRlc1tpZHgrK10gJiA2MztcbiAgICAgIGlmICgodTAgJiAweEUwKSA9PSAweEMwKSB7XG4gICAgICAgIHN0ciArPSBTdHJpbmcuZnJvbUNoYXJDb2RlKCgodTAgJiAzMSkgPDwgNikgfCB1MSk7XG4gICAgICAgIGNvbnRpbnVlO1xuICAgICAgfVxuICAgICAgY29uc3QgdTIgPSBieXRlc1tpZHgrK10gJiA2MztcbiAgICAgIGlmICgodTAgJiAweEYwKSA9PSAweEUwKSB7XG4gICAgICAgIHUwID0gKCh1MCAmIDE1KSA8PCAxMikgfCAodTEgPDwgNikgfCB1MjtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIC8vIGN1dCB3YXJuaW5nXG4gICAgICAgIHUwID0gKCh1MCAmIDcpIDw8IDE4KSB8ICh1MSA8PCAxMikgfCAodTIgPDwgNikgfCAoYnl0ZXNbaWR4KytdICYgNjMpO1xuICAgICAgfVxuICAgICAgaWYgKHUwIDwgMHgxMDAwMCkge1xuICAgICAgICBzdHIgKz0gU3RyaW5nLmZyb21DaGFyQ29kZSh1MCk7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBjb25zdCBjaCA9IHUwIC0gMHgxMDAwMDtcbiAgICAgICAgc3RyICs9IFN0cmluZy5mcm9tQ2hhckNvZGUoMHhEODAwIHwgKGNoID4+IDEwKSwgMHhEQzAwIHwgKGNoICYgMHgzRkYpKTtcbiAgICAgIH1cbiAgICB9XG4gICAgcmV0dXJuIHN0cjtcbiAgfVxufVxuIl19