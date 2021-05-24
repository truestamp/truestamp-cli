export class HttpRequest {
    method;
    protocol;
    hostname;
    port;
    path;
    query;
    headers;
    body;
    constructor(options) {
        this.method = options.method || "GET";
        this.hostname = options.hostname || "localhost";
        this.port = options.port;
        this.query = options.query || {};
        this.headers = options.headers || {};
        this.body = options.body;
        this.protocol = options.protocol
            ? options.protocol.substr(-1) !== ":"
                ? `${options.protocol}:`
                : options.protocol
            : "https:";
        this.path = options.path ? (options.path.charAt(0) !== "/" ? `/${options.path}` : options.path) : "/";
    }
    static isInstance(request) {
        if (!request)
            return false;
        const req = request;
        return ("method" in req &&
            "protocol" in req &&
            "hostname" in req &&
            "path" in req &&
            typeof req["query"] === "object" &&
            typeof req["headers"] === "object");
    }
    clone() {
        const cloned = new HttpRequest({
            ...this,
            headers: { ...this.headers },
        });
        if (cloned.query)
            cloned.query = cloneQuery(cloned.query);
        return cloned;
    }
}
function cloneQuery(query) {
    return Object.keys(query).reduce((carry, paramName) => {
        const param = query[paramName];
        return {
            ...carry,
            [paramName]: Array.isArray(param) ? [...param] : param,
        };
    }, {});
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaHR0cFJlcXVlc3QuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJodHRwUmVxdWVzdC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFNQSxNQUFNLE9BQU8sV0FBVztJQUNmLE1BQU0sQ0FBUztJQUNmLFFBQVEsQ0FBUztJQUNqQixRQUFRLENBQVM7SUFDakIsSUFBSSxDQUFVO0lBQ2QsSUFBSSxDQUFTO0lBQ2IsS0FBSyxDQUFvQjtJQUN6QixPQUFPLENBQVk7SUFDbkIsSUFBSSxDQUFPO0lBRWxCLFlBQVksT0FBMkI7UUFDckMsSUFBSSxDQUFDLE1BQU0sR0FBRyxPQUFPLENBQUMsTUFBTSxJQUFJLEtBQUssQ0FBQztRQUN0QyxJQUFJLENBQUMsUUFBUSxHQUFHLE9BQU8sQ0FBQyxRQUFRLElBQUksV0FBVyxDQUFDO1FBQ2hELElBQUksQ0FBQyxJQUFJLEdBQUcsT0FBTyxDQUFDLElBQUksQ0FBQztRQUN6QixJQUFJLENBQUMsS0FBSyxHQUFHLE9BQU8sQ0FBQyxLQUFLLElBQUksRUFBRSxDQUFDO1FBQ2pDLElBQUksQ0FBQyxPQUFPLEdBQUcsT0FBTyxDQUFDLE9BQU8sSUFBSSxFQUFFLENBQUM7UUFDckMsSUFBSSxDQUFDLElBQUksR0FBRyxPQUFPLENBQUMsSUFBSSxDQUFDO1FBQ3pCLElBQUksQ0FBQyxRQUFRLEdBQUcsT0FBTyxDQUFDLFFBQVE7WUFDOUIsQ0FBQyxDQUFDLE9BQU8sQ0FBQyxRQUFRLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQyxDQUFDLEtBQUssR0FBRztnQkFDbkMsQ0FBQyxDQUFDLEdBQUcsT0FBTyxDQUFDLFFBQVEsR0FBRztnQkFDeEIsQ0FBQyxDQUFDLE9BQU8sQ0FBQyxRQUFRO1lBQ3BCLENBQUMsQ0FBQyxRQUFRLENBQUM7UUFDYixJQUFJLENBQUMsSUFBSSxHQUFHLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDLEtBQUssR0FBRyxDQUFDLENBQUMsQ0FBQyxJQUFJLE9BQU8sQ0FBQyxJQUFJLEVBQUUsQ0FBQyxDQUFDLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUM7SUFDeEcsQ0FBQztJQUVELE1BQU0sQ0FBQyxVQUFVLENBQUMsT0FBZ0I7UUFFaEMsSUFBSSxDQUFDLE9BQU87WUFBRSxPQUFPLEtBQUssQ0FBQztRQUMzQixNQUFNLEdBQUcsR0FBUSxPQUFPLENBQUM7UUFDekIsT0FBTyxDQUNMLFFBQVEsSUFBSSxHQUFHO1lBQ2YsVUFBVSxJQUFJLEdBQUc7WUFDakIsVUFBVSxJQUFJLEdBQUc7WUFDakIsTUFBTSxJQUFJLEdBQUc7WUFDYixPQUFPLEdBQUcsQ0FBQyxPQUFPLENBQUMsS0FBSyxRQUFRO1lBQ2hDLE9BQU8sR0FBRyxDQUFDLFNBQVMsQ0FBQyxLQUFLLFFBQVEsQ0FDbkMsQ0FBQztJQUNKLENBQUM7SUFFRCxLQUFLO1FBQ0gsTUFBTSxNQUFNLEdBQUcsSUFBSSxXQUFXLENBQUM7WUFDN0IsR0FBRyxJQUFJO1lBQ1AsT0FBTyxFQUFFLEVBQUUsR0FBRyxJQUFJLENBQUMsT0FBTyxFQUFFO1NBQzdCLENBQUMsQ0FBQztRQUNILElBQUksTUFBTSxDQUFDLEtBQUs7WUFBRSxNQUFNLENBQUMsS0FBSyxHQUFHLFVBQVUsQ0FBQyxNQUFNLENBQUMsS0FBSyxDQUFDLENBQUM7UUFDMUQsT0FBTyxNQUFNLENBQUM7SUFDaEIsQ0FBQztDQUNGO0FBRUQsU0FBUyxVQUFVLENBQUMsS0FBd0I7SUFDMUMsT0FBTyxNQUFNLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDLE1BQU0sQ0FBQyxDQUFDLEtBQXdCLEVBQUUsU0FBaUIsRUFBRSxFQUFFO1FBQy9FLE1BQU0sS0FBSyxHQUFHLEtBQUssQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUMvQixPQUFPO1lBQ0wsR0FBRyxLQUFLO1lBQ1IsQ0FBQyxTQUFTLENBQUMsRUFBRSxLQUFLLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDLEdBQUcsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDLEtBQUs7U0FDdkQsQ0FBQztJQUNKLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQztBQUNULENBQUMifQ==