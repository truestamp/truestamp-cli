import { validate as validateArn } from "../util-arn-parser/mod.ts";
export function validateBucketNameMiddleware() {
    return (next) => async (args) => {
        const { input: { Bucket }, } = args;
        if (typeof Bucket === "string" && !validateArn(Bucket) && Bucket.indexOf("/") >= 0) {
            const err = new Error(`Bucket name shouldn't contain '/', received '${Bucket}'`);
            err.name = "InvalidBucketName";
            throw err;
        }
        return next({ ...args });
    };
}
export const validateBucketNameMiddlewareOptions = {
    step: "initialize",
    tags: ["VALIDATE_BUCKET_NAME"],
    name: "validateBucketNameMiddleware",
    override: true,
};
export const getValidateBucketNamePlugin = (unused) => ({
    applyToStack: (clientStack) => {
        clientStack.add(validateBucketNameMiddleware(), validateBucketNameMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidmFsaWRhdGUtYnVja2V0LW5hbWUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJ2YWxpZGF0ZS1idWNrZXQtbmFtZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFTQSxPQUFPLEVBQUUsUUFBUSxJQUFJLFdBQVcsRUFBRSxNQUFNLDJCQUEyQixDQUFDO0FBS3BFLE1BQU0sVUFBVSw0QkFBNEI7SUFDMUMsT0FBTyxDQUNMLElBQW9DLEVBQ0osRUFBRSxDQUFDLEtBQUssRUFDeEMsSUFBcUMsRUFDSyxFQUFFO1FBQzVDLE1BQU0sRUFDSixLQUFLLEVBQUUsRUFBRSxNQUFNLEVBQUUsR0FDbEIsR0FBRyxJQUFJLENBQUM7UUFDVCxJQUFJLE9BQU8sTUFBTSxLQUFLLFFBQVEsSUFBSSxDQUFDLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxNQUFNLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxJQUFJLENBQUMsRUFBRTtZQUNsRixNQUFNLEdBQUcsR0FBRyxJQUFJLEtBQUssQ0FBQyxnREFBZ0QsTUFBTSxHQUFHLENBQUMsQ0FBQztZQUNqRixHQUFHLENBQUMsSUFBSSxHQUFHLG1CQUFtQixDQUFDO1lBQy9CLE1BQU0sR0FBRyxDQUFDO1NBQ1g7UUFDRCxPQUFPLElBQUksQ0FBQyxFQUFFLEdBQUcsSUFBSSxFQUFFLENBQUMsQ0FBQztJQUMzQixDQUFDLENBQUM7QUFDSixDQUFDO0FBS0QsTUFBTSxDQUFDLE1BQU0sbUNBQW1DLEdBQTZCO0lBQzNFLElBQUksRUFBRSxZQUFZO0lBQ2xCLElBQUksRUFBRSxDQUFDLHNCQUFzQixDQUFDO0lBQzlCLElBQUksRUFBRSw4QkFBOEI7SUFDcEMsUUFBUSxFQUFFLElBQUk7Q0FDZixDQUFDO0FBTUYsTUFBTSxDQUFDLE1BQU0sMkJBQTJCLEdBQUcsQ0FBQyxNQUFXLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ2hGLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsNEJBQTRCLEVBQUUsRUFBRSxtQ0FBbUMsQ0FBQyxDQUFDO0lBQ3ZGLENBQUM7Q0FDRixDQUFDLENBQUMifQ==