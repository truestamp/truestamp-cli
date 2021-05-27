export const REGION_ENV_NAME = "AWS_REGION";
export const REGION_INI_NAME = "region";
export const NODE_REGION_CONFIG_OPTIONS = {
    environmentVariableSelector: (env) => env[REGION_ENV_NAME],
    configFileSelector: (profile) => profile[REGION_INI_NAME],
    default: () => {
        throw new Error("Region is missing");
    },
};
export const NODE_REGION_CONFIG_FILE_OPTIONS = {
    preferredFile: "credentials",
};
export const resolveRegionConfig = (input) => {
    if (!input.region) {
        throw new Error("Region is missing");
    }
    return {
        ...input,
        region: normalizeRegion(input.region),
    };
};
const normalizeRegion = (region) => {
    if (typeof region === "string") {
        const promisified = Promise.resolve(region);
        return () => promisified;
    }
    return region;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUmVnaW9uQ29uZmlnLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiUmVnaW9uQ29uZmlnLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUdBLE1BQU0sQ0FBQyxNQUFNLGVBQWUsR0FBRyxZQUFZLENBQUM7QUFDNUMsTUFBTSxDQUFDLE1BQU0sZUFBZSxHQUFHLFFBQVEsQ0FBQztBQUV4QyxNQUFNLENBQUMsTUFBTSwwQkFBMEIsR0FBa0M7SUFDdkUsMkJBQTJCLEVBQUUsQ0FBQyxHQUFHLEVBQUUsRUFBRSxDQUFDLEdBQUcsQ0FBQyxlQUFlLENBQUM7SUFDMUQsa0JBQWtCLEVBQUUsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUFDLE9BQU8sQ0FBQyxlQUFlLENBQUM7SUFDekQsT0FBTyxFQUFFLEdBQUcsRUFBRTtRQUNaLE1BQU0sSUFBSSxLQUFLLENBQUMsbUJBQW1CLENBQUMsQ0FBQztJQUN2QyxDQUFDO0NBQ0YsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLCtCQUErQixHQUF1QjtJQUNqRSxhQUFhLEVBQUUsYUFBYTtDQUM3QixDQUFDO0FBZUYsTUFBTSxDQUFDLE1BQU0sbUJBQW1CLEdBQUcsQ0FBSSxLQUFpRCxFQUE0QixFQUFFO0lBQ3BILElBQUksQ0FBQyxLQUFLLENBQUMsTUFBTSxFQUFFO1FBQ2pCLE1BQU0sSUFBSSxLQUFLLENBQUMsbUJBQW1CLENBQUMsQ0FBQztLQUN0QztJQUNELE9BQU87UUFDTCxHQUFHLEtBQUs7UUFDUixNQUFNLEVBQUUsZUFBZSxDQUFDLEtBQUssQ0FBQyxNQUFPLENBQUM7S0FDdkMsQ0FBQztBQUNKLENBQUMsQ0FBQztBQUVGLE1BQU0sZUFBZSxHQUFHLENBQUMsTUFBaUMsRUFBb0IsRUFBRTtJQUM5RSxJQUFJLE9BQU8sTUFBTSxLQUFLLFFBQVEsRUFBRTtRQUM5QixNQUFNLFdBQVcsR0FBRyxPQUFPLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxDQUFDO1FBQzVDLE9BQU8sR0FBRyxFQUFFLENBQUMsV0FBVyxDQUFDO0tBQzFCO0lBQ0QsT0FBTyxNQUEwQixDQUFDO0FBQ3BDLENBQUMsQ0FBQyJ9