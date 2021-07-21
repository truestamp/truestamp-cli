import { resolveAwsAuthConfig } from "../middleware-signing/mod.ts";
export const resolveStsAuthConfig = (input, stsClientCtor) => {
    return resolveAwsAuthConfig({
        ...input,
        stsClientCtor,
    });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBNkMsb0JBQW9CLEVBQUUsTUFBTSw4QkFBOEIsQ0FBQztBQXlCL0csTUFBTSxDQUFDLE1BQU0sb0JBQW9CLEdBQUcsQ0FDbEMsS0FBa0QsRUFDbEQsYUFBK0QsRUFDcEMsRUFBRTtJQUM3QixPQUFPLG9CQUFvQixDQUFDO1FBQzFCLEdBQUcsS0FBSztRQUNSLGFBQWE7S0FDZCxDQUFDLENBQUM7QUFDTCxDQUFDLENBQUMifQ==