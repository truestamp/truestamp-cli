export function parseQueryString(querystring) {
    const query = {};
    querystring = querystring.replace(/^\?/, "");
    if (querystring) {
        for (const pair of querystring.split("&")) {
            let [key, value = null] = pair.split("=");
            key = decodeURIComponent(key);
            if (value) {
                value = decodeURIComponent(value);
            }
            if (!(key in query)) {
                query[key] = value;
            }
            else if (Array.isArray(query[key])) {
                query[key].push(value);
            }
            else {
                query[key] = [query[key], value];
            }
        }
    }
    return query;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE1BQU0sVUFBVSxnQkFBZ0IsQ0FBQyxXQUFtQjtJQUNsRCxNQUFNLEtBQUssR0FBc0IsRUFBRSxDQUFDO0lBQ3BDLFdBQVcsR0FBRyxXQUFXLENBQUMsT0FBTyxDQUFDLEtBQUssRUFBRSxFQUFFLENBQUMsQ0FBQztJQUU3QyxJQUFJLFdBQVcsRUFBRTtRQUNmLEtBQUssTUFBTSxJQUFJLElBQUksV0FBVyxDQUFDLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRTtZQUN6QyxJQUFJLENBQUMsR0FBRyxFQUFFLEtBQUssR0FBRyxJQUFJLENBQUMsR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1lBQzFDLEdBQUcsR0FBRyxrQkFBa0IsQ0FBQyxHQUFHLENBQUMsQ0FBQztZQUM5QixJQUFJLEtBQUssRUFBRTtnQkFDVCxLQUFLLEdBQUcsa0JBQWtCLENBQUMsS0FBSyxDQUFDLENBQUM7YUFDbkM7WUFDRCxJQUFJLENBQUMsQ0FBQyxHQUFHLElBQUksS0FBSyxDQUFDLEVBQUU7Z0JBQ25CLEtBQUssQ0FBQyxHQUFHLENBQUMsR0FBRyxLQUFLLENBQUM7YUFDcEI7aUJBQU0sSUFBSSxLQUFLLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxHQUFHLENBQUMsQ0FBQyxFQUFFO2dCQUNuQyxLQUFLLENBQUMsR0FBRyxDQUFtQixDQUFDLElBQUksQ0FBQyxLQUFlLENBQUMsQ0FBQzthQUNyRDtpQkFBTTtnQkFDTCxLQUFLLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFXLEVBQUUsS0FBZSxDQUFDLENBQUM7YUFDdEQ7U0FDRjtLQUNGO0lBRUQsT0FBTyxLQUFLLENBQUM7QUFDZixDQUFDIn0=