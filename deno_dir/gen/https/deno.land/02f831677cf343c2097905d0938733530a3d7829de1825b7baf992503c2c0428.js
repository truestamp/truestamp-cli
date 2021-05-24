import { DOT_PATTERN, getArnResources, getSuffix, getSuffixForArnEndpoint, isBucketNameOptions, isDnsCompatibleBucketName, validateAccountId, validateArnEndpointOptions, validateDNSHostLabel, validateNoDualstack, validateNoFIPS, validateOutpostService, validatePartition, validateRegion, validateS3Service, validateService, } from "./bucketHostnameUtils.ts";
export const bucketHostname = (options) => {
    const { isCustomEndpoint, baseHostname, dualstackEndpoint, accelerateEndpoint } = options;
    if (isCustomEndpoint) {
        if (dualstackEndpoint)
            throw new Error("Dualstack endpoint is not supported with custom endpoint");
        if (accelerateEndpoint)
            throw new Error("Accelerate endpoint is not supported with custom endpoint");
    }
    return isBucketNameOptions(options)
        ?
            getEndpointFromBucketName({ ...options, isCustomEndpoint })
        :
            getEndpointFromArn({ ...options, isCustomEndpoint });
};
const getEndpointFromArn = (options) => {
    const { isCustomEndpoint, baseHostname } = options;
    const [clientRegion, hostnameSuffix] = isCustomEndpoint
        ? [options.clientRegion, baseHostname]
        :
            getSuffixForArnEndpoint(baseHostname);
    const { pathStyleEndpoint, dualstackEndpoint = false, accelerateEndpoint = false, tlsCompatible = true, useArnRegion, bucketName, clientPartition = "aws", clientSigningRegion = clientRegion, } = options;
    validateArnEndpointOptions({ pathStyleEndpoint, accelerateEndpoint, tlsCompatible });
    const { service, partition, accountId, region, resource } = bucketName;
    validateService(service);
    validatePartition(partition, { clientPartition });
    validateAccountId(accountId);
    validateRegion(region, { useArnRegion, clientRegion, clientSigningRegion });
    const { accesspointName, outpostId } = getArnResources(resource);
    const DNSHostLabel = `${accesspointName}-${accountId}`;
    validateDNSHostLabel(DNSHostLabel, { tlsCompatible });
    const endpointRegion = useArnRegion ? region : clientRegion;
    const signingRegion = useArnRegion ? region : clientSigningRegion;
    if (service === "s3-object-lambda") {
        validateNoDualstack(dualstackEndpoint);
        return {
            bucketEndpoint: true,
            hostname: `${DNSHostLabel}.${service}.${endpointRegion}.${hostnameSuffix}`,
            signingRegion,
            signingService: service,
        };
    }
    else if (outpostId) {
        validateOutpostService(service);
        validateDNSHostLabel(outpostId, { tlsCompatible });
        validateNoDualstack(dualstackEndpoint);
        validateNoFIPS(endpointRegion);
        const hostnamePrefix = `${DNSHostLabel}.${outpostId}`;
        return {
            bucketEndpoint: true,
            hostname: `${hostnamePrefix}${isCustomEndpoint ? "" : `.s3-outposts.${endpointRegion}`}.${hostnameSuffix}`,
            signingRegion,
            signingService: "s3-outposts",
        };
    }
    validateS3Service(service);
    const hostnamePrefix = `${DNSHostLabel}`;
    return {
        bucketEndpoint: true,
        hostname: `${hostnamePrefix}${isCustomEndpoint ? "" : `.s3-accesspoint${dualstackEndpoint ? ".dualstack" : ""}.${endpointRegion}`}.${hostnameSuffix}`,
        signingRegion,
    };
};
const getEndpointFromBucketName = ({ accelerateEndpoint = false, clientRegion: region, baseHostname, bucketName, dualstackEndpoint = false, pathStyleEndpoint = false, tlsCompatible = true, isCustomEndpoint = false, }) => {
    const [clientRegion, hostnameSuffix] = isCustomEndpoint ? [region, baseHostname] : getSuffix(baseHostname);
    if (pathStyleEndpoint || !isDnsCompatibleBucketName(bucketName) || (tlsCompatible && DOT_PATTERN.test(bucketName))) {
        return {
            bucketEndpoint: false,
            hostname: dualstackEndpoint ? `s3.dualstack.${clientRegion}.${hostnameSuffix}` : baseHostname,
        };
    }
    if (accelerateEndpoint) {
        baseHostname = `s3-accelerate${dualstackEndpoint ? ".dualstack" : ""}.${hostnameSuffix}`;
    }
    else if (dualstackEndpoint) {
        baseHostname = `s3.dualstack.${clientRegion}.${hostnameSuffix}`;
    }
    return {
        bucketEndpoint: true,
        hostname: `${bucketName}.${baseHostname}`,
    };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYnVja2V0SG9zdG5hbWUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJidWNrZXRIb3N0bmFtZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBR0wsV0FBVyxFQUNYLGVBQWUsRUFDZixTQUFTLEVBQ1QsdUJBQXVCLEVBQ3ZCLG1CQUFtQixFQUNuQix5QkFBeUIsRUFDekIsaUJBQWlCLEVBQ2pCLDBCQUEwQixFQUMxQixvQkFBb0IsRUFDcEIsbUJBQW1CLEVBQ25CLGNBQWMsRUFDZCxzQkFBc0IsRUFDdEIsaUJBQWlCLEVBQ2pCLGNBQWMsRUFDZCxpQkFBaUIsRUFDakIsZUFBZSxHQUNoQixNQUFNLDBCQUEwQixDQUFDO0FBU2xDLE1BQU0sQ0FBQyxNQUFNLGNBQWMsR0FBRyxDQUFDLE9BQWlELEVBQWtCLEVBQUU7SUFDbEcsTUFBTSxFQUFFLGdCQUFnQixFQUFFLFlBQVksRUFBRSxpQkFBaUIsRUFBRSxrQkFBa0IsRUFBRSxHQUFHLE9BQU8sQ0FBQztJQUUxRixJQUFJLGdCQUFnQixFQUFFO1FBQ3BCLElBQUksaUJBQWlCO1lBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQywwREFBMEQsQ0FBQyxDQUFDO1FBQ25HLElBQUksa0JBQWtCO1lBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQywyREFBMkQsQ0FBQyxDQUFDO0tBQ3RHO0lBRUQsT0FBTyxtQkFBbUIsQ0FBQyxPQUFPLENBQUM7UUFDakMsQ0FBQztZQUNDLHlCQUF5QixDQUFDLEVBQUUsR0FBRyxPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsQ0FBQztRQUM3RCxDQUFDO1lBQ0Msa0JBQWtCLENBQUMsRUFBRSxHQUFHLE9BQU8sRUFBRSxnQkFBZ0IsRUFBRSxDQUFDLENBQUM7QUFDM0QsQ0FBQyxDQUFDO0FBRUYsTUFBTSxrQkFBa0IsR0FBRyxDQUFDLE9BQTBELEVBQWtCLEVBQUU7SUFDeEcsTUFBTSxFQUFFLGdCQUFnQixFQUFFLFlBQVksRUFBRSxHQUFHLE9BQU8sQ0FBQztJQUNuRCxN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