import { S3 } from "../S3.ts";
import { S3Client } from "../S3Client.ts";
import { ListPartsCommand } from "../commands/ListPartsCommand.ts";
const makePagedClientRequest = async (client, input, ...args) => {
    return await client.send(new ListPartsCommand(input), ...args);
};
const makePagedRequest = async (client, input, ...args) => {
    return await client.listParts(input, ...args);
};
export async function* paginateListParts(config, input, ...additionalArguments) {
    let token = config.startingToken || undefined;
    let hasNext = true;
    let page;
    while (hasNext) {
        input.PartNumberMarker = token;
        input["MaxParts"] = config.pageSize;
        if (config.client instanceof S3) {
            page = await makePagedRequest(config.client, input, ...additionalArguments);
        }
        else if (config.client instanceof S3Client) {
            page = await makePagedClientRequest(config.client, input, ...additionalArguments);
        }
        else {
            throw new Error("Invalid client, expected S3 | S3Client");
        }
        yield page;
        token = page.NextPartNumberMarker;
        hasNext = !!token;
    }
    return undefined;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdFBhcnRzUGFnaW5hdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiTGlzdFBhcnRzUGFnaW5hdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxFQUFFLEVBQUUsTUFBTSxVQUFVLENBQUM7QUFDOUIsT0FBTyxFQUFFLFFBQVEsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBQzFDLE9BQU8sRUFBRSxnQkFBZ0IsRUFBaUQsTUFBTSxpQ0FBaUMsQ0FBQztBQU9sSCxNQUFNLHNCQUFzQixHQUFHLEtBQUssRUFDbEMsTUFBZ0IsRUFDaEIsS0FBNEIsRUFDNUIsR0FBRyxJQUFTLEVBQ3FCLEVBQUU7SUFFbkMsT0FBTyxNQUFNLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxnQkFBZ0IsQ0FBQyxLQUFLLENBQUMsRUFBRSxHQUFHLElBQUksQ0FBQyxDQUFDO0FBQ2pFLENBQUMsQ0FBQztBQUlGLE1BQU0sZ0JBQWdCLEdBQUcsS0FBSyxFQUM1QixNQUFVLEVBQ1YsS0FBNEIsRUFDNUIsR0FBRyxJQUFTLEVBQ3FCLEVBQUU7SUFFbkMsT0FBTyxNQUFNLE1BQU0sQ0FBQyxTQUFTLENBQUMsS0FBSyxFQUFFLEdBQUcsSUFBSSxDQUFDLENBQUM7QUFDaEQsQ0FBQyxDQUFDO0FBQ0YsTUFBTSxDQUFDLEtBQUssU0FBUyxDQUFDLENBQUMsaUJBQWlCLENBQ3RDLE1BQWlDLEVBQ2pDLEtBQTRCLEVBQzVCLEdBQUcsbUJBQXdCO0lBRzNCLElBQUksS0FBSyxHQUE4QyxNQUFNLENBQUMsYUFBYSxJQUFJLFNBQVMsQ0FBQztJQUN6RixJQUFJLE9BQU8sR0FBRyxJQUFJLENBQUM7SUFDbkIsSUFBSSxJQUE0QixDQUFDO0lBQ2pDLE9BQU8sT0FBTyxFQUFFO1FBQ2QsS0FBSyxDQUFDLGdCQUFnQixHQUFHLEtBQUssQ0FBQztRQUMvQixLQUFLLENBQUMsVUFBVSxDQUFDLEdBQUcsTUFBTSxDQUFDLFFBQVEsQ0FBQztRQUNwQyxJQUFJLE1BQU0sQ0FBQyxNQUFNLFlBQVksRUFBRSxFQUFFO1lBQy9CLElBQUksR0FBRyxNQUFNLGdCQUFnQixDQUFDLE1BQU0sQ0FBQyxNQUFNLEVBQUUsS0FBSyxFQUFFLEdBQUcsbUJBQW1CLENBQUMsQ0FBQztTQUM3RTthQUFNLElBQUksTUFBTSxDQUFDLE1BQU0sWUFBWSxRQUFRLEVBQUU7WUFDNUMsSUFBSSxHQUFHLE1BQU0sc0JBQXNCLENBQUMsTUFBTSxDQUFDLE1BQU0sRUFBRSxLQUFLLEVBQUUsR0FBRyxtQkFBbUIsQ0FBQyxDQUFDO1NBQ25GO2FBQU07WUFDTCxNQUFNLElBQUksS0FBSyxDQUFDLHdDQUF3QyxDQUFDLENBQUM7U0FDM0Q7UUFDRCxNQUFNLElBQUksQ0FBQztRQUNYLEtBQUssR0FBRyxJQUFJLENBQUMsb0JBQW9CLENBQUM7UUFDbEMsT0FBTyxHQUFHLENBQUMsQ0FBQyxLQUFLLENBQUM7S0FDbkI7SUFFRCxPQUFPLFNBQVMsQ0FBQztBQUNuQixDQUFDIn0=