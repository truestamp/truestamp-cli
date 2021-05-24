import { SSO } from "../SSO.ts";
import { SSOClient } from "../SSOClient.ts";
import { ListAccountRolesCommand, } from "../commands/ListAccountRolesCommand.ts";
const makePagedClientRequest = async (client, input, ...args) => {
    return await client.send(new ListAccountRolesCommand(input), ...args);
};
const makePagedRequest = async (client, input, ...args) => {
    return await client.listAccountRoles(input, ...args);
};
export async function* paginateListAccountRoles(config, input, ...additionalArguments) {
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiTGlzdEFjY291bnRSb2xlc1BhZ2luYXRvci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIkxpc3RBY2NvdW50Um9sZXNQYWdpbmF0b3IudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLEdBQUcsRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUNoQyxPQUFPLEVBQUUsU0FBUyxFQUFFLE1BQU0saUJBQWlCLENBQUM7QUFDNUMsT0FBTyxFQUNMLHVCQUF1QixHQUd4QixNQUFNLHdDQUF3QyxDQUFDO0FBT2hELE1BQU0sc0JBQXNCLEdBQUcsS0FBSyxFQUNsQyxNQUFpQixFQUNqQixLQUFtQyxFQUNuQyxHQUFHLElBQVMsRUFDNEIsRUFBRTtJQUUxQyxPQUFPLE1BQU0sTUFBTSxDQUFDLElBQUksQ0FBQyxJQUFJLHVCQUF1QixDQUFDLEtBQUssQ0FBQyxFQUFFLEdBQUcsSUFBSSxDQUFDLENBQUM7QUFDeEUsQ0FBQyxDQUFDO0FBSUYsTUFBTSxnQkFBZ0IsR0FBRyxLQUFLLEVBQzVCLE1BQVcsRUFDWCxLQUFtQyxFQUNuQyxHQUFHLElBQVMsRUFDNEIsRUFBRTtJQUUxQyxPQUFPLE1BQU0sTUFBTSxDQUFDLGdCQUFnQixDQUFDLEtBQUssRUFBRSxHQUFHLElBQUksQ0FBQyxDQUFDO0FBQ3ZELENBQUMsQ0FBQztBQUNGLE1BQU0sQ0FBQyxLQUFLLFNBQVMsQ0FBQyxDQUFDLHdCQUF3QixDQUM3QyxNQUFrQyxFQUNsQyxLQUFtQyxFQUNuQyxHQUFHLG1CQUF3QjtJQUczQixJQUFJLEtBQUssR0FBdUMsTUFBTSxDQUFDLGFBQWEsSUFBSSxTQUFTLENBQUM7SUFDbEYsSUFBSSxPQUFPLEdBQUcsSUFBSSxDQUFDO0lBQ25CLElBQUksSUFBbUMsQ0FBQztJQUN4QyxPQUFPLE9BQU8sRUFBRTtRQUNkLEtBQUssQ0FBQyxTQUFTLEdBQUcsS0FBSyxDQUFDO1FBQ3hCLEtBQUssQ0FBQyxZQUFZLENBQUMsR0FBRyxNQUFNLENBQUMsUUFBUSxDQUFDO1FBQ3RDLElBQUksTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLEVBQUU7WUFDaEMsSUFBSSxHQUFHLE1BQU0sZ0JBQWdCLENBQUMsTUFBTSxDQUFDLE1BQU0sRUFBRSxLQUFLLEVBQUUsR0FBRyxtQkFBbUIsQ0FBQyxDQUFDO1NBQzdFO2FBQU0sSUFBSSxNQUFNLENBQUMsTUFBTSxZQUFZLFNBQVMsRUFBRTtZQUM3QyxJQUFJLEdBQUcsTUFBTSxzQkFBc0IsQ0FBQyxNQUFNLENBQUMsTUFBTSxFQUFFLEtBQUssRUFBRSxHQUFHLG1CQUFtQixDQUFDLENBQUM7U0FDbkY7YUFBTTtZQUNMLE1BQU0sSUFBSSxLQUFLLENBQUMsMENBQTBDLENBQUMsQ0FBQztTQUM3RDtRQUNELE1BQU0sSUFBSSxDQUFDO1FBQ1gsS0FBSyxHQUFHLElBQUksQ0FBQyxTQUFTLENBQUM7UUFDdkIsT0FBTyxHQUFHLENBQUMsQ0FBQyxLQUFLLENBQUM7S0FDbkI7SUFFRCxPQUFPLFNBQVMsQ0FBQztBQUNuQixDQUFDIn0=