import { HttpRequest as __HttpRequest } from "../../protocol-http/mod.ts";
import { extendedEncodeURIComponent as __extendedEncodeURIComponent, getValueFromTextNode as __getValueFromTextNode, } from "../../smithy-client/mod.ts";
import { parse as xmlParse } from "https://jspm.dev/fast-xml-parser";
export const serializeAws_queryAssumeRoleCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryAssumeRoleRequest(input, context),
        Action: "AssumeRole",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryAssumeRoleWithSAMLCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryAssumeRoleWithSAMLRequest(input, context),
        Action: "AssumeRoleWithSAML",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryAssumeRoleWithWebIdentityCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryAssumeRoleWithWebIdentityRequest(input, context),
        Action: "AssumeRoleWithWebIdentity",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryDecodeAuthorizationMessageCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryDecodeAuthorizationMessageRequest(input, context),
        Action: "DecodeAuthorizationMessage",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetAccessKeyInfoCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetAccessKeyInfoRequest(input, context),
        Action: "GetAccessKeyInfo",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetCallerIdentityCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetCallerIdentityRequest(input, context),
        Action: "GetCallerIdentity",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetFederationTokenCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetFederationTokenRequest(input, context),
        Action: "GetFederationToken",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetSessionTokenCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetSessionTokenRequest(input, context),
        Action: "GetSessionToken",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const deserializeAws_queryAssumeRoleCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryAssumeRoleCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryAssumeRoleResponse(data.AssumeRoleResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryAssumeRoleCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ExpiredTokenException":
        case "com.amazonaws.sts#ExpiredTokenException":
            response = {
                ...(await deserializeAws_queryExpiredTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryAssumeRoleWithSAMLCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryAssumeRoleWithSAMLCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryAssumeRoleWithSAMLResponse(data.AssumeRoleWithSAMLResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryAssumeRoleWithSAMLCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ExpiredTokenException":
        case "com.amazonaws.sts#ExpiredTokenException":
            response = {
                ...(await deserializeAws_queryExpiredTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "IDPRejectedClaimException":
        case "com.amazonaws.sts#IDPRejectedClaimException":
            response = {
                ...(await deserializeAws_queryIDPRejectedClaimExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "InvalidIdentityTokenException":
        case "com.amazonaws.sts#InvalidIdentityTokenException":
            response = {
                ...(await deserializeAws_queryInvalidIdentityTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryAssumeRoleWithWebIdentityCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryAssumeRoleWithWebIdentityCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryAssumeRoleWithWebIdentityResponse(data.AssumeRoleWithWebIdentityResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryAssumeRoleWithWebIdentityCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ExpiredTokenException":
        case "com.amazonaws.sts#ExpiredTokenException":
            response = {
                ...(await deserializeAws_queryExpiredTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "IDPCommunicationErrorException":
        case "com.amazonaws.sts#IDPCommunicationErrorException":
            response = {
                ...(await deserializeAws_queryIDPCommunicationErrorExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "IDPRejectedClaimException":
        case "com.amazonaws.sts#IDPRejectedClaimException":
            response = {
                ...(await deserializeAws_queryIDPRejectedClaimExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "InvalidIdentityTokenException":
        case "com.amazonaws.sts#InvalidIdentityTokenException":
            response = {
                ...(await deserializeAws_queryInvalidIdentityTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryDecodeAuthorizationMessageCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryDecodeAuthorizationMessageCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryDecodeAuthorizationMessageResponse(data.DecodeAuthorizationMessageResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryDecodeAuthorizationMessageCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidAuthorizationMessageException":
        case "com.amazonaws.sts#InvalidAuthorizationMessageException":
            response = {
                ...(await deserializeAws_queryInvalidAuthorizationMessageExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetAccessKeyInfoCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetAccessKeyInfoCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetAccessKeyInfoResponse(data.GetAccessKeyInfoResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetAccessKeyInfoCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetCallerIdentityCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetCallerIdentityCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetCallerIdentityResponse(data.GetCallerIdentityResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetCallerIdentityCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetFederationTokenCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetFederationTokenCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetFederationTokenResponse(data.GetFederationTokenResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetFederationTokenCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetSessionTokenCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetSessionTokenCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetSessionTokenResponse(data.GetSessionTokenResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetSessionTokenCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
const deserializeAws_queryExpiredTokenExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryExpiredTokenException(body.Error, context);
    const contents = {
        name: "ExpiredTokenException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryIDPCommunicationErrorExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryIDPCommunicationErrorException(body.Error, context);
    const contents = {
        name: "IDPCommunicationErrorException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryIDPRejectedClaimExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryIDPRejectedClaimException(body.Error, context);
    const contents = {
        name: "IDPRejectedClaimException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryInvalidAuthorizationMessageExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryInvalidAuthorizationMessageException(body.Error, context);
    const contents = {
        name: "InvalidAuthorizationMessageException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryInvalidIdentityTokenExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryInvalidIdentityTokenException(body.Error, context);
    const contents = {
        name: "InvalidIdentityTokenException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryMalformedPolicyDocumentExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryMalformedPolicyDocumentException(body.Error, context);
    const contents = {
        name: "MalformedPolicyDocumentException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryPackedPolicyTooLargeExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryPackedPolicyTooLargeException(body.Error, context);
    const contents = {
        name: "PackedPolicyTooLargeException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryRegionDisabledExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryRegionDisabledException(body.Error, context);
    const contents = {
        name: "RegionDisabledException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const serializeAws_queryAssumeRoleRequest = (input, context) => {
    const entries = {};
    if (input.RoleArn !== undefined && input.RoleArn !== null) {
        entries["RoleArn"] = input.RoleArn;
    }
    if (input.RoleSessionName !== undefined && input.RoleSessionName !== null) {
        entries["RoleSessionName"] = input.RoleSessionName;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const memberEntries = serializeAws_querytagListType(input.Tags, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `Tags.${key}`;
            entries[loc] = value;
        });
    }
    if (input.TransitiveTagKeys !== undefined && input.TransitiveTagKeys !== null) {
        const memberEntries = serializeAws_querytagKeyListType(input.TransitiveTagKeys, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `TransitiveTagKeys.${key}`;
            entries[loc] = value;
        });
    }
    if (input.ExternalId !== undefined && input.ExternalId !== null) {
        entries["ExternalId"] = input.ExternalId;
    }
    if (input.SerialNumber !== undefined && input.SerialNumber !== null) {
        entries["SerialNumber"] = input.SerialNumber;
    }
    if (input.TokenCode !== undefined && input.TokenCode !== null) {
        entries["TokenCode"] = input.TokenCode;
    }
    if (input.SourceIdentity !== undefined && input.SourceIdentity !== null) {
        entries["SourceIdentity"] = input.SourceIdentity;
    }
    return entries;
};
const serializeAws_queryAssumeRoleWithSAMLRequest = (input, context) => {
    const entries = {};
    if (input.RoleArn !== undefined && input.RoleArn !== null) {
        entries["RoleArn"] = input.RoleArn;
    }
    if (input.PrincipalArn !== undefined && input.PrincipalArn !== null) {
        entries["PrincipalArn"] = input.PrincipalArn;
    }
    if (input.SAMLAssertion !== undefined && input.SAMLAssertion !== null) {
        entries["SAMLAssertion"] = input.SAMLAssertion;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    return entries;
};
const serializeAws_queryAssumeRoleWithWebIdentityRequest = (input, context) => {
    const entries = {};
    if (input.RoleArn !== undefined && input.RoleArn !== null) {
        entries["RoleArn"] = input.RoleArn;
    }
    if (input.RoleSessionName !== undefined && input.RoleSessionName !== null) {
        entries["RoleSessionName"] = input.RoleSessionName;
    }
    if (input.WebIdentityToken !== undefined && input.WebIdentityToken !== null) {
        entries["WebIdentityToken"] = input.WebIdentityToken;
    }
    if (input.ProviderId !== undefined && input.ProviderId !== null) {
        entries["ProviderId"] = input.ProviderId;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    return entries;
};
const serializeAws_queryDecodeAuthorizationMessageRequest = (input, context) => {
    const entries = {};
    if (input.EncodedMessage !== undefined && input.EncodedMessage !== null) {
        entries["EncodedMessage"] = input.EncodedMessage;
    }
    return entries;
};
const serializeAws_queryGetAccessKeyInfoRequest = (input, context) => {
    const entries = {};
    if (input.AccessKeyId !== undefined && input.AccessKeyId !== null) {
        entries["AccessKeyId"] = input.AccessKeyId;
    }
    return entries;
};
const serializeAws_queryGetCallerIdentityRequest = (input, context) => {
    const entries = {};
    return entries;
};
const serializeAws_queryGetFederationTokenRequest = (input, context) => {
    const entries = {};
    if (input.Name !== undefined && input.Name !== null) {
        entries["Name"] = input.Name;
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const memberEntries = serializeAws_querytagListType(input.Tags, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `Tags.${key}`;
            entries[loc] = value;
        });
    }
    return entries;
};
const serializeAws_queryGetSessionTokenRequest = (input, context) => {
    const entries = {};
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    if (input.SerialNumber !== undefined && input.SerialNumber !== null) {
        entries["SerialNumber"] = input.SerialNumber;
    }
    if (input.TokenCode !== undefined && input.TokenCode !== null) {
        entries["TokenCode"] = input.TokenCode;
    }
    return entries;
};
const serializeAws_querypolicyDescriptorListType = (input, context) => {
    const entries = {};
    let counter = 1;
    for (let entry of input) {
        if (entry === null) {
            continue;
        }
        const memberEntries = serializeAws_queryPolicyDescriptorType(entry, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            entries[`member.${counter}.${key}`] = value;
        });
        counter++;
    }
    return entries;
};
const serializeAws_queryPolicyDescriptorType = (input, context) => {
    const entries = {};
    if (input.arn !== undefined && input.arn !== null) {
        entries["arn"] = input.arn;
    }
    return entries;
};
const serializeAws_queryTag = (input, context) => {
    const entries = {};
    if (input.Key !== undefined && input.Key !== null) {
        entries["Key"] = input.Key;
    }
    if (input.Value !== undefined && input.Value !== null) {
        entries["Value"] = input.Value;
    }
    return entries;
};
const serializeAws_querytagKeyListType = (input, context) => {
    const entries = {};
    let counter = 1;
    for (let entry of input) {
        if (entry === null) {
            continue;
        }
        entries[`member.${counter}`] = entry;
        counter++;
    }
    return entries;
};
const serializeAws_querytagListType = (input, context) => {
    const entries = {};
    let counter = 1;
    for (let entry of input) {
        if (entry === null) {
            continue;
        }
        const memberEntries = serializeAws_queryTag(entry, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            entries[`member.${counter}.${key}`] = value;
        });
        counter++;
    }
    return entries;
};
const deserializeAws_queryAssumedRoleUser = (output, context) => {
    let contents = {
        AssumedRoleId: undefined,
        Arn: undefined,
    };
    if (output["AssumedRoleId"] !== undefined) {
        contents.AssumedRoleId = output["AssumedRoleId"];
    }
    if (output["Arn"] !== undefined) {
        contents.Arn = output["Arn"];
    }
    return contents;
};
const deserializeAws_queryAssumeRoleResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        AssumedRoleUser: undefined,
        PackedPolicySize: undefined,
        SourceIdentity: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["AssumedRoleUser"] !== undefined) {
        contents.AssumedRoleUser = deserializeAws_queryAssumedRoleUser(output["AssumedRoleUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    if (output["SourceIdentity"] !== undefined) {
        contents.SourceIdentity = output["SourceIdentity"];
    }
    return contents;
};
const deserializeAws_queryAssumeRoleWithSAMLResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        AssumedRoleUser: undefined,
        PackedPolicySize: undefined,
        Subject: undefined,
        SubjectType: undefined,
        Issuer: undefined,
        Audience: undefined,
        NameQualifier: undefined,
        SourceIdentity: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["AssumedRoleUser"] !== undefined) {
        contents.AssumedRoleUser = deserializeAws_queryAssumedRoleUser(output["AssumedRoleUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    if (output["Subject"] !== undefined) {
        contents.Subject = output["Subject"];
    }
    if (output["SubjectType"] !== undefined) {
        contents.SubjectType = output["SubjectType"];
    }
    if (output["Issuer"] !== undefined) {
        contents.Issuer = output["Issuer"];
    }
    if (output["Audience"] !== undefined) {
        contents.Audience = output["Audience"];
    }
    if (output["NameQualifier"] !== undefined) {
        contents.NameQualifier = output["NameQualifier"];
    }
    if (output["SourceIdentity"] !== undefined) {
        contents.SourceIdentity = output["SourceIdentity"];
    }
    return contents;
};
const deserializeAws_queryAssumeRoleWithWebIdentityResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        SubjectFromWebIdentityToken: undefined,
        AssumedRoleUser: undefined,
        PackedPolicySize: undefined,
        Provider: undefined,
        Audience: undefined,
        SourceIdentity: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["SubjectFromWebIdentityToken"] !== undefined) {
        contents.SubjectFromWebIdentityToken = output["SubjectFromWebIdentityToken"];
    }
    if (output["AssumedRoleUser"] !== undefined) {
        contents.AssumedRoleUser = deserializeAws_queryAssumedRoleUser(output["AssumedRoleUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    if (output["Provider"] !== undefined) {
        contents.Provider = output["Provider"];
    }
    if (output["Audience"] !== undefined) {
        contents.Audience = output["Audience"];
    }
    if (output["SourceIdentity"] !== undefined) {
        contents.SourceIdentity = output["SourceIdentity"];
    }
    return contents;
};
const deserializeAws_queryCredentials = (output, context) => {
    let contents = {
        AccessKeyId: undefined,
        SecretAccessKey: undefined,
        SessionToken: undefined,
        Expiration: undefined,
    };
    if (output["AccessKeyId"] !== undefined) {
        contents.AccessKeyId = output["AccessKeyId"];
    }
    if (output["SecretAccessKey"] !== undefined) {
        contents.SecretAccessKey = output["SecretAccessKey"];
    }
    if (output["SessionToken"] !== undefined) {
        contents.SessionToken = output["SessionToken"];
    }
    if (output["Expiration"] !== undefined) {
        contents.Expiration = new Date(output["Expiration"]);
    }
    return contents;
};
const deserializeAws_queryDecodeAuthorizationMessageResponse = (output, context) => {
    let contents = {
        DecodedMessage: undefined,
    };
    if (output["DecodedMessage"] !== undefined) {
        contents.DecodedMessage = output["DecodedMessage"];
    }
    return contents;
};
const deserializeAws_queryExpiredTokenException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryFederatedUser = (output, context) => {
    let contents = {
        FederatedUserId: undefined,
        Arn: undefined,
    };
    if (output["FederatedUserId"] !== undefined) {
        contents.FederatedUserId = output["FederatedUserId"];
    }
    if (output["Arn"] !== undefined) {
        contents.Arn = output["Arn"];
    }
    return contents;
};
const deserializeAws_queryGetAccessKeyInfoResponse = (output, context) => {
    let contents = {
        Account: undefined,
    };
    if (output["Account"] !== undefined) {
        contents.Account = output["Account"];
    }
    return contents;
};
const deserializeAws_queryGetCallerIdentityResponse = (output, context) => {
    let contents = {
        UserId: undefined,
        Account: undefined,
        Arn: undefined,
    };
    if (output["UserId"] !== undefined) {
        contents.UserId = output["UserId"];
    }
    if (output["Account"] !== undefined) {
        contents.Account = output["Account"];
    }
    if (output["Arn"] !== undefined) {
        contents.Arn = output["Arn"];
    }
    return contents;
};
const deserializeAws_queryGetFederationTokenResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        FederatedUser: undefined,
        PackedPolicySize: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["FederatedUser"] !== undefined) {
        contents.FederatedUser = deserializeAws_queryFederatedUser(output["FederatedUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    return contents;
};
const deserializeAws_queryGetSessionTokenResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    return contents;
};
const deserializeAws_queryIDPCommunicationErrorException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryIDPRejectedClaimException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryInvalidAuthorizationMessageException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryInvalidIdentityTokenException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryMalformedPolicyDocumentException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryPackedPolicyTooLargeException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeAws_queryRegionDisabledException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = output["message"];
    }
    return contents;
};
const deserializeMetadata = (output) => ({
    httpStatusCode: output.statusCode,
    requestId: output.headers["x-amzn-requestid"] ?? output.headers["x-amzn-request-id"],
    extendedRequestId: output.headers["x-amz-id-2"],
    cfId: output.headers["x-amz-cf-id"],
});
const collectBody = (streamBody = new Uint8Array(), context) => {
    if (streamBody instanceof Uint8Array) {
        return Promise.resolve(streamBody);
    }
    return context.streamCollector(streamBody) || Promise.resolve(new Uint8Array());
};
const collectBodyString = (streamBody, context) => collectBody(streamBody, context).then((body) => context.utf8Encoder(body));
const buildHttpRpcRequest = async (context, headers, path, resolvedHostname, body) => {
    const { hostname, protocol = "https", port } = await context.endpoint();
    const contents = {
        protocol,
        hostname,
        port,
        method: "POST",
        path,
        headers,
    };
    if (resolvedHostname !== undefined) {
        contents.hostname = resolvedHostname;
    }
    if (body !== undefined) {
        contents.body = body;
    }
    return new __HttpRequest(contents);
};
const decodeEscapedXML = (str) => str
    .replace(/&amp;/g, "&")
    .replace(/&apos;/g, "'")
    .replace(/&quot;/g, '"')
    .replace(/&gt;/g, ">")
    .replace(/&lt;/g, "<");
const parseBody = (streamBody, context) => collectBodyString(streamBody, context).then((encoded) => {
    if (encoded.length) {
        const parsedObj = xmlParse(encoded, {
            attributeNamePrefix: "",
            ignoreAttributes: false,
            parseNodeValue: false,
            trimValues: false,
            tagValueProcessor: (val) => (val.trim() === "" ? "" : decodeEscapedXML(val)),
        });
        const textNodeName = "#text";
        const key = Object.keys(parsedObj)[0];
        const parsedObjToReturn = parsedObj[key];
        if (parsedObjToReturn[textNodeName]) {
            parsedObjToReturn[key] = parsedObjToReturn[textNodeName];
            delete parsedObjToReturn[textNodeName];
        }
        return __getValueFromTextNode(parsedObjToReturn);
    }
    return {};
});
const buildFormUrlencodedString = (formEntries) => Object.entries(formEntries)
    .map(([key, value]) => __extendedEncodeURIComponent(key) + "=" + __extendedEncodeURIComponent(value))
    .join("&");
const loadQueryErrorCode = (output, data) => {
    if (data.Error.Code !== undefined) {
        return data.Error.Code;
    }
    if (output.statusCode == 404) {
        return "NotFound";
    }
    return "";
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQXdzX3F1ZXJ5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiQXdzX3F1ZXJ5LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQTZDQSxPQUFPLEVBQUUsV0FBVyxJQUFJLGFBQWEsRUFBa0MsTUFBTSw0QkFBNEIsQ0FBQztBQUMxRyxPQUFPLEVBRUwsMEJBQTBCLElBQUksNEJBQTRCLEVBQzFELG9CQUFvQixJQUFJLHNCQUFzQixHQUMvQyxNQUFNLDRCQUE0QixDQUFDO0FBUXBDLE9BQU8sRUFBRSxLQUFLLElBQUksUUFBUSxFQUFFLE1BQU0sa0NBQWtDLENBQUM7QUFFckUsTUFBTSxDQUFDLE1BQU0sbUNBQW1DLEdBQUcsS0FBSyxFQUN0RCxLQUE2QixFQUM3QixPQUF1QixFQUNDLEVBQUU7SUFDMUIsTUFBTSxPQUFPLEdBQWdCO1FBQzNCLGNBQWMsRUFBRSxtQ0FBbUM7S0FDcEQsQ0FBQztJQUNGLElBQUksSUFBUyxDQUFDO0lBQ2QsSUFBSSxHQUFHLHlCQUF5QixDQUFDO1FBQy9CLEdBQUcsbUNBQW1DLENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQztRQUN0RCxNQUFNLEVBQUUsWUFBWTtRQUNwQixPQUFPLEVBQUUsWUFBWTtLQUN0QixDQUFDLENBQUM7SUFDSCxPQUFPLG1CQUFtQixDQUFDLE9BQU8sRUFBRSxPQUFPLEVBQUUsR0FBRyxFQUFFLFNBQVMsRUFBRSxJQUFJLENBQUMsQ0FBQztBQUNyRSxDQUFDLENBQUM7QUFFRixNQUFNLENBQUMsTUFBTSwyQ0FBMkMsR0FBRyxLQUFLLEVBQzlELEtBQXFDLEVBQ3JDLE9BQXVCLEVBQ0MsRUFBRTtJQUMxQixNQUFNLE9BQU8sR0FBZ0I7UUFDM0IsY0FBYyxFQUFFLG1DQUFtQztLQUNwRCxDQUFDO0lBQ0YsSUFBSSxJQUFTLENBQUM7SUFDZCxJQUFJLEdBQUcseUJBQXlCLENBQUM7UUFDL0IsR0FBRywyQ0FBMkMsQ0FBQyxLQUFLLEVBQUUsT0FBTyxDQUFDO1FBQzlELE1BQU0sRUFBRSxvQkFBb0I7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