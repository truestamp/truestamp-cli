import { fromFileUrl } from "../path.ts";
export function mkdir(path, options, callback) {
    path = path instanceof URL ? fromFileUrl(path) : path;
    let mode = 0o777;
    let recursive = false;
    if (typeof options == "function") {
        callback = options;
    }
    else if (typeof options === "number") {
        mode = options;
    }
    else if (typeof options === "boolean") {
        recursive = options;
    }
    else if (options) {
        if (options.recursive !== undefined)
            recursive = options.recursive;
        if (options.mode !== undefined)
            mode = options.mode;
    }
    if (typeof recursive !== "boolean") {
        throw new Deno.errors.InvalidData("invalid recursive option , must be a boolean");
    }
    Deno.mkdir(path, { recursive, mode })
        .then(() => {
        if (typeof callback === "function") {
            callback(null);
        }
    }, (err) => {
        if (typeof callback === "function") {
            callback(err);
        }
    });
}
export function mkdirSync(path, options) {
    path = path instanceof URL ? fromFileUrl(path) : path;
    let mode = 0o777;
    let recursive = false;
    if (typeof options === "number") {
        mode = options;
    }
    else if (typeof options === "boolean") {
        recursive = options;
    }
    else if (options) {
        if (options.recursive !== undefined)
            recursive = options.recursive;
        if (options.mode !== undefined)
            mode = options.mode;
    }
    if (typeof recursive !== "boolean") {
        throw new Deno.errors.InvalidData("invalid recursive option , must be a boolean");
    }
    Deno.mkdirSync(path, { recursive, mode });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX21rZGlyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiX2ZzX21rZGlyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSxZQUFZLENBQUM7QUFXekMsTUFBTSxVQUFVLEtBQUssQ0FDbkIsSUFBa0IsRUFDbEIsT0FBMEMsRUFDMUMsUUFBNEI7SUFFNUIsSUFBSSxHQUFHLElBQUksWUFBWSxHQUFHLENBQUMsQ0FBQyxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDO0lBRXRELElBQUksSUFBSSxHQUFHLEtBQUssQ0FBQztJQUNqQixJQUFJLFNBQVMsR0FBRyxLQUFLLENBQUM7SUFFdEIsSUFBSSxPQUFPLE9BQU8sSUFBSSxVQUFVLEVBQUU7UUFDaEMsUUFBUSxHQUFHLE9BQU8sQ0FBQztLQUNwQjtTQUFNLElBQUksT0FBTyxPQUFPLEtBQUssUUFBUSxFQUFFO1FBQ3RDLElBQUksR0FBRyxPQUFPLENBQUM7S0FDaEI7U0FBTSxJQUFJLE9BQU8sT0FBTyxLQUFLLFNBQVMsRUFBRTtRQUN2QyxTQUFTLEdBQUcsT0FBTyxDQUFDO0tBQ3JCO1NBQU0sSUFBSSxPQUFPLEVBQUU7UUFDbEIsSUFBSSxPQUFPLENBQUMsU0FBUyxLQUFLLFNBQVM7WUFBRSxTQUFTLEdBQUcsT0FBTyxDQUFDLFNBQVMsQ0FBQztRQUNuRSxJQUFJLE9BQU8sQ0FBQyxJQUFJLEtBQUssU0FBUztZQUFFLElBQUksR0FBRyxPQUFPLENBQUMsSUFBSSxDQUFDO0tBQ3JEO0lBQ0QsSUFBSSxPQUFPLFNBQVMsS0FBSyxTQUFTLEVBQUU7UUFDbEMsTUFBTSxJQUFJLElBQUksQ0FBQyxNQUFNLENBQUMsV0FBVyxDQUMvQiw4Q0FBOEMsQ0FDL0MsQ0FBQztLQUNIO0lBQ0QsSUFBSSxDQUFDLEtBQUssQ0FBQyxJQUFJLEVBQUUsRUFBRSxTQUFTLEVBQUUsSUFBSSxFQUFFLENBQUM7U0FDbEMsSUFBSSxDQUFDLEdBQUcsRUFBRTtRQUNULElBQUksT0FBTyxRQUFRLEtBQUssVUFBVSxFQUFFO1lBQ2xDLFFBQVEsQ0FBQyxJQUFJLENBQUMsQ0FBQztTQUNoQjtJQUNILENBQUMsRUFBRSxDQUFDLEdBQUcsRUFBRSxFQUFFO1FBQ1QsSUFBSSxPQUFPLFFBQVEsS0FBSyxVQUFVLEVBQUU7WUFDbEMsUUFBUSxDQUFDLEdBQUcsQ0FBQyxDQUFDO1NBQ2Y7SUFDSCxDQUFDLENBQUMsQ0FBQztBQUNQLENBQUM7QUFFRCxNQUFNLFVBQVUsU0FBUyxDQUFDLElBQWtCLEVBQUUsT0FBc0I7SUFDbEUsSUFBSSxHQUFHLElBQUksWUFBWSxHQUFHLENBQUMsQ0FBQyxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDO0lBQ3RELElBQUksSUFBSSxHQUFHLEtBQUssQ0FBQztJQUNqQixJQUFJLFNBQVMsR0FBRyxLQUFLLENBQUM7SUFFdEIsSUFBSSxPQUFPLE9BQU8sS0FBSyxRQUFRLEVBQUU7UUFDL0IsSUFBSSxHQUFHLE9BQU8sQ0FBQztLQUNoQjtTQUFNLElBQUksT0FBTyxPQUFPLEtBQUssU0FBUyxFQUFFO1FBQ3ZDLFNBQVMsR0FBRyxPQUFPLENBQUM7S0FDckI7U0FBTSxJQUFJLE9BQU8sRUFBRTtRQUNsQixJQUFJLE9BQU8sQ0FBQyxTQUFTLEtBQUssU0FBUztZQUFFLFNBQVMsR0FBRyxPQUFPLENBQUMsU0FBUyxDQUFDO1FBQ25FLElBQUksT0FBTyxDQUFDLElBQUksS0FBSyxTQUFTO1lBQUUsSUFBSSxHQUFHLE9BQU8sQ0FBQyxJQUFJLENBQUM7S0FDckQ7SUFDRCxJQUFJLE9BQU8sU0FBUyxLQUFLLFNBQVMsRUFBRTtRQUNsQyxNQUFNLElBQUksSUFBSSxDQUFDLE1BQU0sQ0FBQyxXQUFXLENBQy9CLDhDQUE4QyxDQUMvQyxDQUFDO0tBQ0g7SUFFRCxJQUFJLENBQUMsU0FBUyxDQUFDLElBQUksRUFBRSxFQUFFLFNBQVMsRUFBRSxJQUFJLEVBQUUsQ0FBQyxDQUFDO0FBQzVDLENBQUMifQ==