import { resolveAwsAuthConfig } from "../middleware-signing/mod.ts";
export const resolveStsAuthConfig = (input, { stsClientCtor }) => resolveAwsAuthConfig({
    ...input,
    stsClientCtor,
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBNkMsb0JBQW9CLEVBQUUsTUFBTSw4QkFBOEIsQ0FBQztBQWdDL0csTUFBTSxDQUFDLE1BQU0sb0JBQW9CLEdBQUcsQ0FDbEMsS0FBa0QsRUFDbEQsRUFBRSxhQUFhLEVBQXdCLEVBQ1osRUFBRSxDQUM3QixvQkFBb0IsQ0FBQztJQUNuQixHQUFHLEtBQUs7SUFDUixhQUFhO0NBQ2QsQ0FBQyxDQUFDIn0=