const AWS_TEMPLATE = "sts.{region}.amazonaws.com";
const AWS_CN_TEMPLATE = "sts.{region}.amazonaws.com.cn";
const AWS_ISO_TEMPLATE = "sts.{region}.c2s.ic.gov";
const AWS_ISO_B_TEMPLATE = "sts.{region}.sc2s.sgov.gov";
const AWS_US_GOV_TEMPLATE = "sts.{region}.amazonaws.com";
const AWS_REGIONS = new Set([
    "af-south-1",
    "ap-east-1",
    "ap-northeast-1",
    "ap-northeast-2",
    "ap-northeast-3",
    "ap-south-1",
    "ap-southeast-1",
    "ap-southeast-2",
    "ca-central-1",
    "eu-central-1",
    "eu-north-1",
    "eu-south-1",
    "eu-west-1",
    "eu-west-2",
    "eu-west-3",
    "me-south-1",
    "sa-east-1",
    "us-east-1",
    "us-east-2",
    "us-west-1",
    "us-west-2",
]);
const AWS_CN_REGIONS = new Set(["cn-north-1", "cn-northwest-1"]);
const AWS_ISO_REGIONS = new Set(["us-iso-east-1"]);
const AWS_ISO_B_REGIONS = new Set(["us-isob-east-1"]);
const AWS_US_GOV_REGIONS = new Set(["us-gov-east-1", "us-gov-west-1"]);
export const defaultRegionInfoProvider = (region, options) => {
    let regionInfo = undefined;
    switch (region) {
        case "af-south-1":
            regionInfo = {
                hostname: "sts.af-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-east-1":
            regionInfo = {
                hostname: "sts.ap-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-1":
            regionInfo = {
                hostname: "sts.ap-northeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-2":
            regionInfo = {
                hostname: "sts.ap-northeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-3":
            regionInfo = {
                hostname: "sts.ap-northeast-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-south-1":
            regionInfo = {
                hostname: "sts.ap-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-southeast-1":
            regionInfo = {
                hostname: "sts.ap-southeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-southeast-2":
            regionInfo = {
                hostname: "sts.ap-southeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "aws-global":
            regionInfo = {
                hostname: "sts.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "ca-central-1":
            regionInfo = {
                hostname: "sts.ca-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "cn-north-1":
            regionInfo = {
                hostname: "sts.cn-north-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "cn-northwest-1":
            regionInfo = {
                hostname: "sts.cn-northwest-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "eu-central-1":
            regionInfo = {
                hostname: "sts.eu-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-north-1":
            regionInfo = {
                hostname: "sts.eu-north-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-south-1":
            regionInfo = {
                hostname: "sts.eu-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-1":
            regionInfo = {
                hostname: "sts.eu-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-2":
            regionInfo = {
                hostname: "sts.eu-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-3":
            regionInfo = {
                hostname: "sts.eu-west-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "me-south-1":
            regionInfo = {
                hostname: "sts.me-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "sa-east-1":
            regionInfo = {
                hostname: "sts.sa-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-1":
            regionInfo = {
                hostname: "sts.us-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-1-fips":
            regionInfo = {
                hostname: "sts-fips.us-east-1.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "us-east-2":
            regionInfo = {
                hostname: "sts.us-east-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-2-fips":
            regionInfo = {
                hostname: "sts-fips.us-east-2.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-2",
            };
            break;
        case "us-gov-east-1":
            regionInfo = {
                hostname: "sts.us-gov-east-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "us-gov-east-1-fips":
            regionInfo = {
                hostname: "sts.us-gov-east-1.amazonaws.com",
                partition: "aws-us-gov",
                signingRegion: "us-gov-east-1",
            };
            break;
        case "us-gov-west-1":
            regionInfo = {
                hostname: "sts.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "us-gov-west-1-fips":
            regionInfo = {
                hostname: "sts.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
                signingRegion: "us-gov-west-1",
            };
            break;
        case "us-iso-east-1":
            regionInfo = {
                hostname: "sts.us-iso-east-1.c2s.ic.gov",
                partition: "aws-iso",
            };
            break;
        case "us-isob-east-1":
            regionInfo = {
                hostname: "sts.us-isob-east-1.sc2s.sgov.gov",
                partition: "aws-iso-b",
            };
            break;
        case "us-west-1":
            regionInfo = {
                hostname: "sts.us-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-west-1-fips":
            regionInfo = {
                hostname: "sts-fips.us-west-1.amazonaws.com",
                partition: "aws",
                signingRegion: "us-west-1",
            };
            break;
        case "us-west-2":
            regionInfo = {
                hostname: "sts.us-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-west-2-fips":
            regionInfo = {
                hostname: "sts-fips.us-west-2.amazonaws.com",
                partition: "aws",
                signingRegion: "us-west-2",
            };
            break;
        default:
            if (AWS_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_TEMPLATE.replace("{region}", region),
                    partition: "aws",
                };
            }
            if (AWS_CN_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_CN_TEMPLATE.replace("{region}", region),
                    partition: "aws-cn",
                };
            }
            if (AWS_ISO_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_ISO_TEMPLATE.replace("{region}", region),
                    partition: "aws-iso",
                };
            }
            if (AWS_ISO_B_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_ISO_B_TEMPLATE.replace("{region}", region),
                    partition: "aws-iso-b",
                };
            }
            if (AWS_US_GOV_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_US_GOV_TEMPLATE.replace("{region}", region),
                    partition: "aws-us-gov",
                };
            }
            if (regionInfo === undefined) {
                regionInfo = {
                    hostname: AWS_TEMPLATE.replace("{region}", region),
                    partition: "aws",
                };
            }
    }
    return Promise.resolve({ signingService: "sts", ...regionInfo });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZW5kcG9pbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUdBLE1BQU0sWUFBWSxHQUFHLDRCQUE0QixDQUFDO0FBQ2xELE1BQU0sZUFBZSxHQUFHLCtCQUErQixDQUFDO0FBQ3hELE1BQU0sZ0JBQWdCLEdBQUcseUJBQXlCLENBQUM7QUFDbkQsTUFBTSxrQkFBa0IsR0FBRyw0QkFBNEIsQ0FBQztBQUN4RCxNQUFNLG1CQUFtQixHQUFHLDRCQUE0QixDQUFDO0FBR3pELE1BQU0sV0FBVyxHQUFHLElBQUksR0FBRyxDQUFDO0lBQzFCLFlBQVk7SUFDWixXQUFXO0lBQ1gsZ0JBQWdCO0lBQ2hCLGdCQUFnQjtJQUNoQixnQkFBZ0I7SUFDaEIsWUFBWTtJQUNaLGdCQUFnQjtJQUNoQixnQkFBZ0I7SUFDaEIsY0FBYztJQUNkLGNBQWM7SUFDZCxZQUFZO0lBQ1osWUFBWTtJQUNaLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztJQUNYLFlBQVk7SUFDWixXQUFXO0lBQ1gsV0FBVztJQUNYLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztDQUNaLENBQUMsQ0FBQztBQUNILE1BQU0sY0FBYyxHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsWUFBWSxFQUFFLGdCQUFnQixDQUFDLENBQUMsQ0FBQztBQUNqRSxNQUFNLGVBQWUsR0FBRyxJQUFJLEdBQUcsQ0FBQyxDQUFDLGVBQWUsQ0FBQyxDQUFDLENBQUM7QUFDbkQsTUFBTSxpQkFBaUIsR0FBRyxJQUFJLEdBQUcsQ0FBQyxDQUFDLGdCQUFnQixDQUFDLENBQUMsQ0FBQztBQUN0RCxNQUFNLGtCQUFrQixHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsZUFBZSxFQUFFLGVBQWUsQ0FBQyxDQUFDLENBQUM7QUFFdkUsTUFBTSxDQUFDLE1BQU0seUJBQXlCLEdBQXVCLENBQUMsTUFBYyxFQUFFLE9BQWEsRUFBRSxFQUFFO0lBQzdGLElBQUksVUFBVSxH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