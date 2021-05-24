import { escapeUri } from "../util-uri-escape/mod.ts";
export function buildQueryString(query) {
    const parts = [];
    for (let key of Object.keys(query).sort()) {
        const value = query[key];
        key = escapeUri(key);
        if (Array.isArray(value)) {
            for (let i = 0, iLen = value.length; i < iLen; i++) {
                parts.push(`${key}=${escapeUri(value[i])}`);
            }
        }
        else {
            let qsEntry = key;
            if (value || typeof value === "string") {
                qsEntry += `=${escapeUri(value)}`;
            }
            parts.push(qsEntry);
        }
    }
    return parts.join("&");
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxTQUFTLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUV0RCxNQUFNLFVBQVUsZ0JBQWdCLENBQUMsS0FBd0I7SUFDdkQsTUFBTSxLQUFLLEdBQWEsRUFBRSxDQUFDO0lBQzNCLEtBQUssSUFBSSxHQUFHLElBQUksTUFBTSxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxJQUFJLEVBQUUsRUFBRTtRQUN6QyxNQUFNLEtBQUssR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLENBQUM7UUFDekIsR0FBRyxHQUFHLFNBQVMsQ0FBQyxHQUFHLENBQUMsQ0FBQztRQUNyQixJQUFJLEtBQUssQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEVBQUU7WUFDeEIsS0FBSyxJQUFJLENBQUMsR0FBRyxDQUFDLEVBQUUsSUFBSSxHQUFHLEtBQUssQ0FBQyxNQUFNLEVBQUUsQ0FBQyxHQUFHLElBQUksRUFBRSxDQUFDLEVBQUUsRUFBRTtnQkFDbEQsS0FBSyxDQUFDLElBQUksQ0FBQyxHQUFHLEdBQUcsSUFBSSxTQUFTLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDO2FBQzdDO1NBQ0Y7YUFBTTtZQUNMLElBQUksT0FBTyxHQUFHLEdBQUcsQ0FBQztZQUNsQixJQUFJLEtBQUssSUFBSSxPQUFPLEtBQUssS0FBSyxRQUFRLEVBQUU7Z0JBQ3RDLE9BQU8sSUFBSSxJQUFJLFNBQVMsQ0FBQyxLQUFLLENBQUMsRUFBRSxDQUFDO2FBQ25DO1lBQ0QsS0FBSyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsQ0FBQztTQUNyQjtLQUNGO0lBRUQsT0FBTyxLQUFLLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO0FBQ3pCLENBQUMifQ==