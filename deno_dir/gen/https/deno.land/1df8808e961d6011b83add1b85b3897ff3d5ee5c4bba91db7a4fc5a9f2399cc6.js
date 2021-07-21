const AWS_TEMPLATE = "s3.{region}.amazonaws.com";
const AWS_CN_TEMPLATE = "s3.{region}.amazonaws.com.cn";
const AWS_ISO_TEMPLATE = "s3.{region}.c2s.ic.gov";
const AWS_ISO_B_TEMPLATE = "s3.{region}.sc2s.sgov.gov";
const AWS_US_GOV_TEMPLATE = "s3.{region}.amazonaws.com";
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
        case "accesspoint-af-south-1":
            regionInfo = {
                hostname: "s3-accesspoint.af-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-east-1":
            regionInfo = {
                hostname: "s3-accesspoint.ap-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-northeast-1":
            regionInfo = {
                hostname: "s3-accesspoint.ap-northeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-northeast-2":
            regionInfo = {
                hostname: "s3-accesspoint.ap-northeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-northeast-3":
            regionInfo = {
                hostname: "s3-accesspoint.ap-northeast-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-south-1":
            regionInfo = {
                hostname: "s3-accesspoint.ap-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-southeast-1":
            regionInfo = {
                hostname: "s3-accesspoint.ap-southeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ap-southeast-2":
            regionInfo = {
                hostname: "s3-accesspoint.ap-southeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-ca-central-1":
            regionInfo = {
                hostname: "s3-accesspoint.ca-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-cn-north-1":
            regionInfo = {
                hostname: "s3-accesspoint.cn-north-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "accesspoint-cn-northwest-1":
            regionInfo = {
                hostname: "s3-accesspoint.cn-northwest-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "accesspoint-eu-central-1":
            regionInfo = {
                hostname: "s3-accesspoint.eu-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-eu-north-1":
            regionInfo = {
                hostname: "s3-accesspoint.eu-north-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-eu-south-1":
            regionInfo = {
                hostname: "s3-accesspoint.eu-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-eu-west-1":
            regionInfo = {
                hostname: "s3-accesspoint.eu-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-eu-west-2":
            regionInfo = {
                hostname: "s3-accesspoint.eu-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-eu-west-3":
            regionInfo = {
                hostname: "s3-accesspoint.eu-west-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-me-south-1":
            regionInfo = {
                hostname: "s3-accesspoint.me-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-sa-east-1":
            regionInfo = {
                hostname: "s3-accesspoint.sa-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-us-east-1":
            regionInfo = {
                hostname: "s3-accesspoint.us-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-us-east-2":
            regionInfo = {
                hostname: "s3-accesspoint.us-east-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-us-gov-east-1":
            regionInfo = {
                hostname: "s3-accesspoint.us-gov-east-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "accesspoint-us-gov-west-1":
            regionInfo = {
                hostname: "s3-accesspoint.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "accesspoint-us-west-1":
            regionInfo = {
                hostname: "s3-accesspoint.us-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "accesspoint-us-west-2":
            regionInfo = {
                hostname: "s3-accesspoint.us-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "af-south-1":
            regionInfo = {
                hostname: "s3.af-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-east-1":
            regionInfo = {
                hostname: "s3.ap-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-1":
            regionInfo = {
                hostname: "s3.ap-northeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-2":
            regionInfo = {
                hostname: "s3.ap-northeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-3":
            regionInfo = {
                hostname: "s3.ap-northeast-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-south-1":
            regionInfo = {
                hostname: "s3.ap-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-southeast-1":
            regionInfo = {
                hostname: "s3.ap-southeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-southeast-2":
            regionInfo = {
                hostname: "s3.ap-southeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "aws-global":
            regionInfo = {
                hostname: "s3.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "ca-central-1":
            regionInfo = {
                hostname: "s3.ca-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "cn-north-1":
            regionInfo = {
                hostname: "s3.cn-north-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "cn-northwest-1":
            regionInfo = {
                hostname: "s3.cn-northwest-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "eu-central-1":
            regionInfo = {
                hostname: "s3.eu-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-north-1":
            regionInfo = {
                hostname: "s3.eu-north-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-south-1":
            regionInfo = {
                hostname: "s3.eu-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-1":
            regionInfo = {
                hostname: "s3.eu-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-2":
            regionInfo = {
                hostname: "s3.eu-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-3":
            regionInfo = {
                hostname: "s3.eu-west-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-accesspoint-ca-central-1":
            regionInfo = {
                hostname: "s3-accesspoint-fips.ca-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-accesspoint-us-east-1":
            regionInfo = {
                hostname: "s3-accesspoint-fips.us-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-accesspoint-us-east-2":
            regionInfo = {
                hostname: "s3-accesspoint-fips.us-east-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-accesspoint-us-gov-east-1":
            regionInfo = {
                hostname: "s3-accesspoint-fips.us-gov-east-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "fips-accesspoint-us-gov-west-1":
            regionInfo = {
                hostname: "s3-accesspoint-fips.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "fips-accesspoint-us-west-1":
            regionInfo = {
                hostname: "s3-accesspoint-fips.us-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-accesspoint-us-west-2":
            regionInfo = {
                hostname: "s3-accesspoint-fips.us-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-us-gov-west-1":
            regionInfo = {
                hostname: "s3-fips.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
                signingRegion: "us-gov-west-1",
            };
            break;
        case "me-south-1":
            regionInfo = {
                hostname: "s3.me-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "s3-external-1":
            regionInfo = {
                hostname: "s3-external-1.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "sa-east-1":
            regionInfo = {
                hostname: "s3.sa-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-1":
            regionInfo = {
                hostname: "s3.us-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-2":
            regionInfo = {
                hostname: "s3.us-east-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-gov-east-1":
            regionInfo = {
                hostname: "s3.us-gov-east-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "us-gov-west-1":
            regionInfo = {
                hostname: "s3.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "us-iso-east-1":
            regionInfo = {
                hostname: "s3.us-iso-east-1.c2s.ic.gov",
                partition: "aws-iso",
            };
            break;
        case "us-isob-east-1":
            regionInfo = {
                hostname: "s3.us-isob-east-1.sc2s.sgov.gov",
                partition: "aws-iso-b",
            };
            break;
        case "us-west-1":
            regionInfo = {
                hostname: "s3.us-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-west-2":
            regionInfo = {
                hostname: "s3.us-west-2.amazonaws.com",
                partition: "aws",
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
    return Promise.resolve({ signingService: "s3", ...regionInfo });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZW5kcG9pbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUdBLE1BQU0sWUFBWSxHQUFHLDJCQUEyQixDQUFDO0FBQ2pELE1BQU0sZUFBZSxHQUFHLDhCQUE4QixDQUFDO0FBQ3ZELE1BQU0sZ0JBQWdCLEdBQUcsd0JBQXdCLENBQUM7QUFDbEQsTUFBTSxrQkFBa0IsR0FBRywyQkFBMkIsQ0FBQztBQUN2RCxNQUFNLG1CQUFtQixHQUFHLDJCQUEyQixDQUFDO0FBR3hELE1BQU0sV0FBVyxHQUFHLElBQUksR0FBRyxDQUFDO0lBQzFCLFlBQVk7SUFDWixXQUFXO0lBQ1gsZ0JBQWdCO0lBQ2hCLGdCQUFnQjtJQUNoQixnQkFBZ0I7SUFDaEIsWUFBWTtJQUNaLGdCQUFnQjtJQUNoQixnQkFBZ0I7SUFDaEIsY0FBYztJQUNkLGNBQWM7SUFDZCxZQUFZO0lBQ1osWUFBWTtJQUNaLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztJQUNYLFlBQVk7SUFDWixXQUFXO0lBQ1gsV0FBVztJQUNYLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztDQUNaLENBQUMsQ0FBQztBQUNILE1BQU0sY0FBYyxHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsWUFBWSxFQUFFLGdCQUFnQixDQUFDLENBQUMsQ0FBQztBQUNqRSxNQUFNLGVBQWUsR0FBRyxJQUFJLEdBQUcsQ0FBQyxDQUFDLGVBQWUsQ0FBQyxDQUFDLENBQUM7QUFDbkQsTUFBTSxpQkFBaUIsR0FBRyxJQUFJLEdBQUcsQ0FBQyxDQUFDLGdCQUFnQixDQUFDLENBQUMsQ0FBQztBQUN0RCxNQUFNLGtCQUFrQixHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsZUFBZSxFQUFFLGVBQWUsQ0FBQyxDQUFDLENBQUM7QUFFdkUsTUFBTSxDQUFDLE1BQU0seUJBQXlCLEdBQXVCLENBQUMsTUFBYyxFQUFFLE9BQWEsRUFBRSxFQUFFO0lBQzdGLElBQUksVUFBVSxH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