import { S3 } from "../S3.ts";
import { S3Client } from "../S3Client.ts";
import { ListObjectsV2Command, } from "../commands/ListObjectsV2Command.ts";
const makePagedClientRequest = async (client, input, ...args) => {
    return await client.send(new ListObjectsV2Command(input), ...args);
};
const makePagedRequest = async (client, input, ...args) => {
    return await client.listObjectsV2(input, ...args);
};
export async function* paginateListObjectsV2(config, input, ...additionalArguments) {
    let token = config.startingToken || undefined;
    let hasNext = true;
    let page;
    while (hasNext) {
        input.ContinuationToken = token;
        input["MaxKeys"] = config.pageSize;
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
        token = page.NextContinuationToken;
        hasNext = !!token;
    }
    return undefined;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdE9iamVjdHNWMlBhZ2luYXRvci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIkxpc3RPYmplY3RzVjJQYWdpbmF0b3IudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLEVBQUUsRUFBRSxNQUFNLFVBQVUsQ0FBQztBQUM5QixPQUFPLEVBQUUsUUFBUSxFQUFFLE1BQU0sZ0JBQWdCLENBQUM7QUFDMUMsT0FBTyxFQUNMLG9CQUFvQixHQUdyQixNQUFNLHFDQUFxQyxDQUFDO0FBTzdDLE1BQU0sc0JBQXNCLEdBQUcsS0FBSyxFQUNsQyxNQUFnQixFQUNoQixLQUFnQyxFQUNoQyxHQUFHLElBQVMsRUFDeUIsRUFBRTtJQUV2QyxPQUFPLE1BQU0sTUFBTSxDQUFDLElBQUksQ0FBQyxJQUFJLG9CQUFvQixDQUFDLEtBQUssQ0FBQyxFQUFFLEdBQUcsSUFBSSxDQUFDLENBQUM7QUFDckUsQ0FBQyxDQUFDO0FBSUYsTUFBTSxnQkFBZ0IsR0FBRyxLQUFLLEVBQzVCLE1BQVUsRUFDVixLQUFnQyxFQUNoQyxHQUFHLElBQVMsRUFDeUIsRUFBRTtJQUV2QyxPQUFPLE1BQU0sTUFBTSxDQUFDLGFBQWEsQ0FBQyxLQUFLLEVBQUUsR0FBRyxJQUFJLENBQUMsQ0FBQztBQUNwRCxDQUFDLENBQUM7QUFDRixNQUFNLENBQUMsS0FBSyxTQUFTLENBQUMsQ0FBQyxxQkFBcUIsQ0FDMUMsTUFBaUMsRUFDakMsS0FBZ0MsRUFDaEMsR0FBRyxtQkFBd0I7SUFHM0IsSUFBSSxLQUFLLEdBQStDLE1BQU0sQ0FBQyxhQUFhLElBQUksU0FBUyxDQUFDO0lBQzFGLElBQUksT0FBTyxHQUFHLElBQUksQ0FBQztJQUNuQixJQUFJLElBQWdDLENBQUM7SUFDckMsT0FBTyxPQUFPLEVBQUU7UUFDZCxLQUFLLENBQUMsaUJBQWlCLEdBQUcsS0FBSyxDQUFDO1FBQ2hDLEtBQUssQ0FBQyxTQUFTLENBQUMsR0FBRyxNQUFNLENBQUMsUUFBUSxDQUFDO1FBQ25DLElBQUksTUFBTSxDQUFDLE1BQU0sWUFBWSxFQUFFLEVBQUU7WUFDL0IsSUFBSSxHQUFHLE1BQU0sZ0JBQWdCLENBQUMsTUFBTSxDQUFDLE1BQU0sRUFBRSxLQUFLLEVBQUUsR0FBRyxtQkFBbUIsQ0FBQyxDQUFDO1NBQzdFO2FBQU0sSUFBSSxNQUFNLENBQUMsTUFBTSxZQUFZLFFBQVEsRUFBRTtZQUM1QyxJQUFJLEdBQUcsTUFBTSxzQkFBc0IsQ0FBQyxNQUFNLENBQUMsTUFBTSxFQUFFLEtBQUssRUFBRSxHQUFHLG1CQUFtQixDQUFDLENBQUM7U0FDbkY7YUFBTTtZQUNMLE1BQU0sSUFBSSxLQUFLLENBQUMsd0NBQXdDLENBQUMsQ0FBQztTQUMzRDtRQUNELE1BQU0sSUFBSSxDQUFDO1FBQ1gsS0FBSyxHQUFHLElBQUksQ0FBQyxxQkFBcUIsQ0FBQztRQUNuQyxPQUFPLEdBQUcsQ0FBQyxDQUFDLEtBQUssQ0FBQztLQUNuQjtJQUVELE9BQU8sU0FBUyxDQUFDO0FBQ25CLENBQUMifQ==