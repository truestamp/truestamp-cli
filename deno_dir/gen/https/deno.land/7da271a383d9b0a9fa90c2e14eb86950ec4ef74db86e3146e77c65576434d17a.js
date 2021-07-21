import { parseQueryString } from "../querystring-parser/mod.ts";
export const parseUrl = (url) => {
    const { hostname, pathname, port, protocol, search } = new URL(url);
    let query;
    if (search) {
        query = parseQueryString(search);
    }
    return {
        hostname,
        port: port ? parseInt(port) : undefined,
        protocol,
        path: pathname,
        query,
    };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxnQkFBZ0IsRUFBRSxNQUFNLDhCQUE4QixDQUFDO0FBR2hFLE1BQU0sQ0FBQyxNQUFNLFFBQVEsR0FBYyxDQUFDLEdBQVcsRUFBWSxFQUFFO0lBQzNELE1BQU0sRUFBRSxRQUFRLEVBQUUsUUFBUSxFQUFFLElBQUksRUFBRSxRQUFRLEVBQUUsTUFBTSxFQUFFLEdBQUcsSUFBSSxHQUFHLENBQUMsR0FBRyxDQUFDLENBQUM7SUFFcEUsSUFBSSxLQUFvQyxDQUFDO0lBQ3pDLElBQUksTUFBTSxFQUFFO1FBQ1YsS0FBSyxHQUFHLGdCQUFnQixDQUFDLE1BQU0sQ0FBQyxDQUFDO0tBQ2xDO0lBRUQsT0FBTztRQUNMLFFBQVE7UUFDUixJQUFJLEVBQUUsSUFBSSxDQUFDLENBQUMsQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDLFNBQVM7UUFDdkMsUUFBUTtRQUNSLElBQUksRUFBRSxRQUFRO1FBQ2QsS0FBSztLQUNOLENBQUM7QUFDSixDQUFDLENBQUMifQ==