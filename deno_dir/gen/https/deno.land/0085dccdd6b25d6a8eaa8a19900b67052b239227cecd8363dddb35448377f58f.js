import { HttpRequest as __HttpRequest } from "../../protocol-http/mod.ts";
import { expectNumber as __expectNumber, expectString as __expectString, } from "../../smithy-client/mod.ts";
export const serializeAws_restJson1GetRoleCredentialsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/federation/credentials";
    const query = {
        ...(input.roleName !== undefined && { role_name: input.roleName }),
        ...(input.accountId !== undefined && { account_id: input.accountId }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restJson1ListAccountRolesCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/assignment/roles";
    const query = {
        ...(input.nextToken !== undefined && { next_token: input.nextToken }),
        ...(input.maxResults !== undefined && { max_result: input.maxResults.toString() }),
        ...(input.accountId !== undefined && { account_id: input.accountId }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restJson1ListAccountsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/assignment/accounts";
    const query = {
        ...(input.nextToken !== undefined && { next_token: input.nextToken }),
        ...(input.maxResults !== undefined && { max_result: input.maxResults.toString() }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restJson1LogoutCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/logout";
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        body,
    });
};
export const deserializeAws_restJson1GetRoleCredentialsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1GetRoleCredentialsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        roleCredentials: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.roleCredentials !== undefined && data.roleCredentials !== null) {
        contents.roleCredentials = deserializeAws_restJson1RoleCredentials(data.roleCredentials, context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restJson1GetRoleCredentialsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "ResourceNotFoundException":
        case "com.amazonaws.sso#ResourceNotFoundException":
            response = {
                ...(await deserializeAws_restJson1ResourceNotFoundExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_restJson1ListAccountRolesCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1ListAccountRolesCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        nextToken: undefined,
        roleList: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.nextToken !== undefined && data.nextToken !== null) {
        contents.nextToken = __expectString(data.nextToken);
    }
    if (data.roleList !== undefined && data.roleList !== null) {
        contents.roleList = deserializeAws_restJson1RoleListType(data.roleList, context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restJson1ListAccountRolesCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "ResourceNotFoundException":
        case "com.amazonaws.sso#ResourceNotFoundException":
            response = {
                ...(await deserializeAws_restJson1ResourceNotFoundExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_restJson1ListAccountsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1ListAccountsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        accountList: undefined,
        nextToken: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.accountList !== undefined && data.accountList !== null) {
        contents.accountList = deserializeAws_restJson1AccountListType(data.accountList, context);
    }
    if (data.nextToken !== undefined && data.nextToken !== null) {
        contents.nextToken = __expectString(data.nextToken);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restJson1ListAccountsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "ResourceNotFoundException":
        case "com.amazonaws.sso#ResourceNotFoundException":
            response = {
                ...(await deserializeAws_restJson1ResourceNotFoundExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_restJson1LogoutCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1LogoutCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restJson1LogoutCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
const deserializeAws_restJson1InvalidRequestExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "InvalidRequestException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = __expectString(data.message);
    }
    return contents;
};
const deserializeAws_restJson1ResourceNotFoundExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "ResourceNotFoundException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = __expectString(data.message);
    }
    return contents;
};
const deserializeAws_restJson1TooManyRequestsExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "TooManyRequestsException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = __expectString(data.message);
    }
    return contents;
};
const deserializeAws_restJson1UnauthorizedExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "UnauthorizedException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = __expectString(data.message);
    }
    return contents;
};
const deserializeAws_restJson1AccountInfo = (output, context) => {
    return {
        accountId: __expectString(output.accountId),
        accountName: __expectString(output.accountName),
        emailAddress: __expectString(output.emailAddress),
    };
};
const deserializeAws_restJson1AccountListType = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restJson1AccountInfo(entry, context);
    });
};
const deserializeAws_restJson1RoleCredentials = (output, context) => {
    return {
        accessKeyId: __expectString(output.accessKeyId),
        expiration: __expectNumber(output.expiration),
        secretAccessKey: __expectString(output.secretAccessKey),
        sessionToken: __expectString(output.sessionToken),
    };
};
const deserializeAws_restJson1RoleInfo = (output, context) => {
    return {
        accountId: __expectString(output.accountId),
        roleName: __expectString(output.roleName),
    };
};
const deserializeAws_restJson1RoleListType = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restJson1RoleInfo(entry, context);
    });
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
const isSerializableHeaderValue = (value) => value !== undefined &&
    value !== null &&
    value !== "" &&
    (!Object.getOwnPropertyNames(value).includes("length") || value.length != 0) &&
    (!Object.getOwnPropertyNames(value).includes("size") || value.size != 0);
