import { SSO } from "../SSO.ts";
import { SSOClient } from "../SSOClient.ts";
import { ListAccountsCommand, } from "../commands/ListAccountsCommand.ts";
const makePagedClientRequest = async (client, input, ...args) => {
    return await client.send(new ListAccountsCommand(input), ...args);
};
const makePagedRequest = async (client, input, ...args) => {
    return await client.listAccounts(input, ...args);
};
export async function* paginateListAccounts(config, input, ...additionalArguments) {
    let token = config.startingToken || undefined;
    let hasNext = true;
    let page;
    while (hasNext) {
        input.nextToken = token;
        input["maxResults"] = config.pageSize;
        if (config.client instanceof SSO) {
            page = await makePagedRequest(config.client, input, ...additionalArguments);
        }
        else if (config.client instanceof SSOClient) {
            page = await makePagedClientRequest(config.client, input, ...additionalArguments);
        }
        else {
            throw new Error("Invalid client, expected SSO | SSOClient");
        }
        yield page;
        token = page.nextToken;
        hasNext = !!token;
    }
    return undefined;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdEFjY291bnRzUGFnaW5hdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiTGlzdEFjY291bnRzUGFnaW5hdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxHQUFHLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDaEMsT0FBTyxFQUFFLFNBQVMsRUFBRSxNQUFNLGlCQUFpQixDQUFDO0FBQzVDLE9BQU8sRUFDTCxtQkFBbUIsR0FHcEIsTUFBTSxvQ0FBb0MsQ0FBQztBQU81QyxNQUFNLHNCQUFzQixHQUFHLEtBQUssRUFDbEMsTUFBaUIsRUFDakIsS0FBK0IsRUFDL0IsR0FBRyxJQUFTLEVBQ3dCLEVBQUU7SUFFdEMsT0FBTyxNQUFNLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxtQkFBbUIsQ0FBQyxLQUFLLENBQUMsRUFBRSxHQUFHLElBQUksQ0FBQyxDQUFDO0FBQ3BFLENBQUMsQ0FBQztBQUlGLE1BQU0sZ0JBQWdCLEdBQUcsS0FBSyxFQUM1QixNQUFXLEVBQ1gsS0FBK0IsRUFDL0IsR0FBRyxJQUFTLEVBQ3dCLEVBQUU7SUFFdEMsT0FBTyxNQUFNLE1BQU0sQ0FBQyxZQUFZLENBQUMsS0FBSyxFQUFFLEdBQUcsSUFBSSxDQUFDLENBQUM7QUFDbkQsQ0FBQyxDQUFDO0FBQ0YsTUFBTSxDQUFDLEtBQUssU0FBUyxDQUFDLENBQUMsb0JBQW9CLENBQ3pDLE1BQWtDLEVBQ2xDLEtBQStCLEVBQy9CLEdBQUcsbUJBQXdCO0lBRzNCLElBQUksS0FBSyxHQUF1QyxNQUFNLENBQUMsYUFBYSxJQUFJLFNBQVMsQ0FBQztJQUNsRixJQUFJLE9BQU8sR0FBRyxJQUFJLENBQUM7SUFDbkIsSUFBSSxJQUErQixDQUFDO0lBQ3BDLE9BQU8sT0FBTyxFQUFFO1FBQ2QsS0FBSyxDQUFDLFNBQVMsR0FBRyxLQUFLLENBQUM7UUFDeEIsS0FBSyxDQUFDLFlBQVksQ0FBQyxHQUFHLE1BQU0sQ0FBQyxRQUFRLENBQUM7UUFDdEMsSUFBSSxNQUFNLENBQUMsTUFBTSxZQUFZLEdBQUcsRUFBRTtZQUNoQyxJQUFJLEdBQUcsTUFBTSxnQkFBZ0IsQ0FBQyxNQUFNLENBQUMsTUFBTSxFQUFFLEtBQUssRUFBRSxHQUFHLG1CQUFtQixDQUFDLENBQUM7U0FDN0U7YUFBTSxJQUFJLE1BQU0sQ0FBQyxNQUFNLFlBQVksU0FBUyxFQUFFO1lBQzdDLElBQUksR0FBRyxNQUFNLHNCQUFzQixDQUFDLE1BQU0sQ0FBQyxNQUFNLEVBQUUsS0FBSyxFQUFFLEdBQUcsbUJBQW1CLENBQUMsQ0FBQztTQUNuRjthQUFNO1lBQ0wsTUFBTSxJQUFJLEtBQUssQ0FBQywwQ0FBMEMsQ0FBQyxDQUFDO1NBQzdEO1FBQ0QsTUFBTSxJQUFJLENBQUM7UUFDWCxLQUFLLEdBQUcsSUFBSSxDQUFDLFNBQVMsQ0FBQztRQUN2QixPQUFPLEdBQUcsQ0FBQyxDQUFDLEtBQUssQ0FBQztLQUNuQjtJQUVELE9BQU8sU0FBUyxDQUFDO0FBQ25CLENBQUMifQ==