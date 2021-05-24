import { cloneRequest } from "./cloneRequest.ts";
export function moveHeadersToQuery(request, options = {}) {
    const { headers, query = {} } = typeof request.clone === "function" ? request.clone() : cloneRequest(request);
    for (const name of Object.keys(headers)) {
        const lname = name.toLowerCase();
        if (lname.substr(0, 6) === "x-amz-" && !options.unhoistableHeaders?.has(lname)) {
            query[name] = headers[name];
            delete headers[name];
        }
    }
    return {
        ...request,
        headers,
        query,
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW92ZUhlYWRlcnNUb1F1ZXJ5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW92ZUhlYWRlcnNUb1F1ZXJ5LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxZQUFZLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUtqRCxNQUFNLFVBQVUsa0JBQWtCLENBQ2hDLE9BQW9CLEVBQ3BCLFVBQWdELEVBQUU7SUFFbEQsTUFBTSxFQUFFLE9BQU8sRUFBRSxLQUFLLEdBQUcsRUFBdUIsRUFBRSxHQUNoRCxPQUFRLE9BQWUsQ0FBQyxLQUFLLEtBQUssVUFBVSxDQUFDLENBQUMsQ0FBRSxPQUFlLENBQUMsS0FBSyxFQUFFLENBQUMsQ0FBQyxDQUFDLFlBQVksQ0FBQyxPQUFPLENBQUMsQ0FBQztJQUNsRyxLQUFLLE1BQU0sSUFBSSxJQUFJLE1BQU0sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLEVBQUU7UUFDdkMsTUFBTSxLQUFLLEdBQUcsSUFBSSxDQUFDLFdBQVcsRUFBRSxDQUFDO1FBQ2pDLElBQUksS0FBSyxDQUFDLE1BQU0sQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLEtBQUssUUFBUSxJQUFJLENBQUMsT0FBTyxDQUFDLGtCQUFrQixFQUFFLEdBQUcsQ0FBQyxLQUFLLENBQUMsRUFBRTtZQUM5RSxLQUFLLENBQUMsSUFBSSxDQUFDLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDO1lBQzVCLE9BQU8sT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDO1NBQ3RCO0tBQ0Y7SUFFRCxPQUFPO1FBQ0wsR0FBRyxPQUFPO1FBQ1YsT0FBTztRQUNQLEtBQUs7S0FDTixDQUFDO0FBQ0osQ0FBQyJ9