const parseBody = (streamBody, context) => collectBodyString(streamBody, context).then((encoded) => {
    if (encoded.length) {
        return JSON.parse(encoded);
    }
    return {};
});
const loadRestJsonErrorCode = (output, data) => {
    const findKey = (object, key) => Object.keys(object).find((k) => k.toLowerCase() === key.toLowerCase());
    const sanitizeErrorCode = (rawValue) => {
        let cleanValue = rawValue;
        if (cleanValue.indexOf(":") >= 0) {
            cleanValue = cleanValue.split(":")[0];
        }
        if (cleanValue.indexOf("#") >= 0) {
            cleanValue = cleanValue.split("#")[1];
        }
        return cleanValue;
    };
    const headerKey = findKey(output.headers, "x-amzn-errortype");
    if (headerKey !== undefined) {
        return sanitizeErrorCode(output.headers[headerKey]);
    }
    if (data.code !== undefined) {
        return sanitizeErrorCode(data.code);
    }
    if (data["__type"] !== undefined) {
        return sanitizeErrorCode(data["__type"]);
    }
    return "";
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQXdzX3Jlc3RKc29uMS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIkF3c19yZXN0SnNvbjEudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBYUEsT0FBTyxFQUFFLFdBQVcsSUFBSSxhQUFhLEVBQWtDLE1BQU0sNEJBQTRCLENBQUM7QUFDMUcsT0FBTyxFQUNMLFlBQVksSUFBSSxjQUFjLEVBQzlCLFlBQVksSUFBSSxjQUFjLEdBRS9CLE1BQU0sNEJBQTRCLENBQUM7QUFTcEMsTUFBTSxDQUFDLE1BQU0sK0NBQStDLEdBQUcsS0FBSyxFQUNsRSxLQUFxQyxFQUNyQyxPQUF1QixFQUNDLEVBQUU7SUFDMUIsTUFBTSxFQUFFLFFBQVEsRUFBRSxRQUFRLEdBQUcsT0FBTyxFQUFFLElBQUksRUFBRSxJQUFJLEVBQUUsUUFBUSxFQUFFLEdBQUcsTUFBTSxPQUFPLENBQUMsUUFBUSxFQUFFLENBQUM7SUFDeEYsTUFBTSxPQUFPLEdBQVE7UUFDbkIsR0FBRyxDQUFDLHlCQUF5QixDQUFDLEtBQUssQ0FBQyxXQUFXLENBQUMsSUFBSSxFQUFFLHdCQUF3QixFQUFFLEtBQUssQ0FBQyxXQUFZLEVBQUUsQ0FBQztLQUN0RyxDQUFDO0lBQ0YsSUFBSSxZQUFZLEdBQUcsR0FBRyxRQUFRLEVBQUUsUUFBUSxDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQyxRQUFRLENBQUMsS0FBSyxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxRQUFRLElBQUksRUFBRSxFQUFFLEdBQUcseUJBQXlCLENBQUM7SUFDckgsTUFBTSxLQUFLLEdBQVE7UUFDakIsR0FBRyxDQUFDLEtBQUssQ0FBQyxRQUFRLEtBQUssU0FBUyxJQUFJLEVBQUUsU0FBUyxFQUFFLEtBQUssQ0FBQyxRQUFRLEVBQUUsQ0FBQztRQUNsRSxHQUFHLENBQUMsS0FBSyxDQUFDLFNBQVMsS0FBSyxTQUFTLElBQUksRUFBRSxVQUFVLEVBQUUsS0FBSyxDQUFDLFNBQVMsRUFB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