import { AnalyticsFilter, LifecycleRuleFilter, MetricsFilter, ReplicationRuleFilter, } from "../models/models_0.ts";
import { HttpRequest as __HttpRequest, isValidHostname as __isValidHostname, } from "../../protocol-http/mod.ts";
import { dateToUtcString as __dateToUtcString, expectString as __expectString, extendedEncodeURIComponent as __extendedEncodeURIComponent, getArrayIfSingleItem as __getArrayIfSingleItem, getValueFromTextNode as __getValueFromTextNode, parseBoolean as __parseBoolean, } from "../../smithy-client/mod.ts";
import { XmlNode as __XmlNode, XmlText as __XmlText } from "../../xml-builder/mod.ts";
import { decodeHTML } from "https://jspm.dev/entities";
import { parse as xmlParse } from "https://jspm.dev/fast-xml-parser";
export const serializeAws_restXmlAbortMultipartUploadCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "AbortMultipartUpload",
        ...(input.UploadId !== undefined && { uploadId: input.UploadId }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlCompleteMultipartUploadCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "CompleteMultipartUpload",
        ...(input.UploadId !== undefined && { uploadId: input.UploadId }),
    };
    let body;
    if (input.MultipartUpload !== undefined) {
        body = serializeAws_restXmlCompletedMultipartUpload(input.MultipartUpload, context);
    }
    let contents;
    if (input.MultipartUpload !== undefined) {
        contents = serializeAws_restXmlCompletedMultipartUpload(input.MultipartUpload, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlCopyObjectCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ACL) && { "x-amz-acl": input.ACL }),
        ...(isSerializableHeaderValue(input.CacheControl) && { "cache-control": input.CacheControl }),
        ...(isSerializableHeaderValue(input.ContentDisposition) && { "content-disposition": input.ContentDisposition }),
        ...(isSerializableHeaderValue(input.ContentEncoding) && { "content-encoding": input.ContentEncoding }),
        ...(isSerializableHeaderValue(input.ContentLanguage) && { "content-language": input.ContentLanguage }),
        ...(isSerializableHeaderValue(input.ContentType) && { "content-type": input.ContentType }),
        ...(isSerializableHeaderValue(input.CopySource) && { "x-amz-copy-source": input.CopySource }),
        ...(isSerializableHeaderValue(input.CopySourceIfMatch) && {
            "x-amz-copy-source-if-match": input.CopySourceIfMatch,
        }),
        ...(isSerializableHeaderValue(input.CopySourceIfModifiedSince) && {
            "x-amz-copy-source-if-modified-since": __dateToUtcString(input.CopySourceIfModifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.CopySourceIfNoneMatch) && {
            "x-amz-copy-source-if-none-match": input.CopySourceIfNoneMatch,
        }),
        ...(isSerializableHeaderValue(input.CopySourceIfUnmodifiedSince) && {
            "x-amz-copy-source-if-unmodified-since": __dateToUtcString(input.CopySourceIfUnmodifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.Expires) && { expires: __dateToUtcString(input.Expires).toString() }),
        ...(isSerializableHeaderValue(input.GrantFullControl) && { "x-amz-grant-full-control": input.GrantFullControl }),
        ...(isSerializableHeaderValue(input.GrantRead) && { "x-amz-grant-read": input.GrantRead }),
        ...(isSerializableHeaderValue(input.GrantReadACP) && { "x-amz-grant-read-acp": input.GrantReadACP }),
        ...(isSerializableHeaderValue(input.GrantWriteACP) && { "x-amz-grant-write-acp": input.GrantWriteACP }),
        ...(isSerializableHeaderValue(input.MetadataDirective) && { "x-amz-metadata-directive": input.MetadataDirective }),
        ...(isSerializableHeaderValue(input.TaggingDirective) && { "x-amz-tagging-directive": input.TaggingDirective }),
        ...(isSerializableHeaderValue(input.ServerSideEncryption) && {
            "x-amz-server-side-encryption": input.ServerSideEncryption,
        }),
        ...(isSerializableHeaderValue(input.StorageClass) && { "x-amz-storage-class": input.StorageClass }),
        ...(isSerializableHeaderValue(input.WebsiteRedirectLocation) && {
            "x-amz-website-redirect-location": input.WebsiteRedirectLocation,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSKeyId) && {
            "x-amz-server-side-encryption-aws-kms-key-id": input.SSEKMSKeyId,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSEncryptionContext) && {
            "x-amz-server-side-encryption-context": input.SSEKMSEncryptionContext,
        }),
        ...(isSerializableHeaderValue(input.BucketKeyEnabled) && {
            "x-amz-server-side-encryption-bucket-key-enabled": input.BucketKeyEnabled.toString(),
        }),
        ...(isSerializableHeaderValue(input.CopySourceSSECustomerAlgorithm) && {
            "x-amz-copy-source-server-side-encryption-customer-algorithm": input.CopySourceSSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.CopySourceSSECustomerKey) && {
            "x-amz-copy-source-server-side-encryption-customer-key": input.CopySourceSSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.CopySourceSSECustomerKeyMD5) && {
            "x-amz-copy-source-server-side-encryption-customer-key-md5": input.CopySourceSSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.Tagging) && { "x-amz-tagging": input.Tagging }),
        ...(isSerializableHeaderValue(input.ObjectLockMode) && { "x-amz-object-lock-mode": input.ObjectLockMode }),
        ...(isSerializableHeaderValue(input.ObjectLockRetainUntilDate) && {
            "x-amz-object-lock-retain-until-date": (input.ObjectLockRetainUntilDate.toISOString().split(".")[0] + "Z").toString(),
        }),
        ...(isSerializableHeaderValue(input.ObjectLockLegalHoldStatus) && {
            "x-amz-object-lock-legal-hold": input.ObjectLockLegalHoldStatus,
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
        ...(isSerializableHeaderValue(input.ExpectedSourceBucketOwner) && {
            "x-amz-source-expected-bucket-owner": input.ExpectedSourceBucketOwner,
        }),
        ...(input.Metadata !== undefined &&
            Object.keys(input.Metadata).reduce((acc, suffix) => ({
                ...acc,
                [`x-amz-meta-${suffix.toLowerCase()}`]: input.Metadata[suffix],
            }), {})),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "CopyObject",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlCreateBucketCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ACL) && { "x-amz-acl": input.ACL }),
        ...(isSerializableHeaderValue(input.GrantFullControl) && { "x-amz-grant-full-control": input.GrantFullControl }),
        ...(isSerializableHeaderValue(input.GrantRead) && { "x-amz-grant-read": input.GrantRead }),
        ...(isSerializableHeaderValue(input.GrantReadACP) && { "x-amz-grant-read-acp": input.GrantReadACP }),
        ...(isSerializableHeaderValue(input.GrantWrite) && { "x-amz-grant-write": input.GrantWrite }),
        ...(isSerializableHeaderValue(input.GrantWriteACP) && { "x-amz-grant-write-acp": input.GrantWriteACP }),
        ...(isSerializableHeaderValue(input.ObjectLockEnabledForBucket) && {
            "x-amz-bucket-object-lock-enabled": input.ObjectLockEnabledForBucket.toString(),
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    let body;
    if (input.CreateBucketConfiguration !== undefined) {
        body = serializeAws_restXmlCreateBucketConfiguration(input.CreateBucketConfiguration, context);
    }
    let contents;
    if (input.CreateBucketConfiguration !== undefined) {
        contents = serializeAws_restXmlCreateBucketConfiguration(input.CreateBucketConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        body,
    });
};
export const serializeAws_restXmlCreateMultipartUploadCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ACL) && { "x-amz-acl": input.ACL }),
        ...(isSerializableHeaderValue(input.CacheControl) && { "cache-control": input.CacheControl }),
        ...(isSerializableHeaderValue(input.ContentDisposition) && { "content-disposition": input.ContentDisposition }),
        ...(isSerializableHeaderValue(input.ContentEncoding) && { "content-encoding": input.ContentEncoding }),
        ...(isSerializableHeaderValue(input.ContentLanguage) && { "content-language": input.ContentLanguage }),
        ...(isSerializableHeaderValue(input.ContentType) && { "content-type": input.ContentType }),
        ...(isSerializableHeaderValue(input.Expires) && { expires: __dateToUtcString(input.Expires).toString() }),
        ...(isSerializableHeaderValue(input.GrantFullControl) && { "x-amz-grant-full-control": input.GrantFullControl }),
        ...(isSerializableHeaderValue(input.GrantRead) && { "x-amz-grant-read": input.GrantRead }),
        ...(isSerializableHeaderValue(input.GrantReadACP) && { "x-amz-grant-read-acp": input.GrantReadACP }),
        ...(isSerializableHeaderValue(input.GrantWriteACP) && { "x-amz-grant-write-acp": input.GrantWriteACP }),
        ...(isSerializableHeaderValue(input.ServerSideEncryption) && {
            "x-amz-server-side-encryption": input.ServerSideEncryption,
        }),
        ...(isSerializableHeaderValue(input.StorageClass) && { "x-amz-storage-class": input.StorageClass }),
        ...(isSerializableHeaderValue(input.WebsiteRedirectLocation) && {
            "x-amz-website-redirect-location": input.WebsiteRedirectLocation,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSKeyId) && {
            "x-amz-server-side-encryption-aws-kms-key-id": input.SSEKMSKeyId,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSEncryptionContext) && {
            "x-amz-server-side-encryption-context": input.SSEKMSEncryptionContext,
        }),
        ...(isSerializableHeaderValue(input.BucketKeyEnabled) && {
            "x-amz-server-side-encryption-bucket-key-enabled": input.BucketKeyEnabled.toString(),
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.Tagging) && { "x-amz-tagging": input.Tagging }),
        ...(isSerializableHeaderValue(input.ObjectLockMode) && { "x-amz-object-lock-mode": input.ObjectLockMode }),
        ...(isSerializableHeaderValue(input.ObjectLockRetainUntilDate) && {
            "x-amz-object-lock-retain-until-date": (input.ObjectLockRetainUntilDate.toISOString().split(".")[0] + "Z").toString(),
        }),
        ...(isSerializableHeaderValue(input.ObjectLockLegalHoldStatus) && {
            "x-amz-object-lock-legal-hold": input.ObjectLockLegalHoldStatus,
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
        ...(input.Metadata !== undefined &&
            Object.keys(input.Metadata).reduce((acc, suffix) => ({
                ...acc,
                [`x-amz-meta-${suffix.toLowerCase()}`]: input.Metadata[suffix],
            }), {})),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        uploads: "",
        "x-id": "CreateMultipartUpload",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketAnalyticsConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        analytics: "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketCorsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        cors: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketEncryptionCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        encryption: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketIntelligentTieringConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {};
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "intelligent-tiering": "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketInventoryConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        inventory: "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketLifecycleCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        lifecycle: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketMetricsConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        metrics: "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketOwnershipControlsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        ownershipControls: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketPolicyCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        policy: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketReplicationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        replication: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketTaggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        tagging: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteBucketWebsiteCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        website: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteObjectCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.MFA) && { "x-amz-mfa": input.MFA }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.BypassGovernanceRetention) && {
            "x-amz-bypass-governance-retention": input.BypassGovernanceRetention.toString(),
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "DeleteObject",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteObjectsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.MFA) && { "x-amz-mfa": input.MFA }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.BypassGovernanceRetention) && {
            "x-amz-bypass-governance-retention": input.BypassGovernanceRetention.toString(),
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        delete: "",
        "x-id": "DeleteObjects",
    };
    let body;
    if (input.Delete !== undefined) {
        body = serializeAws_restXmlDelete(input.Delete, context);
    }
    let contents;
    if (input.Delete !== undefined) {
        contents = serializeAws_restXmlDelete(input.Delete, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeleteObjectTaggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        tagging: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlDeletePublicAccessBlockCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        publicAccessBlock: "",
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "DELETE",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlGetBucketAccelerateConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        accelerate: "",
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
export const serializeAws_restXmlGetBucketAclCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        acl: "",
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
export const serializeAws_restXmlGetBucketAnalyticsConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        analytics: "",
        "x-id": "GetBucketAnalyticsConfiguration",
        ...(input.Id !== undefined && { id: input.Id }),
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
export const serializeAws_restXmlGetBucketCorsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        cors: "",
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
export const serializeAws_restXmlGetBucketEncryptionCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        encryption: "",
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
export const serializeAws_restXmlGetBucketIntelligentTieringConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {};
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "intelligent-tiering": "",
        "x-id": "GetBucketIntelligentTieringConfiguration",
        ...(input.Id !== undefined && { id: input.Id }),
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
export const serializeAws_restXmlGetBucketInventoryConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        inventory: "",
        "x-id": "GetBucketInventoryConfiguration",
        ...(input.Id !== undefined && { id: input.Id }),
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
export const serializeAws_restXmlGetBucketLifecycleConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        lifecycle: "",
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
export const serializeAws_restXmlGetBucketLocationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        location: "",
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
export const serializeAws_restXmlGetBucketLoggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        logging: "",
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
export const serializeAws_restXmlGetBucketMetricsConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        metrics: "",
        "x-id": "GetBucketMetricsConfiguration",
        ...(input.Id !== undefined && { id: input.Id }),
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
export const serializeAws_restXmlGetBucketNotificationConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        notification: "",
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
export const serializeAws_restXmlGetBucketOwnershipControlsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        ownershipControls: "",
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
export const serializeAws_restXmlGetBucketPolicyCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        policy: "",
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
export const serializeAws_restXmlGetBucketPolicyStatusCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        policyStatus: "",
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
export const serializeAws_restXmlGetBucketReplicationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        replication: "",
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
export const serializeAws_restXmlGetBucketRequestPaymentCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        requestPayment: "",
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
export const serializeAws_restXmlGetBucketTaggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        tagging: "",
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
export const serializeAws_restXmlGetBucketVersioningCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        versioning: "",
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
export const serializeAws_restXmlGetBucketWebsiteCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        website: "",
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
export const serializeAws_restXmlGetObjectCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.IfMatch) && { "if-match": input.IfMatch }),
        ...(isSerializableHeaderValue(input.IfModifiedSince) && {
            "if-modified-since": __dateToUtcString(input.IfModifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.IfNoneMatch) && { "if-none-match": input.IfNoneMatch }),
        ...(isSerializableHeaderValue(input.IfUnmodifiedSince) && {
            "if-unmodified-since": __dateToUtcString(input.IfUnmodifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.Range) && { range: input.Range }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "GetObject",
        ...(input.ResponseCacheControl !== undefined && { "response-cache-control": input.ResponseCacheControl }),
        ...(input.ResponseContentDisposition !== undefined && {
            "response-content-disposition": input.ResponseContentDisposition,
        }),
        ...(input.ResponseContentEncoding !== undefined && { "response-content-encoding": input.ResponseContentEncoding }),
        ...(input.ResponseContentLanguage !== undefined && { "response-content-language": input.ResponseContentLanguage }),
        ...(input.ResponseContentType !== undefined && { "response-content-type": input.ResponseContentType }),
        ...(input.ResponseExpires !== undefined && {
            "response-expires": __dateToUtcString(input.ResponseExpires).toString(),
        }),
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
        ...(input.PartNumber !== undefined && { partNumber: input.PartNumber.toString() }),
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
export const serializeAws_restXmlGetObjectAclCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        acl: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
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
export const serializeAws_restXmlGetObjectLegalHoldCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "legal-hold": "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
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
export const serializeAws_restXmlGetObjectLockConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "object-lock": "",
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
export const serializeAws_restXmlGetObjectRetentionCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        retention: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
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
export const serializeAws_restXmlGetObjectTaggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        tagging: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
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
export const serializeAws_restXmlGetObjectTorrentCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        torrent: "",
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
export const serializeAws_restXmlGetPublicAccessBlockCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        publicAccessBlock: "",
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
export const serializeAws_restXmlHeadBucketCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "HEAD",
        headers,
        path: resolvedPath,
        body,
    });
};
export const serializeAws_restXmlHeadObjectCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.IfMatch) && { "if-match": input.IfMatch }),
        ...(isSerializableHeaderValue(input.IfModifiedSince) && {
            "if-modified-since": __dateToUtcString(input.IfModifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.IfNoneMatch) && { "if-none-match": input.IfNoneMatch }),
        ...(isSerializableHeaderValue(input.IfUnmodifiedSince) && {
            "if-unmodified-since": __dateToUtcString(input.IfUnmodifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.Range) && { range: input.Range }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
        ...(input.PartNumber !== undefined && { partNumber: input.PartNumber.toString() }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "HEAD",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlListBucketAnalyticsConfigurationsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        analytics: "",
        "x-id": "ListBucketAnalyticsConfigurations",
        ...(input.ContinuationToken !== undefined && { "continuation-token": input.ContinuationToken }),
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
export const serializeAws_restXmlListBucketIntelligentTieringConfigurationsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {};
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "intelligent-tiering": "",
        "x-id": "ListBucketIntelligentTieringConfigurations",
        ...(input.ContinuationToken !== undefined && { "continuation-token": input.ContinuationToken }),
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
export const serializeAws_restXmlListBucketInventoryConfigurationsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        inventory: "",
        "x-id": "ListBucketInventoryConfigurations",
        ...(input.ContinuationToken !== undefined && { "continuation-token": input.ContinuationToken }),
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
export const serializeAws_restXmlListBucketMetricsConfigurationsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        metrics: "",
        "x-id": "ListBucketMetricsConfigurations",
        ...(input.ContinuationToken !== undefined && { "continuation-token": input.ContinuationToken }),
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
export const serializeAws_restXmlListBucketsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/";
    let body;
    body = "";
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        body,
    });
};
export const serializeAws_restXmlListMultipartUploadsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        uploads: "",
        ...(input.Delimiter !== undefined && { delimiter: input.Delimiter }),
        ...(input.EncodingType !== undefined && { "encoding-type": input.EncodingType }),
        ...(input.KeyMarker !== undefined && { "key-marker": input.KeyMarker }),
        ...(input.MaxUploads !== undefined && { "max-uploads": input.MaxUploads.toString() }),
        ...(input.Prefix !== undefined && { prefix: input.Prefix }),
        ...(input.UploadIdMarker !== undefined && { "upload-id-marker": input.UploadIdMarker }),
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
export const serializeAws_restXmlListObjectsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        ...(input.Delimiter !== undefined && { delimiter: input.Delimiter }),
        ...(input.EncodingType !== undefined && { "encoding-type": input.EncodingType }),
        ...(input.Marker !== undefined && { marker: input.Marker }),
        ...(input.MaxKeys !== undefined && { "max-keys": input.MaxKeys.toString() }),
        ...(input.Prefix !== undefined && { prefix: input.Prefix }),
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
export const serializeAws_restXmlListObjectsV2Command = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "list-type": "2",
        ...(input.Delimiter !== undefined && { delimiter: input.Delimiter }),
        ...(input.EncodingType !== undefined && { "encoding-type": input.EncodingType }),
        ...(input.MaxKeys !== undefined && { "max-keys": input.MaxKeys.toString() }),
        ...(input.Prefix !== undefined && { prefix: input.Prefix }),
        ...(input.ContinuationToken !== undefined && { "continuation-token": input.ContinuationToken }),
        ...(input.FetchOwner !== undefined && { "fetch-owner": input.FetchOwner.toString() }),
        ...(input.StartAfter !== undefined && { "start-after": input.StartAfter }),
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
export const serializeAws_restXmlListObjectVersionsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        versions: "",
        ...(input.Delimiter !== undefined && { delimiter: input.Delimiter }),
        ...(input.EncodingType !== undefined && { "encoding-type": input.EncodingType }),
        ...(input.KeyMarker !== undefined && { "key-marker": input.KeyMarker }),
        ...(input.MaxKeys !== undefined && { "max-keys": input.MaxKeys.toString() }),
        ...(input.Prefix !== undefined && { prefix: input.Prefix }),
        ...(input.VersionIdMarker !== undefined && { "version-id-marker": input.VersionIdMarker }),
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
export const serializeAws_restXmlListPartsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "ListParts",
        ...(input.MaxParts !== undefined && { "max-parts": input.MaxParts.toString() }),
        ...(input.PartNumberMarker !== undefined && { "part-number-marker": input.PartNumberMarker }),
        ...(input.UploadId !== undefined && { uploadId: input.UploadId }),
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
export const serializeAws_restXmlPutBucketAccelerateConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        accelerate: "",
    };
    let body;
    if (input.AccelerateConfiguration !== undefined) {
        body = serializeAws_restXmlAccelerateConfiguration(input.AccelerateConfiguration, context);
    }
    let contents;
    if (input.AccelerateConfiguration !== undefined) {
        contents = serializeAws_restXmlAccelerateConfiguration(input.AccelerateConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketAclCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ACL) && { "x-amz-acl": input.ACL }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.GrantFullControl) && { "x-amz-grant-full-control": input.GrantFullControl }),
        ...(isSerializableHeaderValue(input.GrantRead) && { "x-amz-grant-read": input.GrantRead }),
        ...(isSerializableHeaderValue(input.GrantReadACP) && { "x-amz-grant-read-acp": input.GrantReadACP }),
        ...(isSerializableHeaderValue(input.GrantWrite) && { "x-amz-grant-write": input.GrantWrite }),
        ...(isSerializableHeaderValue(input.GrantWriteACP) && { "x-amz-grant-write-acp": input.GrantWriteACP }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        acl: "",
    };
    let body;
    if (input.AccessControlPolicy !== undefined) {
        body = serializeAws_restXmlAccessControlPolicy(input.AccessControlPolicy, context);
    }
    let contents;
    if (input.AccessControlPolicy !== undefined) {
        contents = serializeAws_restXmlAccessControlPolicy(input.AccessControlPolicy, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketAnalyticsConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        analytics: "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    if (input.AnalyticsConfiguration !== undefined) {
        body = serializeAws_restXmlAnalyticsConfiguration(input.AnalyticsConfiguration, context);
    }
    let contents;
    if (input.AnalyticsConfiguration !== undefined) {
        contents = serializeAws_restXmlAnalyticsConfiguration(input.AnalyticsConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketCorsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        cors: "",
    };
    let body;
    if (input.CORSConfiguration !== undefined) {
        body = serializeAws_restXmlCORSConfiguration(input.CORSConfiguration, context);
    }
    let contents;
    if (input.CORSConfiguration !== undefined) {
        contents = serializeAws_restXmlCORSConfiguration(input.CORSConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketEncryptionCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        encryption: "",
    };
    let body;
    if (input.ServerSideEncryptionConfiguration !== undefined) {
        body = serializeAws_restXmlServerSideEncryptionConfiguration(input.ServerSideEncryptionConfiguration, context);
    }
    let contents;
    if (input.ServerSideEncryptionConfiguration !== undefined) {
        contents = serializeAws_restXmlServerSideEncryptionConfiguration(input.ServerSideEncryptionConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketIntelligentTieringConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "intelligent-tiering": "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    if (input.IntelligentTieringConfiguration !== undefined) {
        body = serializeAws_restXmlIntelligentTieringConfiguration(input.IntelligentTieringConfiguration, context);
    }
    let contents;
    if (input.IntelligentTieringConfiguration !== undefined) {
        contents = serializeAws_restXmlIntelligentTieringConfiguration(input.IntelligentTieringConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketInventoryConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        inventory: "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    if (input.InventoryConfiguration !== undefined) {
        body = serializeAws_restXmlInventoryConfiguration(input.InventoryConfiguration, context);
    }
    let contents;
    if (input.InventoryConfiguration !== undefined) {
        contents = serializeAws_restXmlInventoryConfiguration(input.InventoryConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketLifecycleConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        lifecycle: "",
    };
    let body;
    if (input.LifecycleConfiguration !== undefined) {
        body = serializeAws_restXmlBucketLifecycleConfiguration(input.LifecycleConfiguration, context);
    }
    let contents;
    if (input.LifecycleConfiguration !== undefined) {
        contents = serializeAws_restXmlBucketLifecycleConfiguration(input.LifecycleConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketLoggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        logging: "",
    };
    let body;
    if (input.BucketLoggingStatus !== undefined) {
        body = serializeAws_restXmlBucketLoggingStatus(input.BucketLoggingStatus, context);
    }
    let contents;
    if (input.BucketLoggingStatus !== undefined) {
        contents = serializeAws_restXmlBucketLoggingStatus(input.BucketLoggingStatus, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketMetricsConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        metrics: "",
        ...(input.Id !== undefined && { id: input.Id }),
    };
    let body;
    if (input.MetricsConfiguration !== undefined) {
        body = serializeAws_restXmlMetricsConfiguration(input.MetricsConfiguration, context);
    }
    let contents;
    if (input.MetricsConfiguration !== undefined) {
        contents = serializeAws_restXmlMetricsConfiguration(input.MetricsConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketNotificationConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        notification: "",
    };
    let body;
    if (input.NotificationConfiguration !== undefined) {
        body = serializeAws_restXmlNotificationConfiguration(input.NotificationConfiguration, context);
    }
    let contents;
    if (input.NotificationConfiguration !== undefined) {
        contents = serializeAws_restXmlNotificationConfiguration(input.NotificationConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketOwnershipControlsCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        ownershipControls: "",
    };
    let body;
    if (input.OwnershipControls !== undefined) {
        body = serializeAws_restXmlOwnershipControls(input.OwnershipControls, context);
    }
    let contents;
    if (input.OwnershipControls !== undefined) {
        contents = serializeAws_restXmlOwnershipControls(input.OwnershipControls, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketPolicyCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "text/plain",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ConfirmRemoveSelfBucketAccess) && {
            "x-amz-confirm-remove-self-bucket-access": input.ConfirmRemoveSelfBucketAccess.toString(),
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        policy: "",
    };
    let body;
    if (input.Policy !== undefined) {
        body = input.Policy;
    }
    let contents;
    if (input.Policy !== undefined) {
        contents = input.Policy;
        body = contents;
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketReplicationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.Token) && { "x-amz-bucket-object-lock-token": input.Token }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        replication: "",
    };
    let body;
    if (input.ReplicationConfiguration !== undefined) {
        body = serializeAws_restXmlReplicationConfiguration(input.ReplicationConfiguration, context);
    }
    let contents;
    if (input.ReplicationConfiguration !== undefined) {
        contents = serializeAws_restXmlReplicationConfiguration(input.ReplicationConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketRequestPaymentCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        requestPayment: "",
    };
    let body;
    if (input.RequestPaymentConfiguration !== undefined) {
        body = serializeAws_restXmlRequestPaymentConfiguration(input.RequestPaymentConfiguration, context);
    }
    let contents;
    if (input.RequestPaymentConfiguration !== undefined) {
        contents = serializeAws_restXmlRequestPaymentConfiguration(input.RequestPaymentConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketTaggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        tagging: "",
    };
    let body;
    if (input.Tagging !== undefined) {
        body = serializeAws_restXmlTagging(input.Tagging, context);
    }
    let contents;
    if (input.Tagging !== undefined) {
        contents = serializeAws_restXmlTagging(input.Tagging, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketVersioningCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.MFA) && { "x-amz-mfa": input.MFA }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        versioning: "",
    };
    let body;
    if (input.VersioningConfiguration !== undefined) {
        body = serializeAws_restXmlVersioningConfiguration(input.VersioningConfiguration, context);
    }
    let contents;
    if (input.VersioningConfiguration !== undefined) {
        contents = serializeAws_restXmlVersioningConfiguration(input.VersioningConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutBucketWebsiteCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        website: "",
    };
    let body;
    if (input.WebsiteConfiguration !== undefined) {
        body = serializeAws_restXmlWebsiteConfiguration(input.WebsiteConfiguration, context);
    }
    let contents;
    if (input.WebsiteConfiguration !== undefined) {
        contents = serializeAws_restXmlWebsiteConfiguration(input.WebsiteConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutObjectCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/octet-stream",
        ...(isSerializableHeaderValue(input.ACL) && { "x-amz-acl": input.ACL }),
        ...(isSerializableHeaderValue(input.CacheControl) && { "cache-control": input.CacheControl }),
        ...(isSerializableHeaderValue(input.ContentDisposition) && { "content-disposition": input.ContentDisposition }),
        ...(isSerializableHeaderValue(input.ContentEncoding) && { "content-encoding": input.ContentEncoding }),
        ...(isSerializableHeaderValue(input.ContentLanguage) && { "content-language": input.ContentLanguage }),
        ...(isSerializableHeaderValue(input.ContentLength) && { "content-length": input.ContentLength.toString() }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ContentType) && { "content-type": input.ContentType }),
        ...(isSerializableHeaderValue(input.Expires) && { expires: __dateToUtcString(input.Expires).toString() }),
        ...(isSerializableHeaderValue(input.GrantFullControl) && { "x-amz-grant-full-control": input.GrantFullControl }),
        ...(isSerializableHeaderValue(input.GrantRead) && { "x-amz-grant-read": input.GrantRead }),
        ...(isSerializableHeaderValue(input.GrantReadACP) && { "x-amz-grant-read-acp": input.GrantReadACP }),
        ...(isSerializableHeaderValue(input.GrantWriteACP) && { "x-amz-grant-write-acp": input.GrantWriteACP }),
        ...(isSerializableHeaderValue(input.ServerSideEncryption) && {
            "x-amz-server-side-encryption": input.ServerSideEncryption,
        }),
        ...(isSerializableHeaderValue(input.StorageClass) && { "x-amz-storage-class": input.StorageClass }),
        ...(isSerializableHeaderValue(input.WebsiteRedirectLocation) && {
            "x-amz-website-redirect-location": input.WebsiteRedirectLocation,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSKeyId) && {
            "x-amz-server-side-encryption-aws-kms-key-id": input.SSEKMSKeyId,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSEncryptionContext) && {
            "x-amz-server-side-encryption-context": input.SSEKMSEncryptionContext,
        }),
        ...(isSerializableHeaderValue(input.BucketKeyEnabled) && {
            "x-amz-server-side-encryption-bucket-key-enabled": input.BucketKeyEnabled.toString(),
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.Tagging) && { "x-amz-tagging": input.Tagging }),
        ...(isSerializableHeaderValue(input.ObjectLockMode) && { "x-amz-object-lock-mode": input.ObjectLockMode }),
        ...(isSerializableHeaderValue(input.ObjectLockRetainUntilDate) && {
            "x-amz-object-lock-retain-until-date": (input.ObjectLockRetainUntilDate.toISOString().split(".")[0] + "Z").toString(),
        }),
        ...(isSerializableHeaderValue(input.ObjectLockLegalHoldStatus) && {
            "x-amz-object-lock-legal-hold": input.ObjectLockLegalHoldStatus,
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
        ...(input.Metadata !== undefined &&
            Object.keys(input.Metadata).reduce((acc, suffix) => ({
                ...acc,
                [`x-amz-meta-${suffix.toLowerCase()}`]: input.Metadata[suffix],
            }), {})),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "PutObject",
    };
    let body;
    if (input.Body !== undefined) {
        body = input.Body;
    }
    let contents;
    if (input.Body !== undefined) {
        contents = input.Body;
        body = contents;
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutObjectAclCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ACL) && { "x-amz-acl": input.ACL }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.GrantFullControl) && { "x-amz-grant-full-control": input.GrantFullControl }),
        ...(isSerializableHeaderValue(input.GrantRead) && { "x-amz-grant-read": input.GrantRead }),
        ...(isSerializableHeaderValue(input.GrantReadACP) && { "x-amz-grant-read-acp": input.GrantReadACP }),
        ...(isSerializableHeaderValue(input.GrantWrite) && { "x-amz-grant-write": input.GrantWrite }),
        ...(isSerializableHeaderValue(input.GrantWriteACP) && { "x-amz-grant-write-acp": input.GrantWriteACP }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        acl: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    if (input.AccessControlPolicy !== undefined) {
        body = serializeAws_restXmlAccessControlPolicy(input.AccessControlPolicy, context);
    }
    let contents;
    if (input.AccessControlPolicy !== undefined) {
        contents = serializeAws_restXmlAccessControlPolicy(input.AccessControlPolicy, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutObjectLegalHoldCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "legal-hold": "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    if (input.LegalHold !== undefined) {
        body = serializeAws_restXmlObjectLockLegalHold(input.LegalHold, context);
    }
    let contents;
    if (input.LegalHold !== undefined) {
        contents = serializeAws_restXmlObjectLockLegalHold(input.LegalHold, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutObjectLockConfigurationCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.Token) && { "x-amz-bucket-object-lock-token": input.Token }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        "object-lock": "",
    };
    let body;
    if (input.ObjectLockConfiguration !== undefined) {
        body = serializeAws_restXmlObjectLockConfiguration(input.ObjectLockConfiguration, context);
    }
    let contents;
    if (input.ObjectLockConfiguration !== undefined) {
        contents = serializeAws_restXmlObjectLockConfiguration(input.ObjectLockConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutObjectRetentionCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.BypassGovernanceRetention) && {
            "x-amz-bypass-governance-retention": input.BypassGovernanceRetention.toString(),
        }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        retention: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    if (input.Retention !== undefined) {
        body = serializeAws_restXmlObjectLockRetention(input.Retention, context);
    }
    let contents;
    if (input.Retention !== undefined) {
        contents = serializeAws_restXmlObjectLockRetention(input.Retention, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutObjectTaggingCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        tagging: "",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    if (input.Tagging !== undefined) {
        body = serializeAws_restXmlTagging(input.Tagging, context);
    }
    let contents;
    if (input.Tagging !== undefined) {
        contents = serializeAws_restXmlTagging(input.Tagging, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlPutPublicAccessBlockCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    const query = {
        publicAccessBlock: "",
    };
    let body;
    if (input.PublicAccessBlockConfiguration !== undefined) {
        body = serializeAws_restXmlPublicAccessBlockConfiguration(input.PublicAccessBlockConfiguration, context);
    }
    let contents;
    if (input.PublicAccessBlockConfiguration !== undefined) {
        contents = serializeAws_restXmlPublicAccessBlockConfiguration(input.PublicAccessBlockConfiguration, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlRestoreObjectCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        restore: "",
        "x-id": "RestoreObject",
        ...(input.VersionId !== undefined && { versionId: input.VersionId }),
    };
    let body;
    if (input.RestoreRequest !== undefined) {
        body = serializeAws_restXmlRestoreRequest(input.RestoreRequest, context);
    }
    let contents;
    if (input.RestoreRequest !== undefined) {
        contents = serializeAws_restXmlRestoreRequest(input.RestoreRequest, context);
        body = '<?xml version="1.0" encoding="UTF-8"?>';
        contents.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
        body += contents.toString();
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlSelectObjectContentCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/xml",
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        select: "",
        "select-type": "2",
        "x-id": "SelectObjectContent",
    };
    let body;
    body = '<?xml version="1.0" encoding="UTF-8"?>';
    const bodyNode = new __XmlNode("SelectObjectContentRequest");
    bodyNode.addAttribute("xmlns", "http://s3.amazonaws.com/doc/2006-03-01/");
    if (input.Expression !== undefined) {
        const node = new __XmlNode("Expression").addChildNode(new __XmlText(input.Expression)).withName("Expression");
        bodyNode.addChildNode(node);
    }
    if (input.ExpressionType !== undefined) {
        const node = new __XmlNode("ExpressionType")
            .addChildNode(new __XmlText(input.ExpressionType))
            .withName("ExpressionType");
        bodyNode.addChildNode(node);
    }
    if (input.InputSerialization !== undefined) {
        const node = serializeAws_restXmlInputSerialization(input.InputSerialization, context).withName("InputSerialization");
        bodyNode.addChildNode(node);
    }
    if (input.OutputSerialization !== undefined) {
        const node = serializeAws_restXmlOutputSerialization(input.OutputSerialization, context).withName("OutputSerialization");
        bodyNode.addChildNode(node);
    }
    if (input.RequestProgress !== undefined) {
        const node = serializeAws_restXmlRequestProgress(input.RequestProgress, context).withName("RequestProgress");
        bodyNode.addChildNode(node);
    }
    if (input.ScanRange !== undefined) {
        const node = serializeAws_restXmlScanRange(input.ScanRange, context).withName("ScanRange");
        bodyNode.addChildNode(node);
    }
    body += bodyNode.toString();
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlUploadPartCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/octet-stream",
        ...(isSerializableHeaderValue(input.ContentLength) && { "content-length": input.ContentLength.toString() }),
        ...(isSerializableHeaderValue(input.ContentMD5) && { "content-md5": input.ContentMD5 }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "UploadPart",
        ...(input.PartNumber !== undefined && { partNumber: input.PartNumber.toString() }),
        ...(input.UploadId !== undefined && { uploadId: input.UploadId }),
    };
    let body;
    if (input.Body !== undefined) {
        body = input.Body;
    }
    let contents;
    if (input.Body !== undefined) {
        contents = input.Body;
        body = contents;
    }
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlUploadPartCopyCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        ...(isSerializableHeaderValue(input.CopySource) && { "x-amz-copy-source": input.CopySource }),
        ...(isSerializableHeaderValue(input.CopySourceIfMatch) && {
            "x-amz-copy-source-if-match": input.CopySourceIfMatch,
        }),
        ...(isSerializableHeaderValue(input.CopySourceIfModifiedSince) && {
            "x-amz-copy-source-if-modified-since": __dateToUtcString(input.CopySourceIfModifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.CopySourceIfNoneMatch) && {
            "x-amz-copy-source-if-none-match": input.CopySourceIfNoneMatch,
        }),
        ...(isSerializableHeaderValue(input.CopySourceIfUnmodifiedSince) && {
            "x-amz-copy-source-if-unmodified-since": __dateToUtcString(input.CopySourceIfUnmodifiedSince).toString(),
        }),
        ...(isSerializableHeaderValue(input.CopySourceRange) && { "x-amz-copy-source-range": input.CopySourceRange }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKey) && {
            "x-amz-server-side-encryption-customer-key": input.SSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.CopySourceSSECustomerAlgorithm) && {
            "x-amz-copy-source-server-side-encryption-customer-algorithm": input.CopySourceSSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.CopySourceSSECustomerKey) && {
            "x-amz-copy-source-server-side-encryption-customer-key": input.CopySourceSSECustomerKey,
        }),
        ...(isSerializableHeaderValue(input.CopySourceSSECustomerKeyMD5) && {
            "x-amz-copy-source-server-side-encryption-customer-key-md5": input.CopySourceSSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.RequestPayer) && { "x-amz-request-payer": input.RequestPayer }),
        ...(isSerializableHeaderValue(input.ExpectedBucketOwner) && {
            "x-amz-expected-bucket-owner": input.ExpectedBucketOwner,
        }),
        ...(isSerializableHeaderValue(input.ExpectedSourceBucketOwner) && {
            "x-amz-source-expected-bucket-owner": input.ExpectedSourceBucketOwner,
        }),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/{Bucket}/{Key+}";
    if (input.Bucket !== undefined) {
        const labelValue = input.Bucket;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Bucket.");
        }
        resolvedPath = resolvedPath.replace("{Bucket}", __extendedEncodeURIComponent(labelValue));
    }
    else {
        throw new Error("No value provided for input HTTP label: Bucket.");
    }
    if (input.Key !== undefined) {
        const labelValue = input.Key;
        if (labelValue.length <= 0) {
            throw new Error("Empty value provided for input HTTP label: Key.");
        }
        resolvedPath = resolvedPath.replace("{Key+}", labelValue
            .split("/")
            .map((segment) => __extendedEncodeURIComponent(segment))
            .join("/"));
    }
    else {
        throw new Error("No value provided for input HTTP label: Key.");
    }
    const query = {
        "x-id": "UploadPartCopy",
        ...(input.PartNumber !== undefined && { partNumber: input.PartNumber.toString() }),
        ...(input.UploadId !== undefined && { uploadId: input.UploadId }),
    };
    let body;
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "PUT",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restXmlWriteGetObjectResponseCommand = async (input, context) => {
    const { hostname, protocol = "https", port, path: basePath } = await context.endpoint();
    const headers = {
        "content-type": "application/octet-stream",
        "x-amz-content-sha256": "UNSIGNED-PAYLOAD",
        ...(isSerializableHeaderValue(input.RequestRoute) && { "x-amz-request-route": input.RequestRoute }),
        ...(isSerializableHeaderValue(input.RequestToken) && { "x-amz-request-token": input.RequestToken }),
        ...(isSerializableHeaderValue(input.StatusCode) && { "x-amz-fwd-status": input.StatusCode.toString() }),
        ...(isSerializableHeaderValue(input.ErrorCode) && { "x-amz-fwd-error-code": input.ErrorCode }),
        ...(isSerializableHeaderValue(input.ErrorMessage) && { "x-amz-fwd-error-message": input.ErrorMessage }),
        ...(isSerializableHeaderValue(input.AcceptRanges) && { "x-amz-fwd-header-accept-ranges": input.AcceptRanges }),
        ...(isSerializableHeaderValue(input.CacheControl) && { "x-amz-fwd-header-cache-control": input.CacheControl }),
        ...(isSerializableHeaderValue(input.ContentDisposition) && {
            "x-amz-fwd-header-content-disposition": input.ContentDisposition,
        }),
        ...(isSerializableHeaderValue(input.ContentEncoding) && {
            "x-amz-fwd-header-content-encoding": input.ContentEncoding,
        }),
        ...(isSerializableHeaderValue(input.ContentLanguage) && {
            "x-amz-fwd-header-content-language": input.ContentLanguage,
        }),
        ...(isSerializableHeaderValue(input.ContentLength) && { "content-length": input.ContentLength.toString() }),
        ...(isSerializableHeaderValue(input.ContentRange) && { "x-amz-fwd-header-content-range": input.ContentRange }),
        ...(isSerializableHeaderValue(input.ContentType) && { "x-amz-fwd-header-content-type": input.ContentType }),
        ...(isSerializableHeaderValue(input.DeleteMarker) && {
            "x-amz-fwd-header-x-amz-delete-marker": input.DeleteMarker.toString(),
        }),
        ...(isSerializableHeaderValue(input.ETag) && { "x-amz-fwd-header-etag": input.ETag }),
        ...(isSerializableHeaderValue(input.Expires) && {
            "x-amz-fwd-header-expires": __dateToUtcString(input.Expires).toString(),
        }),
        ...(isSerializableHeaderValue(input.Expiration) && { "x-amz-fwd-header-x-amz-expiration": input.Expiration }),
        ...(isSerializableHeaderValue(input.LastModified) && {
            "x-amz-fwd-header-last-modified": __dateToUtcString(input.LastModified).toString(),
        }),
        ...(isSerializableHeaderValue(input.MissingMeta) && {
            "x-amz-fwd-header-x-amz-missing-meta": input.MissingMeta.toString(),
        }),
        ...(isSerializableHeaderValue(input.ObjectLockMode) && {
            "x-amz-fwd-header-x-amz-object-lock-mode": input.ObjectLockMode,
        }),
        ...(isSerializableHeaderValue(input.ObjectLockLegalHoldStatus) && {
            "x-amz-fwd-header-x-amz-object-lock-legal-hold": input.ObjectLockLegalHoldStatus,
        }),
        ...(isSerializableHeaderValue(input.ObjectLockRetainUntilDate) && {
            "x-amz-fwd-header-x-amz-object-lock-retain-until-date": (input.ObjectLockRetainUntilDate.toISOString().split(".")[0] + "Z").toString(),
        }),
        ...(isSerializableHeaderValue(input.PartsCount) && {
            "x-amz-fwd-header-x-amz-mp-parts-count": input.PartsCount.toString(),
        }),
        ...(isSerializableHeaderValue(input.ReplicationStatus) && {
            "x-amz-fwd-header-x-amz-replication-status": input.ReplicationStatus,
        }),
        ...(isSerializableHeaderValue(input.RequestCharged) && {
            "x-amz-fwd-header-x-amz-request-charged": input.RequestCharged,
        }),
        ...(isSerializableHeaderValue(input.Restore) && { "x-amz-fwd-header-x-amz-restore": input.Restore }),
        ...(isSerializableHeaderValue(input.ServerSideEncryption) && {
            "x-amz-fwd-header-x-amz-server-side-encryption": input.ServerSideEncryption,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerAlgorithm) && {
            "x-amz-fwd-header-x-amz-server-side-encryption-customer-algorithm": input.SSECustomerAlgorithm,
        }),
        ...(isSerializableHeaderValue(input.SSEKMSKeyId) && {
            "x-amz-fwd-header-x-amz-server-side-encryption-aws-kms-key-id": input.SSEKMSKeyId,
        }),
        ...(isSerializableHeaderValue(input.SSECustomerKeyMD5) && {
            "x-amz-fwd-header-x-amz-server-side-encryption-customer-key-md5": input.SSECustomerKeyMD5,
        }),
        ...(isSerializableHeaderValue(input.StorageClass) && {
            "x-amz-fwd-header-x-amz-storage-class": input.StorageClass,
        }),
        ...(isSerializableHeaderValue(input.TagCount) && {
            "x-amz-fwd-header-x-amz-tagging-count": input.TagCount.toString(),
        }),
        ...(isSerializableHeaderValue(input.VersionId) && { "x-amz-fwd-header-x-amz-version-id": input.VersionId }),
        ...(isSerializableHeaderValue(input.BucketKeyEnabled) && {
            "x-amz-fwd-header-x-amz-server-side-encryption-bucket-key-enabled": input.BucketKeyEnabled.toString(),
        }),
        ...(input.Metadata !== undefined &&
            Object.keys(input.Metadata).reduce((acc, suffix) => ({
                ...acc,
                [`x-amz-meta-${suffix.toLowerCase()}`]: input.Metadata[suffix],
            }), {})),
    };
    let resolvedPath = `${basePath?.endsWith("/") ? basePath.slice(0, -1) : basePath || ""}` + "/WriteGetObjectResponse";
    const query = {
        "x-id": "WriteGetObjectResponse",
    };
    let body;
    if (input.Body !== undefined) {
        body = input.Body;
    }
    let contents;
    if (input.Body !== undefined) {
        contents = input.Body;
        body = contents;
    }
    let { hostname: resolvedHostname } = await context.endpoint();
    if (context.disableHostPrefix !== true) {
        resolvedHostname = "{RequestRoute}." + resolvedHostname;
        if (input.RequestRoute === undefined) {
            throw new Error("Empty value provided for input host prefix: RequestRoute.");
        }
        resolvedHostname = resolvedHostname.replace("{RequestRoute}", input.RequestRoute);
        if (!__isValidHostname(resolvedHostname)) {
            throw new Error("ValidationError: prefixed hostname must be hostname compatible.");
        }
    }
    return new __HttpRequest({
        protocol,
        hostname: resolvedHostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const deserializeAws_restXmlAbortMultipartUploadCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlAbortMultipartUploadCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlAbortMultipartUploadCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NoSuchUpload":
        case "com.amazonaws.s3#NoSuchUpload":
            response = {
                ...(await deserializeAws_restXmlNoSuchUploadResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlCompleteMultipartUploadCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlCompleteMultipartUploadCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Bucket: undefined,
        BucketKeyEnabled: undefined,
        ETag: undefined,
        Expiration: undefined,
        Key: undefined,
        Location: undefined,
        RequestCharged: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
        VersionId: undefined,
    };
    if (output.headers["x-amz-expiration"] !== undefined) {
        contents.Expiration = output.headers["x-amz-expiration"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    if (data["Bucket"] !== undefined) {
        contents.Bucket = __expectString(data["Bucket"]);
    }
    if (data["ETag"] !== undefined) {
        contents.ETag = __expectString(data["ETag"]);
    }
    if (data["Key"] !== undefined) {
        contents.Key = __expectString(data["Key"]);
    }
    if (data["Location"] !== undefined) {
        contents.Location = __expectString(data["Location"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlCompleteMultipartUploadCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlCopyObjectCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlCopyObjectCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        BucketKeyEnabled: undefined,
        CopyObjectResult: undefined,
        CopySourceVersionId: undefined,
        Expiration: undefined,
        RequestCharged: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSEncryptionContext: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
        VersionId: undefined,
    };
    if (output.headers["x-amz-expiration"] !== undefined) {
        contents.Expiration = output.headers["x-amz-expiration"];
    }
    if (output.headers["x-amz-copy-source-version-id"] !== undefined) {
        contents.CopySourceVersionId = output.headers["x-amz-copy-source-version-id"];
    }
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-context"] !== undefined) {
        contents.SSEKMSEncryptionContext = output.headers["x-amz-server-side-encryption-context"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    contents.CopyObjectResult = deserializeAws_restXmlCopyObjectResult(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlCopyObjectCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ObjectNotInActiveTierError":
        case "com.amazonaws.s3#ObjectNotInActiveTierError":
            response = {
                ...(await deserializeAws_restXmlObjectNotInActiveTierErrorResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlCreateBucketCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlCreateBucketCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Location: undefined,
    };
    if (output.headers["location"] !== undefined) {
        contents.Location = output.headers["location"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlCreateBucketCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "BucketAlreadyExists":
        case "com.amazonaws.s3#BucketAlreadyExists":
            response = {
                ...(await deserializeAws_restXmlBucketAlreadyExistsResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "BucketAlreadyOwnedByYou":
        case "com.amazonaws.s3#BucketAlreadyOwnedByYou":
            response = {
                ...(await deserializeAws_restXmlBucketAlreadyOwnedByYouResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlCreateMultipartUploadCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlCreateMultipartUploadCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        AbortDate: undefined,
        AbortRuleId: undefined,
        Bucket: undefined,
        BucketKeyEnabled: undefined,
        Key: undefined,
        RequestCharged: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSEncryptionContext: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
        UploadId: undefined,
    };
    if (output.headers["x-amz-abort-date"] !== undefined) {
        contents.AbortDate = new Date(output.headers["x-amz-abort-date"]);
    }
    if (output.headers["x-amz-abort-rule-id"] !== undefined) {
        contents.AbortRuleId = output.headers["x-amz-abort-rule-id"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-context"] !== undefined) {
        contents.SSEKMSEncryptionContext = output.headers["x-amz-server-side-encryption-context"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    if (data["Bucket"] !== undefined) {
        contents.Bucket = __expectString(data["Bucket"]);
    }
    if (data["Key"] !== undefined) {
        contents.Key = __expectString(data["Key"]);
    }
    if (data["UploadId"] !== undefined) {
        contents.UploadId = __expectString(data["UploadId"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlCreateMultipartUploadCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketAnalyticsConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketAnalyticsConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketAnalyticsConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketCorsCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketCorsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketCorsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketEncryptionCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketEncryptionCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketEncryptionCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketIntelligentTieringConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketIntelligentTieringConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketIntelligentTieringConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketInventoryConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketInventoryConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketInventoryConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketLifecycleCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketLifecycleCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketLifecycleCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketMetricsConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketMetricsConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketMetricsConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketOwnershipControlsCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketOwnershipControlsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketOwnershipControlsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketPolicyCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketPolicyCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketPolicyCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketReplicationCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketReplicationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketReplicationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketTaggingCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketTaggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketTaggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteBucketWebsiteCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteBucketWebsiteCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteBucketWebsiteCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteObjectCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteObjectCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        DeleteMarker: undefined,
        RequestCharged: undefined,
        VersionId: undefined,
    };
    if (output.headers["x-amz-delete-marker"] !== undefined) {
        contents.DeleteMarker = __parseBoolean(output.headers["x-amz-delete-marker"]);
    }
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteObjectCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteObjectsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteObjectsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Deleted: undefined,
        Errors: undefined,
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    if (data.Deleted === "") {
        contents.Deleted = [];
    }
    if (data["Deleted"] !== undefined) {
        contents.Deleted = deserializeAws_restXmlDeletedObjects(__getArrayIfSingleItem(data["Deleted"]), context);
    }
    if (data.Error === "") {
        contents.Errors = [];
    }
    if (data["Error"] !== undefined) {
        contents.Errors = deserializeAws_restXmlErrors(__getArrayIfSingleItem(data["Error"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteObjectsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeleteObjectTaggingCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeleteObjectTaggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        VersionId: undefined,
    };
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeleteObjectTaggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlDeletePublicAccessBlockCommand = async (output, context) => {
    if (output.statusCode !== 204 && output.statusCode >= 300) {
        return deserializeAws_restXmlDeletePublicAccessBlockCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlDeletePublicAccessBlockCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketAccelerateConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketAccelerateConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Status: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["Status"] !== undefined) {
        contents.Status = __expectString(data["Status"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketAccelerateConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketAclCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketAclCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Grants: undefined,
        Owner: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.AccessControlList === "") {
        contents.Grants = [];
    }
    if (data["AccessControlList"] !== undefined && data["AccessControlList"]["Grant"] !== undefined) {
        contents.Grants = deserializeAws_restXmlGrants(__getArrayIfSingleItem(data["AccessControlList"]["Grant"]), context);
    }
    if (data["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(data["Owner"], context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketAclCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketAnalyticsConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketAnalyticsConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        AnalyticsConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.AnalyticsConfiguration = deserializeAws_restXmlAnalyticsConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketAnalyticsConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketCorsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketCorsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        CORSRules: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.CORSRule === "") {
        contents.CORSRules = [];
    }
    if (data["CORSRule"] !== undefined) {
        contents.CORSRules = deserializeAws_restXmlCORSRules(__getArrayIfSingleItem(data["CORSRule"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketCorsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketEncryptionCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketEncryptionCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ServerSideEncryptionConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.ServerSideEncryptionConfiguration = deserializeAws_restXmlServerSideEncryptionConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketEncryptionCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketIntelligentTieringConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketIntelligentTieringConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        IntelligentTieringConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.IntelligentTieringConfiguration = deserializeAws_restXmlIntelligentTieringConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketIntelligentTieringConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketInventoryConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketInventoryConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        InventoryConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.InventoryConfiguration = deserializeAws_restXmlInventoryConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketInventoryConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketLifecycleConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketLifecycleConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Rules: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.Rule === "") {
        contents.Rules = [];
    }
    if (data["Rule"] !== undefined) {
        contents.Rules = deserializeAws_restXmlLifecycleRules(__getArrayIfSingleItem(data["Rule"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketLifecycleConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketLocationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketLocationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        LocationConstraint: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["LocationConstraint"] !== undefined) {
        contents.LocationConstraint = __expectString(data["LocationConstraint"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketLocationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketLoggingCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketLoggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        LoggingEnabled: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["LoggingEnabled"] !== undefined) {
        contents.LoggingEnabled = deserializeAws_restXmlLoggingEnabled(data["LoggingEnabled"], context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketLoggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketMetricsConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketMetricsConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        MetricsConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.MetricsConfiguration = deserializeAws_restXmlMetricsConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketMetricsConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketNotificationConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketNotificationConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        LambdaFunctionConfigurations: undefined,
        QueueConfigurations: undefined,
        TopicConfigurations: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.CloudFunctionConfiguration === "") {
        contents.LambdaFunctionConfigurations = [];
    }
    if (data["CloudFunctionConfiguration"] !== undefined) {
        contents.LambdaFunctionConfigurations = deserializeAws_restXmlLambdaFunctionConfigurationList(__getArrayIfSingleItem(data["CloudFunctionConfiguration"]), context);
    }
    if (data.QueueConfiguration === "") {
        contents.QueueConfigurations = [];
    }
    if (data["QueueConfiguration"] !== undefined) {
        contents.QueueConfigurations = deserializeAws_restXmlQueueConfigurationList(__getArrayIfSingleItem(data["QueueConfiguration"]), context);
    }
    if (data.TopicConfiguration === "") {
        contents.TopicConfigurations = [];
    }
    if (data["TopicConfiguration"] !== undefined) {
        contents.TopicConfigurations = deserializeAws_restXmlTopicConfigurationList(__getArrayIfSingleItem(data["TopicConfiguration"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketNotificationConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketOwnershipControlsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketOwnershipControlsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        OwnershipControls: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.OwnershipControls = deserializeAws_restXmlOwnershipControls(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketOwnershipControlsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketPolicyCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketPolicyCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Policy: undefined,
    };
    const data = await collectBodyString(output.body, context);
    contents.Policy = __expectString(data);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketPolicyCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketPolicyStatusCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketPolicyStatusCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        PolicyStatus: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.PolicyStatus = deserializeAws_restXmlPolicyStatus(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketPolicyStatusCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketReplicationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketReplicationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ReplicationConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.ReplicationConfiguration = deserializeAws_restXmlReplicationConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketReplicationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketRequestPaymentCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketRequestPaymentCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Payer: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["Payer"] !== undefined) {
        contents.Payer = __expectString(data["Payer"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketRequestPaymentCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketTaggingCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketTaggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        TagSet: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.TagSet === "") {
        contents.TagSet = [];
    }
    if (data["TagSet"] !== undefined && data["TagSet"]["Tag"] !== undefined) {
        contents.TagSet = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(data["TagSet"]["Tag"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketTaggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketVersioningCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketVersioningCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        MFADelete: undefined,
        Status: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["MfaDelete"] !== undefined) {
        contents.MFADelete = __expectString(data["MfaDelete"]);
    }
    if (data["Status"] !== undefined) {
        contents.Status = __expectString(data["Status"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketVersioningCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetBucketWebsiteCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetBucketWebsiteCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ErrorDocument: undefined,
        IndexDocument: undefined,
        RedirectAllRequestsTo: undefined,
        RoutingRules: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["ErrorDocument"] !== undefined) {
        contents.ErrorDocument = deserializeAws_restXmlErrorDocument(data["ErrorDocument"], context);
    }
    if (data["IndexDocument"] !== undefined) {
        contents.IndexDocument = deserializeAws_restXmlIndexDocument(data["IndexDocument"], context);
    }
    if (data["RedirectAllRequestsTo"] !== undefined) {
        contents.RedirectAllRequestsTo = deserializeAws_restXmlRedirectAllRequestsTo(data["RedirectAllRequestsTo"], context);
    }
    if (data.RoutingRules === "") {
        contents.RoutingRules = [];
    }
    if (data["RoutingRules"] !== undefined && data["RoutingRules"]["RoutingRule"] !== undefined) {
        contents.RoutingRules = deserializeAws_restXmlRoutingRules(__getArrayIfSingleItem(data["RoutingRules"]["RoutingRule"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetBucketWebsiteCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetObjectCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        AcceptRanges: undefined,
        Body: undefined,
        BucketKeyEnabled: undefined,
        CacheControl: undefined,
        ContentDisposition: undefined,
        ContentEncoding: undefined,
        ContentLanguage: undefined,
        ContentLength: undefined,
        ContentRange: undefined,
        ContentType: undefined,
        DeleteMarker: undefined,
        ETag: undefined,
        Expiration: undefined,
        Expires: undefined,
        LastModified: undefined,
        Metadata: undefined,
        MissingMeta: undefined,
        ObjectLockLegalHoldStatus: undefined,
        ObjectLockMode: undefined,
        ObjectLockRetainUntilDate: undefined,
        PartsCount: undefined,
        ReplicationStatus: undefined,
        RequestCharged: undefined,
        Restore: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
        StorageClass: undefined,
        TagCount: undefined,
        VersionId: undefined,
        WebsiteRedirectLocation: undefined,
    };
    if (output.headers["x-amz-delete-marker"] !== undefined) {
        contents.DeleteMarker = __parseBoolean(output.headers["x-amz-delete-marker"]);
    }
    if (output.headers["accept-ranges"] !== undefined) {
        contents.AcceptRanges = output.headers["accept-ranges"];
    }
    if (output.headers["x-amz-expiration"] !== undefined) {
        contents.Expiration = output.headers["x-amz-expiration"];
    }
    if (output.headers["x-amz-restore"] !== undefined) {
        contents.Restore = output.headers["x-amz-restore"];
    }
    if (output.headers["last-modified"] !== undefined) {
        contents.LastModified = new Date(output.headers["last-modified"]);
    }
    if (output.headers["content-length"] !== undefined) {
        contents.ContentLength = parseInt(output.headers["content-length"], 10);
    }
    if (output.headers["etag"] !== undefined) {
        contents.ETag = output.headers["etag"];
    }
    if (output.headers["x-amz-missing-meta"] !== undefined) {
        contents.MissingMeta = parseInt(output.headers["x-amz-missing-meta"], 10);
    }
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    if (output.headers["cache-control"] !== undefined) {
        contents.CacheControl = output.headers["cache-control"];
    }
    if (output.headers["content-disposition"] !== undefined) {
        contents.ContentDisposition = output.headers["content-disposition"];
    }
    if (output.headers["content-encoding"] !== undefined) {
        contents.ContentEncoding = output.headers["content-encoding"];
    }
    if (output.headers["content-language"] !== undefined) {
        contents.ContentLanguage = output.headers["content-language"];
    }
    if (output.headers["content-range"] !== undefined) {
        contents.ContentRange = output.headers["content-range"];
    }
    if (output.headers["content-type"] !== undefined) {
        contents.ContentType = output.headers["content-type"];
    }
    if (output.headers["expires"] !== undefined) {
        contents.Expires = new Date(output.headers["expires"]);
    }
    if (output.headers["x-amz-website-redirect-location"] !== undefined) {
        contents.WebsiteRedirectLocation = output.headers["x-amz-website-redirect-location"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-storage-class"] !== undefined) {
        contents.StorageClass = output.headers["x-amz-storage-class"];
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    if (output.headers["x-amz-replication-status"] !== undefined) {
        contents.ReplicationStatus = output.headers["x-amz-replication-status"];
    }
    if (output.headers["x-amz-mp-parts-count"] !== undefined) {
        contents.PartsCount = parseInt(output.headers["x-amz-mp-parts-count"], 10);
    }
    if (output.headers["x-amz-tagging-count"] !== undefined) {
        contents.TagCount = parseInt(output.headers["x-amz-tagging-count"], 10);
    }
    if (output.headers["x-amz-object-lock-mode"] !== undefined) {
        contents.ObjectLockMode = output.headers["x-amz-object-lock-mode"];
    }
    if (output.headers["x-amz-object-lock-retain-until-date"] !== undefined) {
        contents.ObjectLockRetainUntilDate = new Date(output.headers["x-amz-object-lock-retain-until-date"]);
    }
    if (output.headers["x-amz-object-lock-legal-hold"] !== undefined) {
        contents.ObjectLockLegalHoldStatus = output.headers["x-amz-object-lock-legal-hold"];
    }
    Object.keys(output.headers).forEach((header) => {
        if (contents.Metadata === undefined) {
            contents.Metadata = {};
        }
        if (header.startsWith("x-amz-meta-")) {
            contents.Metadata[header.substring(11)] = output.headers[header];
        }
    });
    const data = output.body;
    contents.Body = data;
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidObjectState":
        case "com.amazonaws.s3#InvalidObjectState":
            response = {
                ...(await deserializeAws_restXmlInvalidObjectStateResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "NoSuchKey":
        case "com.amazonaws.s3#NoSuchKey":
            response = {
                ...(await deserializeAws_restXmlNoSuchKeyResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlGetObjectAclCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectAclCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Grants: undefined,
        Owner: undefined,
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    if (data.AccessControlList === "") {
        contents.Grants = [];
    }
    if (data["AccessControlList"] !== undefined && data["AccessControlList"]["Grant"] !== undefined) {
        contents.Grants = deserializeAws_restXmlGrants(__getArrayIfSingleItem(data["AccessControlList"]["Grant"]), context);
    }
    if (data["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(data["Owner"], context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectAclCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NoSuchKey":
        case "com.amazonaws.s3#NoSuchKey":
            response = {
                ...(await deserializeAws_restXmlNoSuchKeyResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlGetObjectLegalHoldCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectLegalHoldCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        LegalHold: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.LegalHold = deserializeAws_restXmlObjectLockLegalHold(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectLegalHoldCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetObjectLockConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectLockConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ObjectLockConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.ObjectLockConfiguration = deserializeAws_restXmlObjectLockConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectLockConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetObjectRetentionCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectRetentionCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Retention: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.Retention = deserializeAws_restXmlObjectLockRetention(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectRetentionCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetObjectTaggingCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectTaggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        TagSet: undefined,
        VersionId: undefined,
    };
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    const data = await parseBody(output.body, context);
    if (data.TagSet === "") {
        contents.TagSet = [];
    }
    if (data["TagSet"] !== undefined && data["TagSet"]["Tag"] !== undefined) {
        contents.TagSet = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(data["TagSet"]["Tag"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectTaggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetObjectTorrentCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetObjectTorrentCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Body: undefined,
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = output.body;
    contents.Body = data;
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetObjectTorrentCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlGetPublicAccessBlockCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlGetPublicAccessBlockCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        PublicAccessBlockConfiguration: undefined,
    };
    const data = await parseBody(output.body, context);
    contents.PublicAccessBlockConfiguration = deserializeAws_restXmlPublicAccessBlockConfiguration(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlGetPublicAccessBlockCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlHeadBucketCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlHeadBucketCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlHeadBucketCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NotFound":
        case "com.amazonaws.s3#NotFound":
            response = {
                ...(await deserializeAws_restXmlNotFoundResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlHeadObjectCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlHeadObjectCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        AcceptRanges: undefined,
        ArchiveStatus: undefined,
        BucketKeyEnabled: undefined,
        CacheControl: undefined,
        ContentDisposition: undefined,
        ContentEncoding: undefined,
        ContentLanguage: undefined,
        ContentLength: undefined,
        ContentType: undefined,
        DeleteMarker: undefined,
        ETag: undefined,
        Expiration: undefined,
        Expires: undefined,
        LastModified: undefined,
        Metadata: undefined,
        MissingMeta: undefined,
        ObjectLockLegalHoldStatus: undefined,
        ObjectLockMode: undefined,
        ObjectLockRetainUntilDate: undefined,
        PartsCount: undefined,
        ReplicationStatus: undefined,
        RequestCharged: undefined,
        Restore: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
        StorageClass: undefined,
        VersionId: undefined,
        WebsiteRedirectLocation: undefined,
    };
    if (output.headers["x-amz-delete-marker"] !== undefined) {
        contents.DeleteMarker = __parseBoolean(output.headers["x-amz-delete-marker"]);
    }
    if (output.headers["accept-ranges"] !== undefined) {
        contents.AcceptRanges = output.headers["accept-ranges"];
    }
    if (output.headers["x-amz-expiration"] !== undefined) {
        contents.Expiration = output.headers["x-amz-expiration"];
    }
    if (output.headers["x-amz-restore"] !== undefined) {
        contents.Restore = output.headers["x-amz-restore"];
    }
    if (output.headers["x-amz-archive-status"] !== undefined) {
        contents.ArchiveStatus = output.headers["x-amz-archive-status"];
    }
    if (output.headers["last-modified"] !== undefined) {
        contents.LastModified = new Date(output.headers["last-modified"]);
    }
    if (output.headers["content-length"] !== undefined) {
        contents.ContentLength = parseInt(output.headers["content-length"], 10);
    }
    if (output.headers["etag"] !== undefined) {
        contents.ETag = output.headers["etag"];
    }
    if (output.headers["x-amz-missing-meta"] !== undefined) {
        contents.MissingMeta = parseInt(output.headers["x-amz-missing-meta"], 10);
    }
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    if (output.headers["cache-control"] !== undefined) {
        contents.CacheControl = output.headers["cache-control"];
    }
    if (output.headers["content-disposition"] !== undefined) {
        contents.ContentDisposition = output.headers["content-disposition"];
    }
    if (output.headers["content-encoding"] !== undefined) {
        contents.ContentEncoding = output.headers["content-encoding"];
    }
    if (output.headers["content-language"] !== undefined) {
        contents.ContentLanguage = output.headers["content-language"];
    }
    if (output.headers["content-type"] !== undefined) {
        contents.ContentType = output.headers["content-type"];
    }
    if (output.headers["expires"] !== undefined) {
        contents.Expires = new Date(output.headers["expires"]);
    }
    if (output.headers["x-amz-website-redirect-location"] !== undefined) {
        contents.WebsiteRedirectLocation = output.headers["x-amz-website-redirect-location"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-storage-class"] !== undefined) {
        contents.StorageClass = output.headers["x-amz-storage-class"];
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    if (output.headers["x-amz-replication-status"] !== undefined) {
        contents.ReplicationStatus = output.headers["x-amz-replication-status"];
    }
    if (output.headers["x-amz-mp-parts-count"] !== undefined) {
        contents.PartsCount = parseInt(output.headers["x-amz-mp-parts-count"], 10);
    }
    if (output.headers["x-amz-object-lock-mode"] !== undefined) {
        contents.ObjectLockMode = output.headers["x-amz-object-lock-mode"];
    }
    if (output.headers["x-amz-object-lock-retain-until-date"] !== undefined) {
        contents.ObjectLockRetainUntilDate = new Date(output.headers["x-amz-object-lock-retain-until-date"]);
    }
    if (output.headers["x-amz-object-lock-legal-hold"] !== undefined) {
        contents.ObjectLockLegalHoldStatus = output.headers["x-amz-object-lock-legal-hold"];
    }
    Object.keys(output.headers).forEach((header) => {
        if (contents.Metadata === undefined) {
            contents.Metadata = {};
        }
        if (header.startsWith("x-amz-meta-")) {
            contents.Metadata[header.substring(11)] = output.headers[header];
        }
    });
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlHeadObjectCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NotFound":
        case "com.amazonaws.s3#NotFound":
            response = {
                ...(await deserializeAws_restXmlNotFoundResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlListBucketAnalyticsConfigurationsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListBucketAnalyticsConfigurationsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        AnalyticsConfigurationList: undefined,
        ContinuationToken: undefined,
        IsTruncated: undefined,
        NextContinuationToken: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.AnalyticsConfiguration === "") {
        contents.AnalyticsConfigurationList = [];
    }
    if (data["AnalyticsConfiguration"] !== undefined) {
        contents.AnalyticsConfigurationList = deserializeAws_restXmlAnalyticsConfigurationList(__getArrayIfSingleItem(data["AnalyticsConfiguration"]), context);
    }
    if (data["ContinuationToken"] !== undefined) {
        contents.ContinuationToken = __expectString(data["ContinuationToken"]);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["NextContinuationToken"] !== undefined) {
        contents.NextContinuationToken = __expectString(data["NextContinuationToken"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListBucketAnalyticsConfigurationsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListBucketIntelligentTieringConfigurationsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListBucketIntelligentTieringConfigurationsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ContinuationToken: undefined,
        IntelligentTieringConfigurationList: undefined,
        IsTruncated: undefined,
        NextContinuationToken: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["ContinuationToken"] !== undefined) {
        contents.ContinuationToken = __expectString(data["ContinuationToken"]);
    }
    if (data.IntelligentTieringConfiguration === "") {
        contents.IntelligentTieringConfigurationList = [];
    }
    if (data["IntelligentTieringConfiguration"] !== undefined) {
        contents.IntelligentTieringConfigurationList = deserializeAws_restXmlIntelligentTieringConfigurationList(__getArrayIfSingleItem(data["IntelligentTieringConfiguration"]), context);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["NextContinuationToken"] !== undefined) {
        contents.NextContinuationToken = __expectString(data["NextContinuationToken"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListBucketIntelligentTieringConfigurationsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListBucketInventoryConfigurationsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListBucketInventoryConfigurationsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ContinuationToken: undefined,
        InventoryConfigurationList: undefined,
        IsTruncated: undefined,
        NextContinuationToken: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["ContinuationToken"] !== undefined) {
        contents.ContinuationToken = __expectString(data["ContinuationToken"]);
    }
    if (data.InventoryConfiguration === "") {
        contents.InventoryConfigurationList = [];
    }
    if (data["InventoryConfiguration"] !== undefined) {
        contents.InventoryConfigurationList = deserializeAws_restXmlInventoryConfigurationList(__getArrayIfSingleItem(data["InventoryConfiguration"]), context);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["NextContinuationToken"] !== undefined) {
        contents.NextContinuationToken = __expectString(data["NextContinuationToken"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListBucketInventoryConfigurationsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListBucketMetricsConfigurationsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListBucketMetricsConfigurationsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        ContinuationToken: undefined,
        IsTruncated: undefined,
        MetricsConfigurationList: undefined,
        NextContinuationToken: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["ContinuationToken"] !== undefined) {
        contents.ContinuationToken = __expectString(data["ContinuationToken"]);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data.MetricsConfiguration === "") {
        contents.MetricsConfigurationList = [];
    }
    if (data["MetricsConfiguration"] !== undefined) {
        contents.MetricsConfigurationList = deserializeAws_restXmlMetricsConfigurationList(__getArrayIfSingleItem(data["MetricsConfiguration"]), context);
    }
    if (data["NextContinuationToken"] !== undefined) {
        contents.NextContinuationToken = __expectString(data["NextContinuationToken"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListBucketMetricsConfigurationsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListBucketsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListBucketsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Buckets: undefined,
        Owner: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.Buckets === "") {
        contents.Buckets = [];
    }
    if (data["Buckets"] !== undefined && data["Buckets"]["Bucket"] !== undefined) {
        contents.Buckets = deserializeAws_restXmlBuckets(__getArrayIfSingleItem(data["Buckets"]["Bucket"]), context);
    }
    if (data["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(data["Owner"], context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListBucketsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListMultipartUploadsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListMultipartUploadsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Bucket: undefined,
        CommonPrefixes: undefined,
        Delimiter: undefined,
        EncodingType: undefined,
        IsTruncated: undefined,
        KeyMarker: undefined,
        MaxUploads: undefined,
        NextKeyMarker: undefined,
        NextUploadIdMarker: undefined,
        Prefix: undefined,
        UploadIdMarker: undefined,
        Uploads: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data["Bucket"] !== undefined) {
        contents.Bucket = __expectString(data["Bucket"]);
    }
    if (data.CommonPrefixes === "") {
        contents.CommonPrefixes = [];
    }
    if (data["CommonPrefixes"] !== undefined) {
        contents.CommonPrefixes = deserializeAws_restXmlCommonPrefixList(__getArrayIfSingleItem(data["CommonPrefixes"]), context);
    }
    if (data["Delimiter"] !== undefined) {
        contents.Delimiter = __expectString(data["Delimiter"]);
    }
    if (data["EncodingType"] !== undefined) {
        contents.EncodingType = __expectString(data["EncodingType"]);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["KeyMarker"] !== undefined) {
        contents.KeyMarker = __expectString(data["KeyMarker"]);
    }
    if (data["MaxUploads"] !== undefined) {
        contents.MaxUploads = parseInt(data["MaxUploads"]);
    }
    if (data["NextKeyMarker"] !== undefined) {
        contents.NextKeyMarker = __expectString(data["NextKeyMarker"]);
    }
    if (data["NextUploadIdMarker"] !== undefined) {
        contents.NextUploadIdMarker = __expectString(data["NextUploadIdMarker"]);
    }
    if (data["Prefix"] !== undefined) {
        contents.Prefix = __expectString(data["Prefix"]);
    }
    if (data["UploadIdMarker"] !== undefined) {
        contents.UploadIdMarker = __expectString(data["UploadIdMarker"]);
    }
    if (data.Upload === "") {
        contents.Uploads = [];
    }
    if (data["Upload"] !== undefined) {
        contents.Uploads = deserializeAws_restXmlMultipartUploadList(__getArrayIfSingleItem(data["Upload"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListMultipartUploadsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListObjectsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListObjectsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        CommonPrefixes: undefined,
        Contents: undefined,
        Delimiter: undefined,
        EncodingType: undefined,
        IsTruncated: undefined,
        Marker: undefined,
        MaxKeys: undefined,
        Name: undefined,
        NextMarker: undefined,
        Prefix: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.CommonPrefixes === "") {
        contents.CommonPrefixes = [];
    }
    if (data["CommonPrefixes"] !== undefined) {
        contents.CommonPrefixes = deserializeAws_restXmlCommonPrefixList(__getArrayIfSingleItem(data["CommonPrefixes"]), context);
    }
    if (data.Contents === "") {
        contents.Contents = [];
    }
    if (data["Contents"] !== undefined) {
        contents.Contents = deserializeAws_restXmlObjectList(__getArrayIfSingleItem(data["Contents"]), context);
    }
    if (data["Delimiter"] !== undefined) {
        contents.Delimiter = __expectString(data["Delimiter"]);
    }
    if (data["EncodingType"] !== undefined) {
        contents.EncodingType = __expectString(data["EncodingType"]);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["Marker"] !== undefined) {
        contents.Marker = __expectString(data["Marker"]);
    }
    if (data["MaxKeys"] !== undefined) {
        contents.MaxKeys = parseInt(data["MaxKeys"]);
    }
    if (data["Name"] !== undefined) {
        contents.Name = __expectString(data["Name"]);
    }
    if (data["NextMarker"] !== undefined) {
        contents.NextMarker = __expectString(data["NextMarker"]);
    }
    if (data["Prefix"] !== undefined) {
        contents.Prefix = __expectString(data["Prefix"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListObjectsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NoSuchBucket":
        case "com.amazonaws.s3#NoSuchBucket":
            response = {
                ...(await deserializeAws_restXmlNoSuchBucketResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlListObjectsV2Command = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListObjectsV2CommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        CommonPrefixes: undefined,
        Contents: undefined,
        ContinuationToken: undefined,
        Delimiter: undefined,
        EncodingType: undefined,
        IsTruncated: undefined,
        KeyCount: undefined,
        MaxKeys: undefined,
        Name: undefined,
        NextContinuationToken: undefined,
        Prefix: undefined,
        StartAfter: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.CommonPrefixes === "") {
        contents.CommonPrefixes = [];
    }
    if (data["CommonPrefixes"] !== undefined) {
        contents.CommonPrefixes = deserializeAws_restXmlCommonPrefixList(__getArrayIfSingleItem(data["CommonPrefixes"]), context);
    }
    if (data.Contents === "") {
        contents.Contents = [];
    }
    if (data["Contents"] !== undefined) {
        contents.Contents = deserializeAws_restXmlObjectList(__getArrayIfSingleItem(data["Contents"]), context);
    }
    if (data["ContinuationToken"] !== undefined) {
        contents.ContinuationToken = __expectString(data["ContinuationToken"]);
    }
    if (data["Delimiter"] !== undefined) {
        contents.Delimiter = __expectString(data["Delimiter"]);
    }
    if (data["EncodingType"] !== undefined) {
        contents.EncodingType = __expectString(data["EncodingType"]);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["KeyCount"] !== undefined) {
        contents.KeyCount = parseInt(data["KeyCount"]);
    }
    if (data["MaxKeys"] !== undefined) {
        contents.MaxKeys = parseInt(data["MaxKeys"]);
    }
    if (data["Name"] !== undefined) {
        contents.Name = __expectString(data["Name"]);
    }
    if (data["NextContinuationToken"] !== undefined) {
        contents.NextContinuationToken = __expectString(data["NextContinuationToken"]);
    }
    if (data["Prefix"] !== undefined) {
        contents.Prefix = __expectString(data["Prefix"]);
    }
    if (data["StartAfter"] !== undefined) {
        contents.StartAfter = __expectString(data["StartAfter"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListObjectsV2CommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NoSuchBucket":
        case "com.amazonaws.s3#NoSuchBucket":
            response = {
                ...(await deserializeAws_restXmlNoSuchBucketResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlListObjectVersionsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListObjectVersionsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        CommonPrefixes: undefined,
        DeleteMarkers: undefined,
        Delimiter: undefined,
        EncodingType: undefined,
        IsTruncated: undefined,
        KeyMarker: undefined,
        MaxKeys: undefined,
        Name: undefined,
        NextKeyMarker: undefined,
        NextVersionIdMarker: undefined,
        Prefix: undefined,
        VersionIdMarker: undefined,
        Versions: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.CommonPrefixes === "") {
        contents.CommonPrefixes = [];
    }
    if (data["CommonPrefixes"] !== undefined) {
        contents.CommonPrefixes = deserializeAws_restXmlCommonPrefixList(__getArrayIfSingleItem(data["CommonPrefixes"]), context);
    }
    if (data.DeleteMarker === "") {
        contents.DeleteMarkers = [];
    }
    if (data["DeleteMarker"] !== undefined) {
        contents.DeleteMarkers = deserializeAws_restXmlDeleteMarkers(__getArrayIfSingleItem(data["DeleteMarker"]), context);
    }
    if (data["Delimiter"] !== undefined) {
        contents.Delimiter = __expectString(data["Delimiter"]);
    }
    if (data["EncodingType"] !== undefined) {
        contents.EncodingType = __expectString(data["EncodingType"]);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["KeyMarker"] !== undefined) {
        contents.KeyMarker = __expectString(data["KeyMarker"]);
    }
    if (data["MaxKeys"] !== undefined) {
        contents.MaxKeys = parseInt(data["MaxKeys"]);
    }
    if (data["Name"] !== undefined) {
        contents.Name = __expectString(data["Name"]);
    }
    if (data["NextKeyMarker"] !== undefined) {
        contents.NextKeyMarker = __expectString(data["NextKeyMarker"]);
    }
    if (data["NextVersionIdMarker"] !== undefined) {
        contents.NextVersionIdMarker = __expectString(data["NextVersionIdMarker"]);
    }
    if (data["Prefix"] !== undefined) {
        contents.Prefix = __expectString(data["Prefix"]);
    }
    if (data["VersionIdMarker"] !== undefined) {
        contents.VersionIdMarker = __expectString(data["VersionIdMarker"]);
    }
    if (data.Version === "") {
        contents.Versions = [];
    }
    if (data["Version"] !== undefined) {
        contents.Versions = deserializeAws_restXmlObjectVersionList(__getArrayIfSingleItem(data["Version"]), context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListObjectVersionsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlListPartsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlListPartsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        AbortDate: undefined,
        AbortRuleId: undefined,
        Bucket: undefined,
        Initiator: undefined,
        IsTruncated: undefined,
        Key: undefined,
        MaxParts: undefined,
        NextPartNumberMarker: undefined,
        Owner: undefined,
        PartNumberMarker: undefined,
        Parts: undefined,
        RequestCharged: undefined,
        StorageClass: undefined,
        UploadId: undefined,
    };
    if (output.headers["x-amz-abort-date"] !== undefined) {
        contents.AbortDate = new Date(output.headers["x-amz-abort-date"]);
    }
    if (output.headers["x-amz-abort-rule-id"] !== undefined) {
        contents.AbortRuleId = output.headers["x-amz-abort-rule-id"];
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    if (data["Bucket"] !== undefined) {
        contents.Bucket = __expectString(data["Bucket"]);
    }
    if (data["Initiator"] !== undefined) {
        contents.Initiator = deserializeAws_restXmlInitiator(data["Initiator"], context);
    }
    if (data["IsTruncated"] !== undefined) {
        contents.IsTruncated = __parseBoolean(data["IsTruncated"]);
    }
    if (data["Key"] !== undefined) {
        contents.Key = __expectString(data["Key"]);
    }
    if (data["MaxParts"] !== undefined) {
        contents.MaxParts = parseInt(data["MaxParts"]);
    }
    if (data["NextPartNumberMarker"] !== undefined) {
        contents.NextPartNumberMarker = __expectString(data["NextPartNumberMarker"]);
    }
    if (data["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(data["Owner"], context);
    }
    if (data["PartNumberMarker"] !== undefined) {
        contents.PartNumberMarker = __expectString(data["PartNumberMarker"]);
    }
    if (data.Part === "") {
        contents.Parts = [];
    }
    if (data["Part"] !== undefined) {
        contents.Parts = deserializeAws_restXmlParts(__getArrayIfSingleItem(data["Part"]), context);
    }
    if (data["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(data["StorageClass"]);
    }
    if (data["UploadId"] !== undefined) {
        contents.UploadId = __expectString(data["UploadId"]);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restXmlListPartsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketAccelerateConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketAccelerateConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketAccelerateConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketAclCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketAclCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketAclCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketAnalyticsConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketAnalyticsConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketAnalyticsConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketCorsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketCorsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketCorsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketEncryptionCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketEncryptionCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketEncryptionCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketIntelligentTieringConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketIntelligentTieringConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketIntelligentTieringConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketInventoryConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketInventoryConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketInventoryConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketLifecycleConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketLifecycleConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketLifecycleConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketLoggingCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketLoggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketLoggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketMetricsConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketMetricsConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketMetricsConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketNotificationConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketNotificationConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketNotificationConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketOwnershipControlsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketOwnershipControlsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketOwnershipControlsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketPolicyCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketPolicyCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketPolicyCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketReplicationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketReplicationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketReplicationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketRequestPaymentCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketRequestPaymentCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketRequestPaymentCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketTaggingCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketTaggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketTaggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketVersioningCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketVersioningCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketVersioningCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutBucketWebsiteCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutBucketWebsiteCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutBucketWebsiteCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutObjectCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutObjectCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        BucketKeyEnabled: undefined,
        ETag: undefined,
        Expiration: undefined,
        RequestCharged: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSEncryptionContext: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
        VersionId: undefined,
    };
    if (output.headers["x-amz-expiration"] !== undefined) {
        contents.Expiration = output.headers["x-amz-expiration"];
    }
    if (output.headers["etag"] !== undefined) {
        contents.ETag = output.headers["etag"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-context"] !== undefined) {
        contents.SSEKMSEncryptionContext = output.headers["x-amz-server-side-encryption-context"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutObjectCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutObjectAclCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutObjectAclCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutObjectAclCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "NoSuchKey":
        case "com.amazonaws.s3#NoSuchKey":
            response = {
                ...(await deserializeAws_restXmlNoSuchKeyResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlPutObjectLegalHoldCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutObjectLegalHoldCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutObjectLegalHoldCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutObjectLockConfigurationCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutObjectLockConfigurationCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutObjectLockConfigurationCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutObjectRetentionCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutObjectRetentionCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        RequestCharged: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutObjectRetentionCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutObjectTaggingCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutObjectTaggingCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        VersionId: undefined,
    };
    if (output.headers["x-amz-version-id"] !== undefined) {
        contents.VersionId = output.headers["x-amz-version-id"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutObjectTaggingCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlPutPublicAccessBlockCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlPutPublicAccessBlockCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlPutPublicAccessBlockCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlRestoreObjectCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlRestoreObjectCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        RequestCharged: undefined,
        RestoreOutputPath: undefined,
    };
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    if (output.headers["x-amz-restore-output-path"] !== undefined) {
        contents.RestoreOutputPath = output.headers["x-amz-restore-output-path"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlRestoreObjectCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ObjectAlreadyInActiveTierError":
        case "com.amazonaws.s3#ObjectAlreadyInActiveTierError":
            response = {
                ...(await deserializeAws_restXmlObjectAlreadyInActiveTierErrorResponse(parsedOutput, context)),
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
export const deserializeAws_restXmlSelectObjectContentCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlSelectObjectContentCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        Payload: undefined,
    };
    const data = context.eventStreamMarshaller.deserialize(output.body, async (event) => {
        const eventName = Object.keys(event)[0];
        const eventHeaders = Object.entries(event[eventName].headers).reduce((accummulator, curr) => {
            accummulator[curr[0]] = curr[1].value;
            return accummulator;
        }, {});
        const eventMessage = {
            headers: eventHeaders,
            body: event[eventName].body,
        };
        const parsedEvent = {
            [eventName]: eventMessage,
        };
        return await deserializeAws_restXmlSelectObjectContentEventStream_event(parsedEvent, context);
    });
    contents.Payload = data;
    return Promise.resolve(contents);
};
const deserializeAws_restXmlSelectObjectContentCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlUploadPartCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlUploadPartCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        BucketKeyEnabled: undefined,
        ETag: undefined,
        RequestCharged: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
    };
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["etag"] !== undefined) {
        contents.ETag = output.headers["etag"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlUploadPartCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlUploadPartCopyCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlUploadPartCopyCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        BucketKeyEnabled: undefined,
        CopyPartResult: undefined,
        CopySourceVersionId: undefined,
        RequestCharged: undefined,
        SSECustomerAlgorithm: undefined,
        SSECustomerKeyMD5: undefined,
        SSEKMSKeyId: undefined,
        ServerSideEncryption: undefined,
    };
    if (output.headers["x-amz-copy-source-version-id"] !== undefined) {
        contents.CopySourceVersionId = output.headers["x-amz-copy-source-version-id"];
    }
    if (output.headers["x-amz-server-side-encryption"] !== undefined) {
        contents.ServerSideEncryption = output.headers["x-amz-server-side-encryption"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-algorithm"] !== undefined) {
        contents.SSECustomerAlgorithm = output.headers["x-amz-server-side-encryption-customer-algorithm"];
    }
    if (output.headers["x-amz-server-side-encryption-customer-key-md5"] !== undefined) {
        contents.SSECustomerKeyMD5 = output.headers["x-amz-server-side-encryption-customer-key-md5"];
    }
    if (output.headers["x-amz-server-side-encryption-aws-kms-key-id"] !== undefined) {
        contents.SSEKMSKeyId = output.headers["x-amz-server-side-encryption-aws-kms-key-id"];
    }
    if (output.headers["x-amz-server-side-encryption-bucket-key-enabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output.headers["x-amz-server-side-encryption-bucket-key-enabled"]);
    }
    if (output.headers["x-amz-request-charged"] !== undefined) {
        contents.RequestCharged = output.headers["x-amz-request-charged"];
    }
    const data = await parseBody(output.body, context);
    contents.CopyPartResult = deserializeAws_restXmlCopyPartResult(data, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlUploadPartCopyCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
export const deserializeAws_restXmlWriteGetObjectResponseCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restXmlWriteGetObjectResponseCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restXmlWriteGetObjectResponseCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestXmlErrorCode(output, parsedOutput.body);
    switch (errorCode) {
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
const deserializeAws_restXmlSelectObjectContentEventStream_event = async (output, context) => {
    if (output["Records"] !== undefined) {
        return {
            Records: await deserializeAws_restXmlRecordsEvent_event(output["Records"], context),
        };
    }
    if (output["Stats"] !== undefined) {
        return {
            Stats: await deserializeAws_restXmlStatsEvent_event(output["Stats"], context),
        };
    }
    if (output["Progress"] !== undefined) {
        return {
            Progress: await deserializeAws_restXmlProgressEvent_event(output["Progress"], context),
        };
    }
    if (output["Cont"] !== undefined) {
        return {
            Cont: await deserializeAws_restXmlContinuationEvent_event(output["Cont"], context),
        };
    }
    if (output["End"] !== undefined) {
        return {
            End: await deserializeAws_restXmlEndEvent_event(output["End"], context),
        };
    }
    return { $unknown: output };
};
const deserializeAws_restXmlContinuationEvent_event = async (output, context) => {
    let contents = {};
    return contents;
};
const deserializeAws_restXmlEndEvent_event = async (output, context) => {
    let contents = {};
    return contents;
};
const deserializeAws_restXmlProgressEvent_event = async (output, context) => {
    let contents = {};
    contents.Details = await parseBody(output.body, context);
    return contents;
};
const deserializeAws_restXmlRecordsEvent_event = async (output, context) => {
    let contents = {};
    contents.Payload = output.body;
    return contents;
};
const deserializeAws_restXmlStatsEvent_event = async (output, context) => {
    let contents = {};
    contents.Details = await parseBody(output.body, context);
    return contents;
};
const deserializeAws_restXmlBucketAlreadyExistsResponse = async (parsedOutput, context) => {
    const contents = {
        name: "BucketAlreadyExists",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlBucketAlreadyOwnedByYouResponse = async (parsedOutput, context) => {
    const contents = {
        name: "BucketAlreadyOwnedByYou",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlInvalidObjectStateResponse = async (parsedOutput, context) => {
    const contents = {
        name: "InvalidObjectState",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        AccessTier: undefined,
        StorageClass: undefined,
    };
    const data = parsedOutput.body;
    if (data["AccessTier"] !== undefined) {
        contents.AccessTier = __expectString(data["AccessTier"]);
    }
    if (data["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(data["StorageClass"]);
    }
    return contents;
};
const deserializeAws_restXmlNoSuchBucketResponse = async (parsedOutput, context) => {
    const contents = {
        name: "NoSuchBucket",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlNoSuchKeyResponse = async (parsedOutput, context) => {
    const contents = {
        name: "NoSuchKey",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlNoSuchUploadResponse = async (parsedOutput, context) => {
    const contents = {
        name: "NoSuchUpload",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlNotFoundResponse = async (parsedOutput, context) => {
    const contents = {
        name: "NotFound",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlObjectAlreadyInActiveTierErrorResponse = async (parsedOutput, context) => {
    const contents = {
        name: "ObjectAlreadyInActiveTierError",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const deserializeAws_restXmlObjectNotInActiveTierErrorResponse = async (parsedOutput, context) => {
    const contents = {
        name: "ObjectNotInActiveTierError",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
    };
    const data = parsedOutput.body;
    return contents;
};
const serializeAws_restXmlAbortIncompleteMultipartUpload = (input, context) => {
    const bodyNode = new __XmlNode("AbortIncompleteMultipartUpload");
    if (input.DaysAfterInitiation !== undefined && input.DaysAfterInitiation !== null) {
        const node = new __XmlNode("DaysAfterInitiation")
            .addChildNode(new __XmlText(String(input.DaysAfterInitiation)))
            .withName("DaysAfterInitiation");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlAccelerateConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("AccelerateConfiguration");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("BucketAccelerateStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlAccessControlPolicy = (input, context) => {
    const bodyNode = new __XmlNode("AccessControlPolicy");
    if (input.Grants !== undefined && input.Grants !== null) {
        const nodes = serializeAws_restXmlGrants(input.Grants, context);
        const containerNode = new __XmlNode("AccessControlList");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    if (input.Owner !== undefined && input.Owner !== null) {
        const node = serializeAws_restXmlOwner(input.Owner, context).withName("Owner");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlAccessControlTranslation = (input, context) => {
    const bodyNode = new __XmlNode("AccessControlTranslation");
    if (input.Owner !== undefined && input.Owner !== null) {
        const node = new __XmlNode("OwnerOverride").addChildNode(new __XmlText(input.Owner)).withName("Owner");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlAllowedHeaders = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = new __XmlNode("AllowedHeader").addChildNode(new __XmlText(entry));
        return node.withName("member");
    });
};
const serializeAws_restXmlAllowedMethods = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = new __XmlNode("AllowedMethod").addChildNode(new __XmlText(entry));
        return node.withName("member");
    });
};
const serializeAws_restXmlAllowedOrigins = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = new __XmlNode("AllowedOrigin").addChildNode(new __XmlText(entry));
        return node.withName("member");
    });
};
const serializeAws_restXmlAnalyticsAndOperator = (input, context) => {
    const bodyNode = new __XmlNode("AnalyticsAndOperator");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const nodes = serializeAws_restXmlTagSet(input.Tags, context);
        nodes.map((node) => {
            node = node.withName("Tag");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlAnalyticsConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("AnalyticsConfiguration");
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("AnalyticsId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlAnalyticsFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    if (input.StorageClassAnalysis !== undefined && input.StorageClassAnalysis !== null) {
        const node = serializeAws_restXmlStorageClassAnalysis(input.StorageClassAnalysis, context).withName("StorageClassAnalysis");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlAnalyticsExportDestination = (input, context) => {
    const bodyNode = new __XmlNode("AnalyticsExportDestination");
    if (input.S3BucketDestination !== undefined && input.S3BucketDestination !== null) {
        const node = serializeAws_restXmlAnalyticsS3BucketDestination(input.S3BucketDestination, context).withName("S3BucketDestination");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlAnalyticsFilter = (input, context) => {
    const bodyNode = new __XmlNode("AnalyticsFilter");
    AnalyticsFilter.visit(input, {
        Prefix: (value) => {
            const node = new __XmlNode("Prefix").addChildNode(new __XmlText(value)).withName("Prefix");
            bodyNode.addChildNode(node);
        },
        Tag: (value) => {
            const node = serializeAws_restXmlTag(value, context).withName("Tag");
            bodyNode.addChildNode(node);
        },
        And: (value) => {
            const node = serializeAws_restXmlAnalyticsAndOperator(value, context).withName("And");
            bodyNode.addChildNode(node);
        },
        _: (name, value) => {
            if (!(value instanceof __XmlNode || value instanceof __XmlText)) {
                throw new Error("Unable to serialize unknown union members in XML.");
            }
            bodyNode.addChildNode(new __XmlNode(name).addChildNode(value));
        },
    });
    return bodyNode;
};
const serializeAws_restXmlAnalyticsS3BucketDestination = (input, context) => {
    const bodyNode = new __XmlNode("AnalyticsS3BucketDestination");
    if (input.Format !== undefined && input.Format !== null) {
        const node = new __XmlNode("AnalyticsS3ExportFileFormat")
            .addChildNode(new __XmlText(input.Format))
            .withName("Format");
        bodyNode.addChildNode(node);
    }
    if (input.BucketAccountId !== undefined && input.BucketAccountId !== null) {
        const node = new __XmlNode("AccountId")
            .addChildNode(new __XmlText(input.BucketAccountId))
            .withName("BucketAccountId");
        bodyNode.addChildNode(node);
    }
    if (input.Bucket !== undefined && input.Bucket !== null) {
        const node = new __XmlNode("BucketName").addChildNode(new __XmlText(input.Bucket)).withName("Bucket");
        bodyNode.addChildNode(node);
    }
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlBucketLifecycleConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("BucketLifecycleConfiguration");
    if (input.Rules !== undefined && input.Rules !== null) {
        const nodes = serializeAws_restXmlLifecycleRules(input.Rules, context);
        nodes.map((node) => {
            node = node.withName("Rule");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlBucketLoggingStatus = (input, context) => {
    const bodyNode = new __XmlNode("BucketLoggingStatus");
    if (input.LoggingEnabled !== undefined && input.LoggingEnabled !== null) {
        const node = serializeAws_restXmlLoggingEnabled(input.LoggingEnabled, context).withName("LoggingEnabled");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlCompletedMultipartUpload = (input, context) => {
    const bodyNode = new __XmlNode("CompletedMultipartUpload");
    if (input.Parts !== undefined && input.Parts !== null) {
        const nodes = serializeAws_restXmlCompletedPartList(input.Parts, context);
        nodes.map((node) => {
            node = node.withName("Part");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlCompletedPart = (input, context) => {
    const bodyNode = new __XmlNode("CompletedPart");
    if (input.ETag !== undefined && input.ETag !== null) {
        const node = new __XmlNode("ETag").addChildNode(new __XmlText(input.ETag)).withName("ETag");
        bodyNode.addChildNode(node);
    }
    if (input.PartNumber !== undefined && input.PartNumber !== null) {
        const node = new __XmlNode("PartNumber")
            .addChildNode(new __XmlText(String(input.PartNumber)))
            .withName("PartNumber");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlCompletedPartList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlCompletedPart(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlCondition = (input, context) => {
    const bodyNode = new __XmlNode("Condition");
    if (input.HttpErrorCodeReturnedEquals !== undefined && input.HttpErrorCodeReturnedEquals !== null) {
        const node = new __XmlNode("HttpErrorCodeReturnedEquals")
            .addChildNode(new __XmlText(input.HttpErrorCodeReturnedEquals))
            .withName("HttpErrorCodeReturnedEquals");
        bodyNode.addChildNode(node);
    }
    if (input.KeyPrefixEquals !== undefined && input.KeyPrefixEquals !== null) {
        const node = new __XmlNode("KeyPrefixEquals")
            .addChildNode(new __XmlText(input.KeyPrefixEquals))
            .withName("KeyPrefixEquals");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlCORSConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("CORSConfiguration");
    if (input.CORSRules !== undefined && input.CORSRules !== null) {
        const nodes = serializeAws_restXmlCORSRules(input.CORSRules, context);
        nodes.map((node) => {
            node = node.withName("CORSRule");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlCORSRule = (input, context) => {
    const bodyNode = new __XmlNode("CORSRule");
    if (input.ID !== undefined && input.ID !== null) {
        const node = new __XmlNode("ID").addChildNode(new __XmlText(input.ID)).withName("ID");
        bodyNode.addChildNode(node);
    }
    if (input.AllowedHeaders !== undefined && input.AllowedHeaders !== null) {
        const nodes = serializeAws_restXmlAllowedHeaders(input.AllowedHeaders, context);
        nodes.map((node) => {
            node = node.withName("AllowedHeader");
            bodyNode.addChildNode(node);
        });
    }
    if (input.AllowedMethods !== undefined && input.AllowedMethods !== null) {
        const nodes = serializeAws_restXmlAllowedMethods(input.AllowedMethods, context);
        nodes.map((node) => {
            node = node.withName("AllowedMethod");
            bodyNode.addChildNode(node);
        });
    }
    if (input.AllowedOrigins !== undefined && input.AllowedOrigins !== null) {
        const nodes = serializeAws_restXmlAllowedOrigins(input.AllowedOrigins, context);
        nodes.map((node) => {
            node = node.withName("AllowedOrigin");
            bodyNode.addChildNode(node);
        });
    }
    if (input.ExposeHeaders !== undefined && input.ExposeHeaders !== null) {
        const nodes = serializeAws_restXmlExposeHeaders(input.ExposeHeaders, context);
        nodes.map((node) => {
            node = node.withName("ExposeHeader");
            bodyNode.addChildNode(node);
        });
    }
    if (input.MaxAgeSeconds !== undefined && input.MaxAgeSeconds !== null) {
        const node = new __XmlNode("MaxAgeSeconds")
            .addChildNode(new __XmlText(String(input.MaxAgeSeconds)))
            .withName("MaxAgeSeconds");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlCORSRules = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlCORSRule(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlCreateBucketConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("CreateBucketConfiguration");
    if (input.LocationConstraint !== undefined && input.LocationConstraint !== null) {
        const node = new __XmlNode("BucketLocationConstraint")
            .addChildNode(new __XmlText(input.LocationConstraint))
            .withName("LocationConstraint");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlCSVInput = (input, context) => {
    const bodyNode = new __XmlNode("CSVInput");
    if (input.FileHeaderInfo !== undefined && input.FileHeaderInfo !== null) {
        const node = new __XmlNode("FileHeaderInfo")
            .addChildNode(new __XmlText(input.FileHeaderInfo))
            .withName("FileHeaderInfo");
        bodyNode.addChildNode(node);
    }
    if (input.Comments !== undefined && input.Comments !== null) {
        const node = new __XmlNode("Comments").addChildNode(new __XmlText(input.Comments)).withName("Comments");
        bodyNode.addChildNode(node);
    }
    if (input.QuoteEscapeCharacter !== undefined && input.QuoteEscapeCharacter !== null) {
        const node = new __XmlNode("QuoteEscapeCharacter")
            .addChildNode(new __XmlText(input.QuoteEscapeCharacter))
            .withName("QuoteEscapeCharacter");
        bodyNode.addChildNode(node);
    }
    if (input.RecordDelimiter !== undefined && input.RecordDelimiter !== null) {
        const node = new __XmlNode("RecordDelimiter")
            .addChildNode(new __XmlText(input.RecordDelimiter))
            .withName("RecordDelimiter");
        bodyNode.addChildNode(node);
    }
    if (input.FieldDelimiter !== undefined && input.FieldDelimiter !== null) {
        const node = new __XmlNode("FieldDelimiter")
            .addChildNode(new __XmlText(input.FieldDelimiter))
            .withName("FieldDelimiter");
        bodyNode.addChildNode(node);
    }
    if (input.QuoteCharacter !== undefined && input.QuoteCharacter !== null) {
        const node = new __XmlNode("QuoteCharacter")
            .addChildNode(new __XmlText(input.QuoteCharacter))
            .withName("QuoteCharacter");
        bodyNode.addChildNode(node);
    }
    if (input.AllowQuotedRecordDelimiter !== undefined && input.AllowQuotedRecordDelimiter !== null) {
        const node = new __XmlNode("AllowQuotedRecordDelimiter")
            .addChildNode(new __XmlText(String(input.AllowQuotedRecordDelimiter)))
            .withName("AllowQuotedRecordDelimiter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlCSVOutput = (input, context) => {
    const bodyNode = new __XmlNode("CSVOutput");
    if (input.QuoteFields !== undefined && input.QuoteFields !== null) {
        const node = new __XmlNode("QuoteFields").addChildNode(new __XmlText(input.QuoteFields)).withName("QuoteFields");
        bodyNode.addChildNode(node);
    }
    if (input.QuoteEscapeCharacter !== undefined && input.QuoteEscapeCharacter !== null) {
        const node = new __XmlNode("QuoteEscapeCharacter")
            .addChildNode(new __XmlText(input.QuoteEscapeCharacter))
            .withName("QuoteEscapeCharacter");
        bodyNode.addChildNode(node);
    }
    if (input.RecordDelimiter !== undefined && input.RecordDelimiter !== null) {
        const node = new __XmlNode("RecordDelimiter")
            .addChildNode(new __XmlText(input.RecordDelimiter))
            .withName("RecordDelimiter");
        bodyNode.addChildNode(node);
    }
    if (input.FieldDelimiter !== undefined && input.FieldDelimiter !== null) {
        const node = new __XmlNode("FieldDelimiter")
            .addChildNode(new __XmlText(input.FieldDelimiter))
            .withName("FieldDelimiter");
        bodyNode.addChildNode(node);
    }
    if (input.QuoteCharacter !== undefined && input.QuoteCharacter !== null) {
        const node = new __XmlNode("QuoteCharacter")
            .addChildNode(new __XmlText(input.QuoteCharacter))
            .withName("QuoteCharacter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlDefaultRetention = (input, context) => {
    const bodyNode = new __XmlNode("DefaultRetention");
    if (input.Mode !== undefined && input.Mode !== null) {
        const node = new __XmlNode("ObjectLockRetentionMode").addChildNode(new __XmlText(input.Mode)).withName("Mode");
        bodyNode.addChildNode(node);
    }
    if (input.Days !== undefined && input.Days !== null) {
        const node = new __XmlNode("Days").addChildNode(new __XmlText(String(input.Days))).withName("Days");
        bodyNode.addChildNode(node);
    }
    if (input.Years !== undefined && input.Years !== null) {
        const node = new __XmlNode("Years").addChildNode(new __XmlText(String(input.Years))).withName("Years");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlDelete = (input, context) => {
    const bodyNode = new __XmlNode("Delete");
    if (input.Objects !== undefined && input.Objects !== null) {
        const nodes = serializeAws_restXmlObjectIdentifierList(input.Objects, context);
        nodes.map((node) => {
            node = node.withName("Object");
            bodyNode.addChildNode(node);
        });
    }
    if (input.Quiet !== undefined && input.Quiet !== null) {
        const node = new __XmlNode("Quiet").addChildNode(new __XmlText(String(input.Quiet))).withName("Quiet");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlDeleteMarkerReplication = (input, context) => {
    const bodyNode = new __XmlNode("DeleteMarkerReplication");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("DeleteMarkerReplicationStatus")
            .addChildNode(new __XmlText(input.Status))
            .withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlDestination = (input, context) => {
    const bodyNode = new __XmlNode("Destination");
    if (input.Bucket !== undefined && input.Bucket !== null) {
        const node = new __XmlNode("BucketName").addChildNode(new __XmlText(input.Bucket)).withName("Bucket");
        bodyNode.addChildNode(node);
    }
    if (input.Account !== undefined && input.Account !== null) {
        const node = new __XmlNode("AccountId").addChildNode(new __XmlText(input.Account)).withName("Account");
        bodyNode.addChildNode(node);
    }
    if (input.StorageClass !== undefined && input.StorageClass !== null) {
        const node = new __XmlNode("StorageClass").addChildNode(new __XmlText(input.StorageClass)).withName("StorageClass");
        bodyNode.addChildNode(node);
    }
    if (input.AccessControlTranslation !== undefined && input.AccessControlTranslation !== null) {
        const node = serializeAws_restXmlAccessControlTranslation(input.AccessControlTranslation, context).withName("AccessControlTranslation");
        bodyNode.addChildNode(node);
    }
    if (input.EncryptionConfiguration !== undefined && input.EncryptionConfiguration !== null) {
        const node = serializeAws_restXmlEncryptionConfiguration(input.EncryptionConfiguration, context).withName("EncryptionConfiguration");
        bodyNode.addChildNode(node);
    }
    if (input.ReplicationTime !== undefined && input.ReplicationTime !== null) {
        const node = serializeAws_restXmlReplicationTime(input.ReplicationTime, context).withName("ReplicationTime");
        bodyNode.addChildNode(node);
    }
    if (input.Metrics !== undefined && input.Metrics !== null) {
        const node = serializeAws_restXmlMetrics(input.Metrics, context).withName("Metrics");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlEncryption = (input, context) => {
    const bodyNode = new __XmlNode("Encryption");
    if (input.EncryptionType !== undefined && input.EncryptionType !== null) {
        const node = new __XmlNode("ServerSideEncryption")
            .addChildNode(new __XmlText(input.EncryptionType))
            .withName("EncryptionType");
        bodyNode.addChildNode(node);
    }
    if (input.KMSKeyId !== undefined && input.KMSKeyId !== null) {
        const node = new __XmlNode("SSEKMSKeyId").addChildNode(new __XmlText(input.KMSKeyId)).withName("KMSKeyId");
        bodyNode.addChildNode(node);
    }
    if (input.KMSContext !== undefined && input.KMSContext !== null) {
        const node = new __XmlNode("KMSContext").addChildNode(new __XmlText(input.KMSContext)).withName("KMSContext");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlEncryptionConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("EncryptionConfiguration");
    if (input.ReplicaKmsKeyID !== undefined && input.ReplicaKmsKeyID !== null) {
        const node = new __XmlNode("ReplicaKmsKeyID")
            .addChildNode(new __XmlText(input.ReplicaKmsKeyID))
            .withName("ReplicaKmsKeyID");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlErrorDocument = (input, context) => {
    const bodyNode = new __XmlNode("ErrorDocument");
    if (input.Key !== undefined && input.Key !== null) {
        const node = new __XmlNode("ObjectKey").addChildNode(new __XmlText(input.Key)).withName("Key");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlEventList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = new __XmlNode("Event").addChildNode(new __XmlText(entry));
        return node.withName("member");
    });
};
const serializeAws_restXmlExistingObjectReplication = (input, context) => {
    const bodyNode = new __XmlNode("ExistingObjectReplication");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("ExistingObjectReplicationStatus")
            .addChildNode(new __XmlText(input.Status))
            .withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlExposeHeaders = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = new __XmlNode("ExposeHeader").addChildNode(new __XmlText(entry));
        return node.withName("member");
    });
};
const serializeAws_restXmlFilterRule = (input, context) => {
    const bodyNode = new __XmlNode("FilterRule");
    if (input.Name !== undefined && input.Name !== null) {
        const node = new __XmlNode("FilterRuleName").addChildNode(new __XmlText(input.Name)).withName("Name");
        bodyNode.addChildNode(node);
    }
    if (input.Value !== undefined && input.Value !== null) {
        const node = new __XmlNode("FilterRuleValue").addChildNode(new __XmlText(input.Value)).withName("Value");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlFilterRuleList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlFilterRule(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlGlacierJobParameters = (input, context) => {
    const bodyNode = new __XmlNode("GlacierJobParameters");
    if (input.Tier !== undefined && input.Tier !== null) {
        const node = new __XmlNode("Tier").addChildNode(new __XmlText(input.Tier)).withName("Tier");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlGrant = (input, context) => {
    const bodyNode = new __XmlNode("Grant");
    if (input.Grantee !== undefined && input.Grantee !== null) {
        const node = serializeAws_restXmlGrantee(input.Grantee, context).withName("Grantee");
        node.addAttribute("xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance");
        bodyNode.addChildNode(node);
    }
    if (input.Permission !== undefined && input.Permission !== null) {
        const node = new __XmlNode("Permission").addChildNode(new __XmlText(input.Permission)).withName("Permission");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlGrantee = (input, context) => {
    const bodyNode = new __XmlNode("Grantee");
    if (input.DisplayName !== undefined && input.DisplayName !== null) {
        const node = new __XmlNode("DisplayName").addChildNode(new __XmlText(input.DisplayName)).withName("DisplayName");
        bodyNode.addChildNode(node);
    }
    if (input.EmailAddress !== undefined && input.EmailAddress !== null) {
        const node = new __XmlNode("EmailAddress").addChildNode(new __XmlText(input.EmailAddress)).withName("EmailAddress");
        bodyNode.addChildNode(node);
    }
    if (input.ID !== undefined && input.ID !== null) {
        const node = new __XmlNode("ID").addChildNode(new __XmlText(input.ID)).withName("ID");
        bodyNode.addChildNode(node);
    }
    if (input.URI !== undefined && input.URI !== null) {
        const node = new __XmlNode("URI").addChildNode(new __XmlText(input.URI)).withName("URI");
        bodyNode.addChildNode(node);
    }
    if (input.Type !== undefined && input.Type !== null) {
        bodyNode.addAttribute("xsi:type", input.Type);
    }
    return bodyNode;
};
const serializeAws_restXmlGrants = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlGrant(entry, context);
        return node.withName("Grant");
    });
};
const serializeAws_restXmlIndexDocument = (input, context) => {
    const bodyNode = new __XmlNode("IndexDocument");
    if (input.Suffix !== undefined && input.Suffix !== null) {
        const node = new __XmlNode("Suffix").addChildNode(new __XmlText(input.Suffix)).withName("Suffix");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInputSerialization = (input, context) => {
    const bodyNode = new __XmlNode("InputSerialization");
    if (input.CSV !== undefined && input.CSV !== null) {
        const node = serializeAws_restXmlCSVInput(input.CSV, context).withName("CSV");
        bodyNode.addChildNode(node);
    }
    if (input.CompressionType !== undefined && input.CompressionType !== null) {
        const node = new __XmlNode("CompressionType")
            .addChildNode(new __XmlText(input.CompressionType))
            .withName("CompressionType");
        bodyNode.addChildNode(node);
    }
    if (input.JSON !== undefined && input.JSON !== null) {
        const node = serializeAws_restXmlJSONInput(input.JSON, context).withName("JSON");
        bodyNode.addChildNode(node);
    }
    if (input.Parquet !== undefined && input.Parquet !== null) {
        const node = serializeAws_restXmlParquetInput(input.Parquet, context).withName("Parquet");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlIntelligentTieringAndOperator = (input, context) => {
    const bodyNode = new __XmlNode("IntelligentTieringAndOperator");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const nodes = serializeAws_restXmlTagSet(input.Tags, context);
        nodes.map((node) => {
            node = node.withName("Tag");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlIntelligentTieringConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("IntelligentTieringConfiguration");
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("IntelligentTieringId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlIntelligentTieringFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("IntelligentTieringStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    if (input.Tierings !== undefined && input.Tierings !== null) {
        const nodes = serializeAws_restXmlTieringList(input.Tierings, context);
        nodes.map((node) => {
            node = node.withName("Tiering");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlIntelligentTieringFilter = (input, context) => {
    const bodyNode = new __XmlNode("IntelligentTieringFilter");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Tag !== undefined && input.Tag !== null) {
        const node = serializeAws_restXmlTag(input.Tag, context).withName("Tag");
        bodyNode.addChildNode(node);
    }
    if (input.And !== undefined && input.And !== null) {
        const node = serializeAws_restXmlIntelligentTieringAndOperator(input.And, context).withName("And");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInventoryConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("InventoryConfiguration");
    if (input.Destination !== undefined && input.Destination !== null) {
        const node = serializeAws_restXmlInventoryDestination(input.Destination, context).withName("Destination");
        bodyNode.addChildNode(node);
    }
    if (input.IsEnabled !== undefined && input.IsEnabled !== null) {
        const node = new __XmlNode("IsEnabled").addChildNode(new __XmlText(String(input.IsEnabled))).withName("IsEnabled");
        bodyNode.addChildNode(node);
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlInventoryFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("InventoryId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.IncludedObjectVersions !== undefined && input.IncludedObjectVersions !== null) {
        const node = new __XmlNode("InventoryIncludedObjectVersions")
            .addChildNode(new __XmlText(input.IncludedObjectVersions))
            .withName("IncludedObjectVersions");
        bodyNode.addChildNode(node);
    }
    if (input.OptionalFields !== undefined && input.OptionalFields !== null) {
        const nodes = serializeAws_restXmlInventoryOptionalFields(input.OptionalFields, context);
        const containerNode = new __XmlNode("OptionalFields");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    if (input.Schedule !== undefined && input.Schedule !== null) {
        const node = serializeAws_restXmlInventorySchedule(input.Schedule, context).withName("Schedule");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInventoryDestination = (input, context) => {
    const bodyNode = new __XmlNode("InventoryDestination");
    if (input.S3BucketDestination !== undefined && input.S3BucketDestination !== null) {
        const node = serializeAws_restXmlInventoryS3BucketDestination(input.S3BucketDestination, context).withName("S3BucketDestination");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInventoryEncryption = (input, context) => {
    const bodyNode = new __XmlNode("InventoryEncryption");
    if (input.SSES3 !== undefined && input.SSES3 !== null) {
        const node = serializeAws_restXmlSSES3(input.SSES3, context).withName("SSE-S3");
        bodyNode.addChildNode(node);
    }
    if (input.SSEKMS !== undefined && input.SSEKMS !== null) {
        const node = serializeAws_restXmlSSEKMS(input.SSEKMS, context).withName("SSE-KMS");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInventoryFilter = (input, context) => {
    const bodyNode = new __XmlNode("InventoryFilter");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInventoryOptionalFields = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = new __XmlNode("InventoryOptionalField").addChildNode(new __XmlText(entry));
        return node.withName("Field");
    });
};
const serializeAws_restXmlInventoryS3BucketDestination = (input, context) => {
    const bodyNode = new __XmlNode("InventoryS3BucketDestination");
    if (input.AccountId !== undefined && input.AccountId !== null) {
        const node = new __XmlNode("AccountId").addChildNode(new __XmlText(input.AccountId)).withName("AccountId");
        bodyNode.addChildNode(node);
    }
    if (input.Bucket !== undefined && input.Bucket !== null) {
        const node = new __XmlNode("BucketName").addChildNode(new __XmlText(input.Bucket)).withName("Bucket");
        bodyNode.addChildNode(node);
    }
    if (input.Format !== undefined && input.Format !== null) {
        const node = new __XmlNode("InventoryFormat").addChildNode(new __XmlText(input.Format)).withName("Format");
        bodyNode.addChildNode(node);
    }
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Encryption !== undefined && input.Encryption !== null) {
        const node = serializeAws_restXmlInventoryEncryption(input.Encryption, context).withName("Encryption");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlInventorySchedule = (input, context) => {
    const bodyNode = new __XmlNode("InventorySchedule");
    if (input.Frequency !== undefined && input.Frequency !== null) {
        const node = new __XmlNode("InventoryFrequency").addChildNode(new __XmlText(input.Frequency)).withName("Frequency");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlJSONInput = (input, context) => {
    const bodyNode = new __XmlNode("JSONInput");
    if (input.Type !== undefined && input.Type !== null) {
        const node = new __XmlNode("JSONType").addChildNode(new __XmlText(input.Type)).withName("Type");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlJSONOutput = (input, context) => {
    const bodyNode = new __XmlNode("JSONOutput");
    if (input.RecordDelimiter !== undefined && input.RecordDelimiter !== null) {
        const node = new __XmlNode("RecordDelimiter")
            .addChildNode(new __XmlText(input.RecordDelimiter))
            .withName("RecordDelimiter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlLambdaFunctionConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("LambdaFunctionConfiguration");
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("NotificationId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.LambdaFunctionArn !== undefined && input.LambdaFunctionArn !== null) {
        const node = new __XmlNode("LambdaFunctionArn")
            .addChildNode(new __XmlText(input.LambdaFunctionArn))
            .withName("CloudFunction");
        bodyNode.addChildNode(node);
    }
    if (input.Events !== undefined && input.Events !== null) {
        const nodes = serializeAws_restXmlEventList(input.Events, context);
        nodes.map((node) => {
            node = node.withName("Event");
            bodyNode.addChildNode(node);
        });
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlNotificationConfigurationFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlLambdaFunctionConfigurationList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlLambdaFunctionConfiguration(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlLifecycleExpiration = (input, context) => {
    const bodyNode = new __XmlNode("LifecycleExpiration");
    if (input.Date !== undefined && input.Date !== null) {
        const node = new __XmlNode("Date")
            .addChildNode(new __XmlText(input.Date.toISOString().split(".")[0] + "Z"))
            .withName("Date");
        bodyNode.addChildNode(node);
    }
    if (input.Days !== undefined && input.Days !== null) {
        const node = new __XmlNode("Days").addChildNode(new __XmlText(String(input.Days))).withName("Days");
        bodyNode.addChildNode(node);
    }
    if (input.ExpiredObjectDeleteMarker !== undefined && input.ExpiredObjectDeleteMarker !== null) {
        const node = new __XmlNode("ExpiredObjectDeleteMarker")
            .addChildNode(new __XmlText(String(input.ExpiredObjectDeleteMarker)))
            .withName("ExpiredObjectDeleteMarker");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlLifecycleRule = (input, context) => {
    const bodyNode = new __XmlNode("LifecycleRule");
    if (input.Expiration !== undefined && input.Expiration !== null) {
        const node = serializeAws_restXmlLifecycleExpiration(input.Expiration, context).withName("Expiration");
        bodyNode.addChildNode(node);
    }
    if (input.ID !== undefined && input.ID !== null) {
        const node = new __XmlNode("ID").addChildNode(new __XmlText(input.ID)).withName("ID");
        bodyNode.addChildNode(node);
    }
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlLifecycleRuleFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("ExpirationStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    if (input.Transitions !== undefined && input.Transitions !== null) {
        const nodes = serializeAws_restXmlTransitionList(input.Transitions, context);
        nodes.map((node) => {
            node = node.withName("Transition");
            bodyNode.addChildNode(node);
        });
    }
    if (input.NoncurrentVersionTransitions !== undefined && input.NoncurrentVersionTransitions !== null) {
        const nodes = serializeAws_restXmlNoncurrentVersionTransitionList(input.NoncurrentVersionTransitions, context);
        nodes.map((node) => {
            node = node.withName("NoncurrentVersionTransition");
            bodyNode.addChildNode(node);
        });
    }
    if (input.NoncurrentVersionExpiration !== undefined && input.NoncurrentVersionExpiration !== null) {
        const node = serializeAws_restXmlNoncurrentVersionExpiration(input.NoncurrentVersionExpiration, context).withName("NoncurrentVersionExpiration");
        bodyNode.addChildNode(node);
    }
    if (input.AbortIncompleteMultipartUpload !== undefined && input.AbortIncompleteMultipartUpload !== null) {
        const node = serializeAws_restXmlAbortIncompleteMultipartUpload(input.AbortIncompleteMultipartUpload, context).withName("AbortIncompleteMultipartUpload");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlLifecycleRuleAndOperator = (input, context) => {
    const bodyNode = new __XmlNode("LifecycleRuleAndOperator");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const nodes = serializeAws_restXmlTagSet(input.Tags, context);
        nodes.map((node) => {
            node = node.withName("Tag");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlLifecycleRuleFilter = (input, context) => {
    const bodyNode = new __XmlNode("LifecycleRuleFilter");
    LifecycleRuleFilter.visit(input, {
        Prefix: (value) => {
            const node = new __XmlNode("Prefix").addChildNode(new __XmlText(value)).withName("Prefix");
            bodyNode.addChildNode(node);
        },
        Tag: (value) => {
            const node = serializeAws_restXmlTag(value, context).withName("Tag");
            bodyNode.addChildNode(node);
        },
        And: (value) => {
            const node = serializeAws_restXmlLifecycleRuleAndOperator(value, context).withName("And");
            bodyNode.addChildNode(node);
        },
        _: (name, value) => {
            if (!(value instanceof __XmlNode || value instanceof __XmlText)) {
                throw new Error("Unable to serialize unknown union members in XML.");
            }
            bodyNode.addChildNode(new __XmlNode(name).addChildNode(value));
        },
    });
    return bodyNode;
};
const serializeAws_restXmlLifecycleRules = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlLifecycleRule(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlLoggingEnabled = (input, context) => {
    const bodyNode = new __XmlNode("LoggingEnabled");
    if (input.TargetBucket !== undefined && input.TargetBucket !== null) {
        const node = new __XmlNode("TargetBucket").addChildNode(new __XmlText(input.TargetBucket)).withName("TargetBucket");
        bodyNode.addChildNode(node);
    }
    if (input.TargetGrants !== undefined && input.TargetGrants !== null) {
        const nodes = serializeAws_restXmlTargetGrants(input.TargetGrants, context);
        const containerNode = new __XmlNode("TargetGrants");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    if (input.TargetPrefix !== undefined && input.TargetPrefix !== null) {
        const node = new __XmlNode("TargetPrefix").addChildNode(new __XmlText(input.TargetPrefix)).withName("TargetPrefix");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlMetadataEntry = (input, context) => {
    const bodyNode = new __XmlNode("MetadataEntry");
    if (input.Name !== undefined && input.Name !== null) {
        const node = new __XmlNode("MetadataKey").addChildNode(new __XmlText(input.Name)).withName("Name");
        bodyNode.addChildNode(node);
    }
    if (input.Value !== undefined && input.Value !== null) {
        const node = new __XmlNode("MetadataValue").addChildNode(new __XmlText(input.Value)).withName("Value");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlMetrics = (input, context) => {
    const bodyNode = new __XmlNode("Metrics");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("MetricsStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    if (input.EventThreshold !== undefined && input.EventThreshold !== null) {
        const node = serializeAws_restXmlReplicationTimeValue(input.EventThreshold, context).withName("EventThreshold");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlMetricsAndOperator = (input, context) => {
    const bodyNode = new __XmlNode("MetricsAndOperator");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const nodes = serializeAws_restXmlTagSet(input.Tags, context);
        nodes.map((node) => {
            node = node.withName("Tag");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlMetricsConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("MetricsConfiguration");
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("MetricsId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlMetricsFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlMetricsFilter = (input, context) => {
    const bodyNode = new __XmlNode("MetricsFilter");
    MetricsFilter.visit(input, {
        Prefix: (value) => {
            const node = new __XmlNode("Prefix").addChildNode(new __XmlText(value)).withName("Prefix");
            bodyNode.addChildNode(node);
        },
        Tag: (value) => {
            const node = serializeAws_restXmlTag(value, context).withName("Tag");
            bodyNode.addChildNode(node);
        },
        And: (value) => {
            const node = serializeAws_restXmlMetricsAndOperator(value, context).withName("And");
            bodyNode.addChildNode(node);
        },
        _: (name, value) => {
            if (!(value instanceof __XmlNode || value instanceof __XmlText)) {
                throw new Error("Unable to serialize unknown union members in XML.");
            }
            bodyNode.addChildNode(new __XmlNode(name).addChildNode(value));
        },
    });
    return bodyNode;
};
const serializeAws_restXmlNoncurrentVersionExpiration = (input, context) => {
    const bodyNode = new __XmlNode("NoncurrentVersionExpiration");
    if (input.NoncurrentDays !== undefined && input.NoncurrentDays !== null) {
        const node = new __XmlNode("Days")
            .addChildNode(new __XmlText(String(input.NoncurrentDays)))
            .withName("NoncurrentDays");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlNoncurrentVersionTransition = (input, context) => {
    const bodyNode = new __XmlNode("NoncurrentVersionTransition");
    if (input.NoncurrentDays !== undefined && input.NoncurrentDays !== null) {
        const node = new __XmlNode("Days")
            .addChildNode(new __XmlText(String(input.NoncurrentDays)))
            .withName("NoncurrentDays");
        bodyNode.addChildNode(node);
    }
    if (input.StorageClass !== undefined && input.StorageClass !== null) {
        const node = new __XmlNode("TransitionStorageClass")
            .addChildNode(new __XmlText(input.StorageClass))
            .withName("StorageClass");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlNoncurrentVersionTransitionList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlNoncurrentVersionTransition(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlNotificationConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("NotificationConfiguration");
    if (input.TopicConfigurations !== undefined && input.TopicConfigurations !== null) {
        const nodes = serializeAws_restXmlTopicConfigurationList(input.TopicConfigurations, context);
        nodes.map((node) => {
            node = node.withName("TopicConfiguration");
            bodyNode.addChildNode(node);
        });
    }
    if (input.QueueConfigurations !== undefined && input.QueueConfigurations !== null) {
        const nodes = serializeAws_restXmlQueueConfigurationList(input.QueueConfigurations, context);
        nodes.map((node) => {
            node = node.withName("QueueConfiguration");
            bodyNode.addChildNode(node);
        });
    }
    if (input.LambdaFunctionConfigurations !== undefined && input.LambdaFunctionConfigurations !== null) {
        const nodes = serializeAws_restXmlLambdaFunctionConfigurationList(input.LambdaFunctionConfigurations, context);
        nodes.map((node) => {
            node = node.withName("CloudFunctionConfiguration");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlNotificationConfigurationFilter = (input, context) => {
    const bodyNode = new __XmlNode("NotificationConfigurationFilter");
    if (input.Key !== undefined && input.Key !== null) {
        const node = serializeAws_restXmlS3KeyFilter(input.Key, context).withName("S3Key");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlObjectIdentifier = (input, context) => {
    const bodyNode = new __XmlNode("ObjectIdentifier");
    if (input.Key !== undefined && input.Key !== null) {
        const node = new __XmlNode("ObjectKey").addChildNode(new __XmlText(input.Key)).withName("Key");
        bodyNode.addChildNode(node);
    }
    if (input.VersionId !== undefined && input.VersionId !== null) {
        const node = new __XmlNode("ObjectVersionId").addChildNode(new __XmlText(input.VersionId)).withName("VersionId");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlObjectIdentifierList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlObjectIdentifier(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlObjectLockConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("ObjectLockConfiguration");
    if (input.ObjectLockEnabled !== undefined && input.ObjectLockEnabled !== null) {
        const node = new __XmlNode("ObjectLockEnabled")
            .addChildNode(new __XmlText(input.ObjectLockEnabled))
            .withName("ObjectLockEnabled");
        bodyNode.addChildNode(node);
    }
    if (input.Rule !== undefined && input.Rule !== null) {
        const node = serializeAws_restXmlObjectLockRule(input.Rule, context).withName("Rule");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlObjectLockLegalHold = (input, context) => {
    const bodyNode = new __XmlNode("ObjectLockLegalHold");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("ObjectLockLegalHoldStatus")
            .addChildNode(new __XmlText(input.Status))
            .withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlObjectLockRetention = (input, context) => {
    const bodyNode = new __XmlNode("ObjectLockRetention");
    if (input.Mode !== undefined && input.Mode !== null) {
        const node = new __XmlNode("ObjectLockRetentionMode").addChildNode(new __XmlText(input.Mode)).withName("Mode");
        bodyNode.addChildNode(node);
    }
    if (input.RetainUntilDate !== undefined && input.RetainUntilDate !== null) {
        const node = new __XmlNode("Date")
            .addChildNode(new __XmlText(input.RetainUntilDate.toISOString().split(".")[0] + "Z"))
            .withName("RetainUntilDate");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlObjectLockRule = (input, context) => {
    const bodyNode = new __XmlNode("ObjectLockRule");
    if (input.DefaultRetention !== undefined && input.DefaultRetention !== null) {
        const node = serializeAws_restXmlDefaultRetention(input.DefaultRetention, context).withName("DefaultRetention");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlOutputLocation = (input, context) => {
    const bodyNode = new __XmlNode("OutputLocation");
    if (input.S3 !== undefined && input.S3 !== null) {
        const node = serializeAws_restXmlS3Location(input.S3, context).withName("S3");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlOutputSerialization = (input, context) => {
    const bodyNode = new __XmlNode("OutputSerialization");
    if (input.CSV !== undefined && input.CSV !== null) {
        const node = serializeAws_restXmlCSVOutput(input.CSV, context).withName("CSV");
        bodyNode.addChildNode(node);
    }
    if (input.JSON !== undefined && input.JSON !== null) {
        const node = serializeAws_restXmlJSONOutput(input.JSON, context).withName("JSON");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlOwner = (input, context) => {
    const bodyNode = new __XmlNode("Owner");
    if (input.DisplayName !== undefined && input.DisplayName !== null) {
        const node = new __XmlNode("DisplayName").addChildNode(new __XmlText(input.DisplayName)).withName("DisplayName");
        bodyNode.addChildNode(node);
    }
    if (input.ID !== undefined && input.ID !== null) {
        const node = new __XmlNode("ID").addChildNode(new __XmlText(input.ID)).withName("ID");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlOwnershipControls = (input, context) => {
    const bodyNode = new __XmlNode("OwnershipControls");
    if (input.Rules !== undefined && input.Rules !== null) {
        const nodes = serializeAws_restXmlOwnershipControlsRules(input.Rules, context);
        nodes.map((node) => {
            node = node.withName("Rule");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlOwnershipControlsRule = (input, context) => {
    const bodyNode = new __XmlNode("OwnershipControlsRule");
    if (input.ObjectOwnership !== undefined && input.ObjectOwnership !== null) {
        const node = new __XmlNode("ObjectOwnership")
            .addChildNode(new __XmlText(input.ObjectOwnership))
            .withName("ObjectOwnership");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlOwnershipControlsRules = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlOwnershipControlsRule(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlParquetInput = (input, context) => {
    const bodyNode = new __XmlNode("ParquetInput");
    return bodyNode;
};
const serializeAws_restXmlPublicAccessBlockConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("PublicAccessBlockConfiguration");
    if (input.BlockPublicAcls !== undefined && input.BlockPublicAcls !== null) {
        const node = new __XmlNode("Setting")
            .addChildNode(new __XmlText(String(input.BlockPublicAcls)))
            .withName("BlockPublicAcls");
        bodyNode.addChildNode(node);
    }
    if (input.IgnorePublicAcls !== undefined && input.IgnorePublicAcls !== null) {
        const node = new __XmlNode("Setting")
            .addChildNode(new __XmlText(String(input.IgnorePublicAcls)))
            .withName("IgnorePublicAcls");
        bodyNode.addChildNode(node);
    }
    if (input.BlockPublicPolicy !== undefined && input.BlockPublicPolicy !== null) {
        const node = new __XmlNode("Setting")
            .addChildNode(new __XmlText(String(input.BlockPublicPolicy)))
            .withName("BlockPublicPolicy");
        bodyNode.addChildNode(node);
    }
    if (input.RestrictPublicBuckets !== undefined && input.RestrictPublicBuckets !== null) {
        const node = new __XmlNode("Setting")
            .addChildNode(new __XmlText(String(input.RestrictPublicBuckets)))
            .withName("RestrictPublicBuckets");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlQueueConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("QueueConfiguration");
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("NotificationId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.QueueArn !== undefined && input.QueueArn !== null) {
        const node = new __XmlNode("QueueArn").addChildNode(new __XmlText(input.QueueArn)).withName("Queue");
        bodyNode.addChildNode(node);
    }
    if (input.Events !== undefined && input.Events !== null) {
        const nodes = serializeAws_restXmlEventList(input.Events, context);
        nodes.map((node) => {
            node = node.withName("Event");
            bodyNode.addChildNode(node);
        });
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlNotificationConfigurationFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlQueueConfigurationList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlQueueConfiguration(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlRedirect = (input, context) => {
    const bodyNode = new __XmlNode("Redirect");
    if (input.HostName !== undefined && input.HostName !== null) {
        const node = new __XmlNode("HostName").addChildNode(new __XmlText(input.HostName)).withName("HostName");
        bodyNode.addChildNode(node);
    }
    if (input.HttpRedirectCode !== undefined && input.HttpRedirectCode !== null) {
        const node = new __XmlNode("HttpRedirectCode")
            .addChildNode(new __XmlText(input.HttpRedirectCode))
            .withName("HttpRedirectCode");
        bodyNode.addChildNode(node);
    }
    if (input.Protocol !== undefined && input.Protocol !== null) {
        const node = new __XmlNode("Protocol").addChildNode(new __XmlText(input.Protocol)).withName("Protocol");
        bodyNode.addChildNode(node);
    }
    if (input.ReplaceKeyPrefixWith !== undefined && input.ReplaceKeyPrefixWith !== null) {
        const node = new __XmlNode("ReplaceKeyPrefixWith")
            .addChildNode(new __XmlText(input.ReplaceKeyPrefixWith))
            .withName("ReplaceKeyPrefixWith");
        bodyNode.addChildNode(node);
    }
    if (input.ReplaceKeyWith !== undefined && input.ReplaceKeyWith !== null) {
        const node = new __XmlNode("ReplaceKeyWith")
            .addChildNode(new __XmlText(input.ReplaceKeyWith))
            .withName("ReplaceKeyWith");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlRedirectAllRequestsTo = (input, context) => {
    const bodyNode = new __XmlNode("RedirectAllRequestsTo");
    if (input.HostName !== undefined && input.HostName !== null) {
        const node = new __XmlNode("HostName").addChildNode(new __XmlText(input.HostName)).withName("HostName");
        bodyNode.addChildNode(node);
    }
    if (input.Protocol !== undefined && input.Protocol !== null) {
        const node = new __XmlNode("Protocol").addChildNode(new __XmlText(input.Protocol)).withName("Protocol");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlReplicaModifications = (input, context) => {
    const bodyNode = new __XmlNode("ReplicaModifications");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("ReplicaModificationsStatus")
            .addChildNode(new __XmlText(input.Status))
            .withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlReplicationConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("ReplicationConfiguration");
    if (input.Role !== undefined && input.Role !== null) {
        const node = new __XmlNode("Role").addChildNode(new __XmlText(input.Role)).withName("Role");
        bodyNode.addChildNode(node);
    }
    if (input.Rules !== undefined && input.Rules !== null) {
        const nodes = serializeAws_restXmlReplicationRules(input.Rules, context);
        nodes.map((node) => {
            node = node.withName("Rule");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlReplicationRule = (input, context) => {
    const bodyNode = new __XmlNode("ReplicationRule");
    if (input.ID !== undefined && input.ID !== null) {
        const node = new __XmlNode("ID").addChildNode(new __XmlText(input.ID)).withName("ID");
        bodyNode.addChildNode(node);
    }
    if (input.Priority !== undefined && input.Priority !== null) {
        const node = new __XmlNode("Priority").addChildNode(new __XmlText(String(input.Priority))).withName("Priority");
        bodyNode.addChildNode(node);
    }
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlReplicationRuleFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("ReplicationRuleStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    if (input.SourceSelectionCriteria !== undefined && input.SourceSelectionCriteria !== null) {
        const node = serializeAws_restXmlSourceSelectionCriteria(input.SourceSelectionCriteria, context).withName("SourceSelectionCriteria");
        bodyNode.addChildNode(node);
    }
    if (input.ExistingObjectReplication !== undefined && input.ExistingObjectReplication !== null) {
        const node = serializeAws_restXmlExistingObjectReplication(input.ExistingObjectReplication, context).withName("ExistingObjectReplication");
        bodyNode.addChildNode(node);
    }
    if (input.Destination !== undefined && input.Destination !== null) {
        const node = serializeAws_restXmlDestination(input.Destination, context).withName("Destination");
        bodyNode.addChildNode(node);
    }
    if (input.DeleteMarkerReplication !== undefined && input.DeleteMarkerReplication !== null) {
        const node = serializeAws_restXmlDeleteMarkerReplication(input.DeleteMarkerReplication, context).withName("DeleteMarkerReplication");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlReplicationRuleAndOperator = (input, context) => {
    const bodyNode = new __XmlNode("ReplicationRuleAndOperator");
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("Prefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const nodes = serializeAws_restXmlTagSet(input.Tags, context);
        nodes.map((node) => {
            node = node.withName("Tag");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlReplicationRuleFilter = (input, context) => {
    const bodyNode = new __XmlNode("ReplicationRuleFilter");
    ReplicationRuleFilter.visit(input, {
        Prefix: (value) => {
            const node = new __XmlNode("Prefix").addChildNode(new __XmlText(value)).withName("Prefix");
            bodyNode.addChildNode(node);
        },
        Tag: (value) => {
            const node = serializeAws_restXmlTag(value, context).withName("Tag");
            bodyNode.addChildNode(node);
        },
        And: (value) => {
            const node = serializeAws_restXmlReplicationRuleAndOperator(value, context).withName("And");
            bodyNode.addChildNode(node);
        },
        _: (name, value) => {
            if (!(value instanceof __XmlNode || value instanceof __XmlText)) {
                throw new Error("Unable to serialize unknown union members in XML.");
            }
            bodyNode.addChildNode(new __XmlNode(name).addChildNode(value));
        },
    });
    return bodyNode;
};
const serializeAws_restXmlReplicationRules = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlReplicationRule(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlReplicationTime = (input, context) => {
    const bodyNode = new __XmlNode("ReplicationTime");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("ReplicationTimeStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    if (input.Time !== undefined && input.Time !== null) {
        const node = serializeAws_restXmlReplicationTimeValue(input.Time, context).withName("Time");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlReplicationTimeValue = (input, context) => {
    const bodyNode = new __XmlNode("ReplicationTimeValue");
    if (input.Minutes !== undefined && input.Minutes !== null) {
        const node = new __XmlNode("Minutes").addChildNode(new __XmlText(String(input.Minutes))).withName("Minutes");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlRequestPaymentConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("RequestPaymentConfiguration");
    if (input.Payer !== undefined && input.Payer !== null) {
        const node = new __XmlNode("Payer").addChildNode(new __XmlText(input.Payer)).withName("Payer");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlRequestProgress = (input, context) => {
    const bodyNode = new __XmlNode("RequestProgress");
    if (input.Enabled !== undefined && input.Enabled !== null) {
        const node = new __XmlNode("EnableRequestProgress")
            .addChildNode(new __XmlText(String(input.Enabled)))
            .withName("Enabled");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlRestoreRequest = (input, context) => {
    const bodyNode = new __XmlNode("RestoreRequest");
    if (input.Days !== undefined && input.Days !== null) {
        const node = new __XmlNode("Days").addChildNode(new __XmlText(String(input.Days))).withName("Days");
        bodyNode.addChildNode(node);
    }
    if (input.GlacierJobParameters !== undefined && input.GlacierJobParameters !== null) {
        const node = serializeAws_restXmlGlacierJobParameters(input.GlacierJobParameters, context).withName("GlacierJobParameters");
        bodyNode.addChildNode(node);
    }
    if (input.Type !== undefined && input.Type !== null) {
        const node = new __XmlNode("RestoreRequestType").addChildNode(new __XmlText(input.Type)).withName("Type");
        bodyNode.addChildNode(node);
    }
    if (input.Tier !== undefined && input.Tier !== null) {
        const node = new __XmlNode("Tier").addChildNode(new __XmlText(input.Tier)).withName("Tier");
        bodyNode.addChildNode(node);
    }
    if (input.Description !== undefined && input.Description !== null) {
        const node = new __XmlNode("Description").addChildNode(new __XmlText(input.Description)).withName("Description");
        bodyNode.addChildNode(node);
    }
    if (input.SelectParameters !== undefined && input.SelectParameters !== null) {
        const node = serializeAws_restXmlSelectParameters(input.SelectParameters, context).withName("SelectParameters");
        bodyNode.addChildNode(node);
    }
    if (input.OutputLocation !== undefined && input.OutputLocation !== null) {
        const node = serializeAws_restXmlOutputLocation(input.OutputLocation, context).withName("OutputLocation");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlRoutingRule = (input, context) => {
    const bodyNode = new __XmlNode("RoutingRule");
    if (input.Condition !== undefined && input.Condition !== null) {
        const node = serializeAws_restXmlCondition(input.Condition, context).withName("Condition");
        bodyNode.addChildNode(node);
    }
    if (input.Redirect !== undefined && input.Redirect !== null) {
        const node = serializeAws_restXmlRedirect(input.Redirect, context).withName("Redirect");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlRoutingRules = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlRoutingRule(entry, context);
        return node.withName("RoutingRule");
    });
};
const serializeAws_restXmlS3KeyFilter = (input, context) => {
    const bodyNode = new __XmlNode("S3KeyFilter");
    if (input.FilterRules !== undefined && input.FilterRules !== null) {
        const nodes = serializeAws_restXmlFilterRuleList(input.FilterRules, context);
        nodes.map((node) => {
            node = node.withName("FilterRule");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlS3Location = (input, context) => {
    const bodyNode = new __XmlNode("S3Location");
    if (input.BucketName !== undefined && input.BucketName !== null) {
        const node = new __XmlNode("BucketName").addChildNode(new __XmlText(input.BucketName)).withName("BucketName");
        bodyNode.addChildNode(node);
    }
    if (input.Prefix !== undefined && input.Prefix !== null) {
        const node = new __XmlNode("LocationPrefix").addChildNode(new __XmlText(input.Prefix)).withName("Prefix");
        bodyNode.addChildNode(node);
    }
    if (input.Encryption !== undefined && input.Encryption !== null) {
        const node = serializeAws_restXmlEncryption(input.Encryption, context).withName("Encryption");
        bodyNode.addChildNode(node);
    }
    if (input.CannedACL !== undefined && input.CannedACL !== null) {
        const node = new __XmlNode("ObjectCannedACL").addChildNode(new __XmlText(input.CannedACL)).withName("CannedACL");
        bodyNode.addChildNode(node);
    }
    if (input.AccessControlList !== undefined && input.AccessControlList !== null) {
        const nodes = serializeAws_restXmlGrants(input.AccessControlList, context);
        const containerNode = new __XmlNode("AccessControlList");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    if (input.Tagging !== undefined && input.Tagging !== null) {
        const node = serializeAws_restXmlTagging(input.Tagging, context).withName("Tagging");
        bodyNode.addChildNode(node);
    }
    if (input.UserMetadata !== undefined && input.UserMetadata !== null) {
        const nodes = serializeAws_restXmlUserMetadata(input.UserMetadata, context);
        const containerNode = new __XmlNode("UserMetadata");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    if (input.StorageClass !== undefined && input.StorageClass !== null) {
        const node = new __XmlNode("StorageClass").addChildNode(new __XmlText(input.StorageClass)).withName("StorageClass");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlScanRange = (input, context) => {
    const bodyNode = new __XmlNode("ScanRange");
    if (input.Start !== undefined && input.Start !== null) {
        const node = new __XmlNode("Start").addChildNode(new __XmlText(String(input.Start))).withName("Start");
        bodyNode.addChildNode(node);
    }
    if (input.End !== undefined && input.End !== null) {
        const node = new __XmlNode("End").addChildNode(new __XmlText(String(input.End))).withName("End");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlSelectParameters = (input, context) => {
    const bodyNode = new __XmlNode("SelectParameters");
    if (input.InputSerialization !== undefined && input.InputSerialization !== null) {
        const node = serializeAws_restXmlInputSerialization(input.InputSerialization, context).withName("InputSerialization");
        bodyNode.addChildNode(node);
    }
    if (input.ExpressionType !== undefined && input.ExpressionType !== null) {
        const node = new __XmlNode("ExpressionType")
            .addChildNode(new __XmlText(input.ExpressionType))
            .withName("ExpressionType");
        bodyNode.addChildNode(node);
    }
    if (input.Expression !== undefined && input.Expression !== null) {
        const node = new __XmlNode("Expression").addChildNode(new __XmlText(input.Expression)).withName("Expression");
        bodyNode.addChildNode(node);
    }
    if (input.OutputSerialization !== undefined && input.OutputSerialization !== null) {
        const node = serializeAws_restXmlOutputSerialization(input.OutputSerialization, context).withName("OutputSerialization");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlServerSideEncryptionByDefault = (input, context) => {
    const bodyNode = new __XmlNode("ServerSideEncryptionByDefault");
    if (input.SSEAlgorithm !== undefined && input.SSEAlgorithm !== null) {
        const node = new __XmlNode("ServerSideEncryption")
            .addChildNode(new __XmlText(input.SSEAlgorithm))
            .withName("SSEAlgorithm");
        bodyNode.addChildNode(node);
    }
    if (input.KMSMasterKeyID !== undefined && input.KMSMasterKeyID !== null) {
        const node = new __XmlNode("SSEKMSKeyId")
            .addChildNode(new __XmlText(input.KMSMasterKeyID))
            .withName("KMSMasterKeyID");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlServerSideEncryptionConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("ServerSideEncryptionConfiguration");
    if (input.Rules !== undefined && input.Rules !== null) {
        const nodes = serializeAws_restXmlServerSideEncryptionRules(input.Rules, context);
        nodes.map((node) => {
            node = node.withName("Rule");
            bodyNode.addChildNode(node);
        });
    }
    return bodyNode;
};
const serializeAws_restXmlServerSideEncryptionRule = (input, context) => {
    const bodyNode = new __XmlNode("ServerSideEncryptionRule");
    if (input.ApplyServerSideEncryptionByDefault !== undefined && input.ApplyServerSideEncryptionByDefault !== null) {
        const node = serializeAws_restXmlServerSideEncryptionByDefault(input.ApplyServerSideEncryptionByDefault, context).withName("ApplyServerSideEncryptionByDefault");
        bodyNode.addChildNode(node);
    }
    if (input.BucketKeyEnabled !== undefined && input.BucketKeyEnabled !== null) {
        const node = new __XmlNode("BucketKeyEnabled")
            .addChildNode(new __XmlText(String(input.BucketKeyEnabled)))
            .withName("BucketKeyEnabled");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlServerSideEncryptionRules = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlServerSideEncryptionRule(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlSourceSelectionCriteria = (input, context) => {
    const bodyNode = new __XmlNode("SourceSelectionCriteria");
    if (input.SseKmsEncryptedObjects !== undefined && input.SseKmsEncryptedObjects !== null) {
        const node = serializeAws_restXmlSseKmsEncryptedObjects(input.SseKmsEncryptedObjects, context).withName("SseKmsEncryptedObjects");
        bodyNode.addChildNode(node);
    }
    if (input.ReplicaModifications !== undefined && input.ReplicaModifications !== null) {
        const node = serializeAws_restXmlReplicaModifications(input.ReplicaModifications, context).withName("ReplicaModifications");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlSSEKMS = (input, context) => {
    const bodyNode = new __XmlNode("SSE-KMS");
    if (input.KeyId !== undefined && input.KeyId !== null) {
        const node = new __XmlNode("SSEKMSKeyId").addChildNode(new __XmlText(input.KeyId)).withName("KeyId");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlSseKmsEncryptedObjects = (input, context) => {
    const bodyNode = new __XmlNode("SseKmsEncryptedObjects");
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("SseKmsEncryptedObjectsStatus")
            .addChildNode(new __XmlText(input.Status))
            .withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlSSES3 = (input, context) => {
    const bodyNode = new __XmlNode("SSE-S3");
    return bodyNode;
};
const serializeAws_restXmlStorageClassAnalysis = (input, context) => {
    const bodyNode = new __XmlNode("StorageClassAnalysis");
    if (input.DataExport !== undefined && input.DataExport !== null) {
        const node = serializeAws_restXmlStorageClassAnalysisDataExport(input.DataExport, context).withName("DataExport");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlStorageClassAnalysisDataExport = (input, context) => {
    const bodyNode = new __XmlNode("StorageClassAnalysisDataExport");
    if (input.OutputSchemaVersion !== undefined && input.OutputSchemaVersion !== null) {
        const node = new __XmlNode("StorageClassAnalysisSchemaVersion")
            .addChildNode(new __XmlText(input.OutputSchemaVersion))
            .withName("OutputSchemaVersion");
        bodyNode.addChildNode(node);
    }
    if (input.Destination !== undefined && input.Destination !== null) {
        const node = serializeAws_restXmlAnalyticsExportDestination(input.Destination, context).withName("Destination");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlTag = (input, context) => {
    const bodyNode = new __XmlNode("Tag");
    if (input.Key !== undefined && input.Key !== null) {
        const node = new __XmlNode("ObjectKey").addChildNode(new __XmlText(input.Key)).withName("Key");
        bodyNode.addChildNode(node);
    }
    if (input.Value !== undefined && input.Value !== null) {
        const node = new __XmlNode("Value").addChildNode(new __XmlText(input.Value)).withName("Value");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlTagging = (input, context) => {
    const bodyNode = new __XmlNode("Tagging");
    if (input.TagSet !== undefined && input.TagSet !== null) {
        const nodes = serializeAws_restXmlTagSet(input.TagSet, context);
        const containerNode = new __XmlNode("TagSet");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    return bodyNode;
};
const serializeAws_restXmlTagSet = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlTag(entry, context);
        return node.withName("Tag");
    });
};
const serializeAws_restXmlTargetGrant = (input, context) => {
    const bodyNode = new __XmlNode("TargetGrant");
    if (input.Grantee !== undefined && input.Grantee !== null) {
        const node = serializeAws_restXmlGrantee(input.Grantee, context).withName("Grantee");
        node.addAttribute("xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance");
        bodyNode.addChildNode(node);
    }
    if (input.Permission !== undefined && input.Permission !== null) {
        const node = new __XmlNode("BucketLogsPermission")
            .addChildNode(new __XmlText(input.Permission))
            .withName("Permission");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlTargetGrants = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlTargetGrant(entry, context);
        return node.withName("Grant");
    });
};
const serializeAws_restXmlTiering = (input, context) => {
    const bodyNode = new __XmlNode("Tiering");
    if (input.Days !== undefined && input.Days !== null) {
        const node = new __XmlNode("IntelligentTieringDays")
            .addChildNode(new __XmlText(String(input.Days)))
            .withName("Days");
        bodyNode.addChildNode(node);
    }
    if (input.AccessTier !== undefined && input.AccessTier !== null) {
        const node = new __XmlNode("IntelligentTieringAccessTier")
            .addChildNode(new __XmlText(input.AccessTier))
            .withName("AccessTier");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlTieringList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlTiering(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlTopicConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("TopicConfiguration");
    if (input.Id !== undefined && input.Id !== null) {
        const node = new __XmlNode("NotificationId").addChildNode(new __XmlText(input.Id)).withName("Id");
        bodyNode.addChildNode(node);
    }
    if (input.TopicArn !== undefined && input.TopicArn !== null) {
        const node = new __XmlNode("TopicArn").addChildNode(new __XmlText(input.TopicArn)).withName("Topic");
        bodyNode.addChildNode(node);
    }
    if (input.Events !== undefined && input.Events !== null) {
        const nodes = serializeAws_restXmlEventList(input.Events, context);
        nodes.map((node) => {
            node = node.withName("Event");
            bodyNode.addChildNode(node);
        });
    }
    if (input.Filter !== undefined && input.Filter !== null) {
        const node = serializeAws_restXmlNotificationConfigurationFilter(input.Filter, context).withName("Filter");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlTopicConfigurationList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlTopicConfiguration(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlTransition = (input, context) => {
    const bodyNode = new __XmlNode("Transition");
    if (input.Date !== undefined && input.Date !== null) {
        const node = new __XmlNode("Date")
            .addChildNode(new __XmlText(input.Date.toISOString().split(".")[0] + "Z"))
            .withName("Date");
        bodyNode.addChildNode(node);
    }
    if (input.Days !== undefined && input.Days !== null) {
        const node = new __XmlNode("Days").addChildNode(new __XmlText(String(input.Days))).withName("Days");
        bodyNode.addChildNode(node);
    }
    if (input.StorageClass !== undefined && input.StorageClass !== null) {
        const node = new __XmlNode("TransitionStorageClass")
            .addChildNode(new __XmlText(input.StorageClass))
            .withName("StorageClass");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlTransitionList = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlTransition(entry, context);
        return node.withName("member");
    });
};
const serializeAws_restXmlUserMetadata = (input, context) => {
    return input
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        const node = serializeAws_restXmlMetadataEntry(entry, context);
        return node.withName("MetadataEntry");
    });
};
const serializeAws_restXmlVersioningConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("VersioningConfiguration");
    if (input.MFADelete !== undefined && input.MFADelete !== null) {
        const node = new __XmlNode("MFADelete").addChildNode(new __XmlText(input.MFADelete)).withName("MfaDelete");
        bodyNode.addChildNode(node);
    }
    if (input.Status !== undefined && input.Status !== null) {
        const node = new __XmlNode("BucketVersioningStatus").addChildNode(new __XmlText(input.Status)).withName("Status");
        bodyNode.addChildNode(node);
    }
    return bodyNode;
};
const serializeAws_restXmlWebsiteConfiguration = (input, context) => {
    const bodyNode = new __XmlNode("WebsiteConfiguration");
    if (input.ErrorDocument !== undefined && input.ErrorDocument !== null) {
        const node = serializeAws_restXmlErrorDocument(input.ErrorDocument, context).withName("ErrorDocument");
        bodyNode.addChildNode(node);
    }
    if (input.IndexDocument !== undefined && input.IndexDocument !== null) {
        const node = serializeAws_restXmlIndexDocument(input.IndexDocument, context).withName("IndexDocument");
        bodyNode.addChildNode(node);
    }
    if (input.RedirectAllRequestsTo !== undefined && input.RedirectAllRequestsTo !== null) {
        const node = serializeAws_restXmlRedirectAllRequestsTo(input.RedirectAllRequestsTo, context).withName("RedirectAllRequestsTo");
        bodyNode.addChildNode(node);
    }
    if (input.RoutingRules !== undefined && input.RoutingRules !== null) {
        const nodes = serializeAws_restXmlRoutingRules(input.RoutingRules, context);
        const containerNode = new __XmlNode("RoutingRules");
        nodes.map((node) => {
            containerNode.addChildNode(node);
        });
        bodyNode.addChildNode(containerNode);
    }
    return bodyNode;
};
const deserializeAws_restXmlAbortIncompleteMultipartUpload = (output, context) => {
    let contents = {
        DaysAfterInitiation: undefined,
    };
    if (output["DaysAfterInitiation"] !== undefined) {
        contents.DaysAfterInitiation = parseInt(output["DaysAfterInitiation"]);
    }
    return contents;
};
const deserializeAws_restXmlAccessControlTranslation = (output, context) => {
    let contents = {
        Owner: undefined,
    };
    if (output["Owner"] !== undefined) {
        contents.Owner = __expectString(output["Owner"]);
    }
    return contents;
};
const deserializeAws_restXmlAllowedHeaders = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return __expectString(entry);
    });
};
const deserializeAws_restXmlAllowedMethods = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return __expectString(entry);
    });
};
const deserializeAws_restXmlAllowedOrigins = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return __expectString(entry);
    });
};
const deserializeAws_restXmlAnalyticsAndOperator = (output, context) => {
    let contents = {
        Prefix: undefined,
        Tags: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output.Tag === "") {
        contents.Tags = [];
    }
    if (output["Tag"] !== undefined) {
        contents.Tags = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(output["Tag"]), context);
    }
    return contents;
};
const deserializeAws_restXmlAnalyticsConfiguration = (output, context) => {
    let contents = {
        Id: undefined,
        Filter: undefined,
        StorageClassAnalysis: undefined,
    };
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlAnalyticsFilter(output["Filter"], context);
    }
    if (output["StorageClassAnalysis"] !== undefined) {
        contents.StorageClassAnalysis = deserializeAws_restXmlStorageClassAnalysis(output["StorageClassAnalysis"], context);
    }
    return contents;
};
const deserializeAws_restXmlAnalyticsConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlAnalyticsConfiguration(entry, context);
    });
};
const deserializeAws_restXmlAnalyticsExportDestination = (output, context) => {
    let contents = {
        S3BucketDestination: undefined,
    };
    if (output["S3BucketDestination"] !== undefined) {
        contents.S3BucketDestination = deserializeAws_restXmlAnalyticsS3BucketDestination(output["S3BucketDestination"], context);
    }
    return contents;
};
const deserializeAws_restXmlAnalyticsFilter = (output, context) => {
    if (output["Prefix"] !== undefined) {
        return {
            Prefix: __expectString(output["Prefix"]),
        };
    }
    if (output["Tag"] !== undefined) {
        return {
            Tag: deserializeAws_restXmlTag(output["Tag"], context),
        };
    }
    if (output["And"] !== undefined) {
        return {
            And: deserializeAws_restXmlAnalyticsAndOperator(output["And"], context),
        };
    }
    return { $unknown: Object.entries(output)[0] };
};
const deserializeAws_restXmlAnalyticsS3BucketDestination = (output, context) => {
    let contents = {
        Format: undefined,
        BucketAccountId: undefined,
        Bucket: undefined,
        Prefix: undefined,
    };
    if (output["Format"] !== undefined) {
        contents.Format = __expectString(output["Format"]);
    }
    if (output["BucketAccountId"] !== undefined) {
        contents.BucketAccountId = __expectString(output["BucketAccountId"]);
    }
    if (output["Bucket"] !== undefined) {
        contents.Bucket = __expectString(output["Bucket"]);
    }
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    return contents;
};
const deserializeAws_restXmlBucket = (output, context) => {
    let contents = {
        Name: undefined,
        CreationDate: undefined,
    };
    if (output["Name"] !== undefined) {
        contents.Name = __expectString(output["Name"]);
    }
    if (output["CreationDate"] !== undefined) {
        contents.CreationDate = new Date(output["CreationDate"]);
    }
    return contents;
};
const deserializeAws_restXmlBuckets = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlBucket(entry, context);
    });
};
const deserializeAws_restXmlCommonPrefix = (output, context) => {
    let contents = {
        Prefix: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    return contents;
};
const deserializeAws_restXmlCommonPrefixList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlCommonPrefix(entry, context);
    });
};
const deserializeAws_restXmlCondition = (output, context) => {
    let contents = {
        HttpErrorCodeReturnedEquals: undefined,
        KeyPrefixEquals: undefined,
    };
    if (output["HttpErrorCodeReturnedEquals"] !== undefined) {
        contents.HttpErrorCodeReturnedEquals = __expectString(output["HttpErrorCodeReturnedEquals"]);
    }
    if (output["KeyPrefixEquals"] !== undefined) {
        contents.KeyPrefixEquals = __expectString(output["KeyPrefixEquals"]);
    }
    return contents;
};
const deserializeAws_restXmlContinuationEvent = (output, context) => {
    let contents = {};
    return contents;
};
const deserializeAws_restXmlCopyObjectResult = (output, context) => {
    let contents = {
        ETag: undefined,
        LastModified: undefined,
    };
    if (output["ETag"] !== undefined) {
        contents.ETag = __expectString(output["ETag"]);
    }
    if (output["LastModified"] !== undefined) {
        contents.LastModified = new Date(output["LastModified"]);
    }
    return contents;
};
const deserializeAws_restXmlCopyPartResult = (output, context) => {
    let contents = {
        ETag: undefined,
        LastModified: undefined,
    };
    if (output["ETag"] !== undefined) {
        contents.ETag = __expectString(output["ETag"]);
    }
    if (output["LastModified"] !== undefined) {
        contents.LastModified = new Date(output["LastModified"]);
    }
    return contents;
};
const deserializeAws_restXmlCORSRule = (output, context) => {
    let contents = {
        ID: undefined,
        AllowedHeaders: undefined,
        AllowedMethods: undefined,
        AllowedOrigins: undefined,
        ExposeHeaders: undefined,
        MaxAgeSeconds: undefined,
    };
    if (output["ID"] !== undefined) {
        contents.ID = __expectString(output["ID"]);
    }
    if (output.AllowedHeader === "") {
        contents.AllowedHeaders = [];
    }
    if (output["AllowedHeader"] !== undefined) {
        contents.AllowedHeaders = deserializeAws_restXmlAllowedHeaders(__getArrayIfSingleItem(output["AllowedHeader"]), context);
    }
    if (output.AllowedMethod === "") {
        contents.AllowedMethods = [];
    }
    if (output["AllowedMethod"] !== undefined) {
        contents.AllowedMethods = deserializeAws_restXmlAllowedMethods(__getArrayIfSingleItem(output["AllowedMethod"]), context);
    }
    if (output.AllowedOrigin === "") {
        contents.AllowedOrigins = [];
    }
    if (output["AllowedOrigin"] !== undefined) {
        contents.AllowedOrigins = deserializeAws_restXmlAllowedOrigins(__getArrayIfSingleItem(output["AllowedOrigin"]), context);
    }
    if (output.ExposeHeader === "") {
        contents.ExposeHeaders = [];
    }
    if (output["ExposeHeader"] !== undefined) {
        contents.ExposeHeaders = deserializeAws_restXmlExposeHeaders(__getArrayIfSingleItem(output["ExposeHeader"]), context);
    }
    if (output["MaxAgeSeconds"] !== undefined) {
        contents.MaxAgeSeconds = parseInt(output["MaxAgeSeconds"]);
    }
    return contents;
};
const deserializeAws_restXmlCORSRules = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlCORSRule(entry, context);
    });
};
const deserializeAws_restXmlDefaultRetention = (output, context) => {
    let contents = {
        Mode: undefined,
        Days: undefined,
        Years: undefined,
    };
    if (output["Mode"] !== undefined) {
        contents.Mode = __expectString(output["Mode"]);
    }
    if (output["Days"] !== undefined) {
        contents.Days = parseInt(output["Days"]);
    }
    if (output["Years"] !== undefined) {
        contents.Years = parseInt(output["Years"]);
    }
    return contents;
};
const deserializeAws_restXmlDeletedObject = (output, context) => {
    let contents = {
        Key: undefined,
        VersionId: undefined,
        DeleteMarker: undefined,
        DeleteMarkerVersionId: undefined,
    };
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["VersionId"] !== undefined) {
        contents.VersionId = __expectString(output["VersionId"]);
    }
    if (output["DeleteMarker"] !== undefined) {
        contents.DeleteMarker = __parseBoolean(output["DeleteMarker"]);
    }
    if (output["DeleteMarkerVersionId"] !== undefined) {
        contents.DeleteMarkerVersionId = __expectString(output["DeleteMarkerVersionId"]);
    }
    return contents;
};
const deserializeAws_restXmlDeletedObjects = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlDeletedObject(entry, context);
    });
};
const deserializeAws_restXmlDeleteMarkerEntry = (output, context) => {
    let contents = {
        Owner: undefined,
        Key: undefined,
        VersionId: undefined,
        IsLatest: undefined,
        LastModified: undefined,
    };
    if (output["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(output["Owner"], context);
    }
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["VersionId"] !== undefined) {
        contents.VersionId = __expectString(output["VersionId"]);
    }
    if (output["IsLatest"] !== undefined) {
        contents.IsLatest = __parseBoolean(output["IsLatest"]);
    }
    if (output["LastModified"] !== undefined) {
        contents.LastModified = new Date(output["LastModified"]);
    }
    return contents;
};
const deserializeAws_restXmlDeleteMarkerReplication = (output, context) => {
    let contents = {
        Status: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    return contents;
};
const deserializeAws_restXmlDeleteMarkers = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlDeleteMarkerEntry(entry, context);
    });
};
const deserializeAws_restXmlDestination = (output, context) => {
    let contents = {
        Bucket: undefined,
        Account: undefined,
        StorageClass: undefined,
        AccessControlTranslation: undefined,
        EncryptionConfiguration: undefined,
        ReplicationTime: undefined,
        Metrics: undefined,
    };
    if (output["Bucket"] !== undefined) {
        contents.Bucket = __expectString(output["Bucket"]);
    }
    if (output["Account"] !== undefined) {
        contents.Account = __expectString(output["Account"]);
    }
    if (output["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(output["StorageClass"]);
    }
    if (output["AccessControlTranslation"] !== undefined) {
        contents.AccessControlTranslation = deserializeAws_restXmlAccessControlTranslation(output["AccessControlTranslation"], context);
    }
    if (output["EncryptionConfiguration"] !== undefined) {
        contents.EncryptionConfiguration = deserializeAws_restXmlEncryptionConfiguration(output["EncryptionConfiguration"], context);
    }
    if (output["ReplicationTime"] !== undefined) {
        contents.ReplicationTime = deserializeAws_restXmlReplicationTime(output["ReplicationTime"], context);
    }
    if (output["Metrics"] !== undefined) {
        contents.Metrics = deserializeAws_restXmlMetrics(output["Metrics"], context);
    }
    return contents;
};
const deserializeAws_restXmlEncryptionConfiguration = (output, context) => {
    let contents = {
        ReplicaKmsKeyID: undefined,
    };
    if (output["ReplicaKmsKeyID"] !== undefined) {
        contents.ReplicaKmsKeyID = __expectString(output["ReplicaKmsKeyID"]);
    }
    return contents;
};
const deserializeAws_restXmlEndEvent = (output, context) => {
    let contents = {};
    return contents;
};
const deserializeAws_restXml_Error = (output, context) => {
    let contents = {
        Key: undefined,
        VersionId: undefined,
        Code: undefined,
        Message: undefined,
    };
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["VersionId"] !== undefined) {
        contents.VersionId = __expectString(output["VersionId"]);
    }
    if (output["Code"] !== undefined) {
        contents.Code = __expectString(output["Code"]);
    }
    if (output["Message"] !== undefined) {
        contents.Message = __expectString(output["Message"]);
    }
    return contents;
};
const deserializeAws_restXmlErrorDocument = (output, context) => {
    let contents = {
        Key: undefined,
    };
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    return contents;
};
const deserializeAws_restXmlErrors = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXml_Error(entry, context);
    });
};
const deserializeAws_restXmlEventList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return __expectString(entry);
    });
};
const deserializeAws_restXmlExistingObjectReplication = (output, context) => {
    let contents = {
        Status: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    return contents;
};
const deserializeAws_restXmlExposeHeaders = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return __expectString(entry);
    });
};
const deserializeAws_restXmlFilterRule = (output, context) => {
    let contents = {
        Name: undefined,
        Value: undefined,
    };
    if (output["Name"] !== undefined) {
        contents.Name = __expectString(output["Name"]);
    }
    if (output["Value"] !== undefined) {
        contents.Value = __expectString(output["Value"]);
    }
    return contents;
};
const deserializeAws_restXmlFilterRuleList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlFilterRule(entry, context);
    });
};
const deserializeAws_restXmlGrant = (output, context) => {
    let contents = {
        Grantee: undefined,
        Permission: undefined,
    };
    if (output["Grantee"] !== undefined) {
        contents.Grantee = deserializeAws_restXmlGrantee(output["Grantee"], context);
    }
    if (output["Permission"] !== undefined) {
        contents.Permission = __expectString(output["Permission"]);
    }
    return contents;
};
const deserializeAws_restXmlGrantee = (output, context) => {
    let contents = {
        DisplayName: undefined,
        EmailAddress: undefined,
        ID: undefined,
        URI: undefined,
        Type: undefined,
    };
    if (output["DisplayName"] !== undefined) {
        contents.DisplayName = __expectString(output["DisplayName"]);
    }
    if (output["EmailAddress"] !== undefined) {
        contents.EmailAddress = __expectString(output["EmailAddress"]);
    }
    if (output["ID"] !== undefined) {
        contents.ID = __expectString(output["ID"]);
    }
    if (output["URI"] !== undefined) {
        contents.URI = __expectString(output["URI"]);
    }
    if (output["xsi:type"] !== undefined) {
        contents.Type = __expectString(output["xsi:type"]);
    }
    return contents;
};
const deserializeAws_restXmlGrants = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlGrant(entry, context);
    });
};
const deserializeAws_restXmlIndexDocument = (output, context) => {
    let contents = {
        Suffix: undefined,
    };
    if (output["Suffix"] !== undefined) {
        contents.Suffix = __expectString(output["Suffix"]);
    }
    return contents;
};
const deserializeAws_restXmlInitiator = (output, context) => {
    let contents = {
        ID: undefined,
        DisplayName: undefined,
    };
    if (output["ID"] !== undefined) {
        contents.ID = __expectString(output["ID"]);
    }
    if (output["DisplayName"] !== undefined) {
        contents.DisplayName = __expectString(output["DisplayName"]);
    }
    return contents;
};
const deserializeAws_restXmlIntelligentTieringAndOperator = (output, context) => {
    let contents = {
        Prefix: undefined,
        Tags: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output.Tag === "") {
        contents.Tags = [];
    }
    if (output["Tag"] !== undefined) {
        contents.Tags = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(output["Tag"]), context);
    }
    return contents;
};
const deserializeAws_restXmlIntelligentTieringConfiguration = (output, context) => {
    let contents = {
        Id: undefined,
        Filter: undefined,
        Status: undefined,
        Tierings: undefined,
    };
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlIntelligentTieringFilter(output["Filter"], context);
    }
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    if (output.Tiering === "") {
        contents.Tierings = [];
    }
    if (output["Tiering"] !== undefined) {
        contents.Tierings = deserializeAws_restXmlTieringList(__getArrayIfSingleItem(output["Tiering"]), context);
    }
    return contents;
};
const deserializeAws_restXmlIntelligentTieringConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlIntelligentTieringConfiguration(entry, context);
    });
};
const deserializeAws_restXmlIntelligentTieringFilter = (output, context) => {
    let contents = {
        Prefix: undefined,
        Tag: undefined,
        And: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output["Tag"] !== undefined) {
        contents.Tag = deserializeAws_restXmlTag(output["Tag"], context);
    }
    if (output["And"] !== undefined) {
        contents.And = deserializeAws_restXmlIntelligentTieringAndOperator(output["And"], context);
    }
    return contents;
};
const deserializeAws_restXmlInventoryConfiguration = (output, context) => {
    let contents = {
        Destination: undefined,
        IsEnabled: undefined,
        Filter: undefined,
        Id: undefined,
        IncludedObjectVersions: undefined,
        OptionalFields: undefined,
        Schedule: undefined,
    };
    if (output["Destination"] !== undefined) {
        contents.Destination = deserializeAws_restXmlInventoryDestination(output["Destination"], context);
    }
    if (output["IsEnabled"] !== undefined) {
        contents.IsEnabled = __parseBoolean(output["IsEnabled"]);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlInventoryFilter(output["Filter"], context);
    }
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["IncludedObjectVersions"] !== undefined) {
        contents.IncludedObjectVersions = __expectString(output["IncludedObjectVersions"]);
    }
    if (output.OptionalFields === "") {
        contents.OptionalFields = [];
    }
    if (output["OptionalFields"] !== undefined && output["OptionalFields"]["Field"] !== undefined) {
        contents.OptionalFields = deserializeAws_restXmlInventoryOptionalFields(__getArrayIfSingleItem(output["OptionalFields"]["Field"]), context);
    }
    if (output["Schedule"] !== undefined) {
        contents.Schedule = deserializeAws_restXmlInventorySchedule(output["Schedule"], context);
    }
    return contents;
};
const deserializeAws_restXmlInventoryConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlInventoryConfiguration(entry, context);
    });
};
const deserializeAws_restXmlInventoryDestination = (output, context) => {
    let contents = {
        S3BucketDestination: undefined,
    };
    if (output["S3BucketDestination"] !== undefined) {
        contents.S3BucketDestination = deserializeAws_restXmlInventoryS3BucketDestination(output["S3BucketDestination"], context);
    }
    return contents;
};
const deserializeAws_restXmlInventoryEncryption = (output, context) => {
    let contents = {
        SSES3: undefined,
        SSEKMS: undefined,
    };
    if (output["SSE-S3"] !== undefined) {
        contents.SSES3 = deserializeAws_restXmlSSES3(output["SSE-S3"], context);
    }
    if (output["SSE-KMS"] !== undefined) {
        contents.SSEKMS = deserializeAws_restXmlSSEKMS(output["SSE-KMS"], context);
    }
    return contents;
};
const deserializeAws_restXmlInventoryFilter = (output, context) => {
    let contents = {
        Prefix: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    return contents;
};
const deserializeAws_restXmlInventoryOptionalFields = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return __expectString(entry);
    });
};
const deserializeAws_restXmlInventoryS3BucketDestination = (output, context) => {
    let contents = {
        AccountId: undefined,
        Bucket: undefined,
        Format: undefined,
        Prefix: undefined,
        Encryption: undefined,
    };
    if (output["AccountId"] !== undefined) {
        contents.AccountId = __expectString(output["AccountId"]);
    }
    if (output["Bucket"] !== undefined) {
        contents.Bucket = __expectString(output["Bucket"]);
    }
    if (output["Format"] !== undefined) {
        contents.Format = __expectString(output["Format"]);
    }
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output["Encryption"] !== undefined) {
        contents.Encryption = deserializeAws_restXmlInventoryEncryption(output["Encryption"], context);
    }
    return contents;
};
const deserializeAws_restXmlInventorySchedule = (output, context) => {
    let contents = {
        Frequency: undefined,
    };
    if (output["Frequency"] !== undefined) {
        contents.Frequency = __expectString(output["Frequency"]);
    }
    return contents;
};
const deserializeAws_restXmlLambdaFunctionConfiguration = (output, context) => {
    let contents = {
        Id: undefined,
        LambdaFunctionArn: undefined,
        Events: undefined,
        Filter: undefined,
    };
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["CloudFunction"] !== undefined) {
        contents.LambdaFunctionArn = __expectString(output["CloudFunction"]);
    }
    if (output.Event === "") {
        contents.Events = [];
    }
    if (output["Event"] !== undefined) {
        contents.Events = deserializeAws_restXmlEventList(__getArrayIfSingleItem(output["Event"]), context);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlNotificationConfigurationFilter(output["Filter"], context);
    }
    return contents;
};
const deserializeAws_restXmlLambdaFunctionConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlLambdaFunctionConfiguration(entry, context);
    });
};
const deserializeAws_restXmlLifecycleExpiration = (output, context) => {
    let contents = {
        Date: undefined,
        Days: undefined,
        ExpiredObjectDeleteMarker: undefined,
    };
    if (output["Date"] !== undefined) {
        contents.Date = new Date(output["Date"]);
    }
    if (output["Days"] !== undefined) {
        contents.Days = parseInt(output["Days"]);
    }
    if (output["ExpiredObjectDeleteMarker"] !== undefined) {
        contents.ExpiredObjectDeleteMarker = __parseBoolean(output["ExpiredObjectDeleteMarker"]);
    }
    return contents;
};
const deserializeAws_restXmlLifecycleRule = (output, context) => {
    let contents = {
        Expiration: undefined,
        ID: undefined,
        Prefix: undefined,
        Filter: undefined,
        Status: undefined,
        Transitions: undefined,
        NoncurrentVersionTransitions: undefined,
        NoncurrentVersionExpiration: undefined,
        AbortIncompleteMultipartUpload: undefined,
    };
    if (output["Expiration"] !== undefined) {
        contents.Expiration = deserializeAws_restXmlLifecycleExpiration(output["Expiration"], context);
    }
    if (output["ID"] !== undefined) {
        contents.ID = __expectString(output["ID"]);
    }
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlLifecycleRuleFilter(output["Filter"], context);
    }
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    if (output.Transition === "") {
        contents.Transitions = [];
    }
    if (output["Transition"] !== undefined) {
        contents.Transitions = deserializeAws_restXmlTransitionList(__getArrayIfSingleItem(output["Transition"]), context);
    }
    if (output.NoncurrentVersionTransition === "") {
        contents.NoncurrentVersionTransitions = [];
    }
    if (output["NoncurrentVersionTransition"] !== undefined) {
        contents.NoncurrentVersionTransitions = deserializeAws_restXmlNoncurrentVersionTransitionList(__getArrayIfSingleItem(output["NoncurrentVersionTransition"]), context);
    }
    if (output["NoncurrentVersionExpiration"] !== undefined) {
        contents.NoncurrentVersionExpiration = deserializeAws_restXmlNoncurrentVersionExpiration(output["NoncurrentVersionExpiration"], context);
    }
    if (output["AbortIncompleteMultipartUpload"] !== undefined) {
        contents.AbortIncompleteMultipartUpload = deserializeAws_restXmlAbortIncompleteMultipartUpload(output["AbortIncompleteMultipartUpload"], context);
    }
    return contents;
};
const deserializeAws_restXmlLifecycleRuleAndOperator = (output, context) => {
    let contents = {
        Prefix: undefined,
        Tags: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output.Tag === "") {
        contents.Tags = [];
    }
    if (output["Tag"] !== undefined) {
        contents.Tags = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(output["Tag"]), context);
    }
    return contents;
};
const deserializeAws_restXmlLifecycleRuleFilter = (output, context) => {
    if (output["Prefix"] !== undefined) {
        return {
            Prefix: __expectString(output["Prefix"]),
        };
    }
    if (output["Tag"] !== undefined) {
        return {
            Tag: deserializeAws_restXmlTag(output["Tag"], context),
        };
    }
    if (output["And"] !== undefined) {
        return {
            And: deserializeAws_restXmlLifecycleRuleAndOperator(output["And"], context),
        };
    }
    return { $unknown: Object.entries(output)[0] };
};
const deserializeAws_restXmlLifecycleRules = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlLifecycleRule(entry, context);
    });
};
const deserializeAws_restXmlLoggingEnabled = (output, context) => {
    let contents = {
        TargetBucket: undefined,
        TargetGrants: undefined,
        TargetPrefix: undefined,
    };
    if (output["TargetBucket"] !== undefined) {
        contents.TargetBucket = __expectString(output["TargetBucket"]);
    }
    if (output.TargetGrants === "") {
        contents.TargetGrants = [];
    }
    if (output["TargetGrants"] !== undefined && output["TargetGrants"]["Grant"] !== undefined) {
        contents.TargetGrants = deserializeAws_restXmlTargetGrants(__getArrayIfSingleItem(output["TargetGrants"]["Grant"]), context);
    }
    if (output["TargetPrefix"] !== undefined) {
        contents.TargetPrefix = __expectString(output["TargetPrefix"]);
    }
    return contents;
};
const deserializeAws_restXmlMetrics = (output, context) => {
    let contents = {
        Status: undefined,
        EventThreshold: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    if (output["EventThreshold"] !== undefined) {
        contents.EventThreshold = deserializeAws_restXmlReplicationTimeValue(output["EventThreshold"], context);
    }
    return contents;
};
const deserializeAws_restXmlMetricsAndOperator = (output, context) => {
    let contents = {
        Prefix: undefined,
        Tags: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output.Tag === "") {
        contents.Tags = [];
    }
    if (output["Tag"] !== undefined) {
        contents.Tags = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(output["Tag"]), context);
    }
    return contents;
};
const deserializeAws_restXmlMetricsConfiguration = (output, context) => {
    let contents = {
        Id: undefined,
        Filter: undefined,
    };
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlMetricsFilter(output["Filter"], context);
    }
    return contents;
};
const deserializeAws_restXmlMetricsConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlMetricsConfiguration(entry, context);
    });
};
const deserializeAws_restXmlMetricsFilter = (output, context) => {
    if (output["Prefix"] !== undefined) {
        return {
            Prefix: __expectString(output["Prefix"]),
        };
    }
    if (output["Tag"] !== undefined) {
        return {
            Tag: deserializeAws_restXmlTag(output["Tag"], context),
        };
    }
    if (output["And"] !== undefined) {
        return {
            And: deserializeAws_restXmlMetricsAndOperator(output["And"], context),
        };
    }
    return { $unknown: Object.entries(output)[0] };
};
const deserializeAws_restXmlMultipartUpload = (output, context) => {
    let contents = {
        UploadId: undefined,
        Key: undefined,
        Initiated: undefined,
        StorageClass: undefined,
        Owner: undefined,
        Initiator: undefined,
    };
    if (output["UploadId"] !== undefined) {
        contents.UploadId = __expectString(output["UploadId"]);
    }
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["Initiated"] !== undefined) {
        contents.Initiated = new Date(output["Initiated"]);
    }
    if (output["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(output["StorageClass"]);
    }
    if (output["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(output["Owner"], context);
    }
    if (output["Initiator"] !== undefined) {
        contents.Initiator = deserializeAws_restXmlInitiator(output["Initiator"], context);
    }
    return contents;
};
const deserializeAws_restXmlMultipartUploadList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlMultipartUpload(entry, context);
    });
};
const deserializeAws_restXmlNoncurrentVersionExpiration = (output, context) => {
    let contents = {
        NoncurrentDays: undefined,
    };
    if (output["NoncurrentDays"] !== undefined) {
        contents.NoncurrentDays = parseInt(output["NoncurrentDays"]);
    }
    return contents;
};
const deserializeAws_restXmlNoncurrentVersionTransition = (output, context) => {
    let contents = {
        NoncurrentDays: undefined,
        StorageClass: undefined,
    };
    if (output["NoncurrentDays"] !== undefined) {
        contents.NoncurrentDays = parseInt(output["NoncurrentDays"]);
    }
    if (output["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(output["StorageClass"]);
    }
    return contents;
};
const deserializeAws_restXmlNoncurrentVersionTransitionList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlNoncurrentVersionTransition(entry, context);
    });
};
const deserializeAws_restXmlNotificationConfigurationFilter = (output, context) => {
    let contents = {
        Key: undefined,
    };
    if (output["S3Key"] !== undefined) {
        contents.Key = deserializeAws_restXmlS3KeyFilter(output["S3Key"], context);
    }
    return contents;
};
const deserializeAws_restXml_Object = (output, context) => {
    let contents = {
        Key: undefined,
        LastModified: undefined,
        ETag: undefined,
        Size: undefined,
        StorageClass: undefined,
        Owner: undefined,
    };
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["LastModified"] !== undefined) {
        contents.LastModified = new Date(output["LastModified"]);
    }
    if (output["ETag"] !== undefined) {
        contents.ETag = __expectString(output["ETag"]);
    }
    if (output["Size"] !== undefined) {
        contents.Size = parseInt(output["Size"]);
    }
    if (output["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(output["StorageClass"]);
    }
    if (output["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(output["Owner"], context);
    }
    return contents;
};
const deserializeAws_restXmlObjectList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXml_Object(entry, context);
    });
};
const deserializeAws_restXmlObjectLockConfiguration = (output, context) => {
    let contents = {
        ObjectLockEnabled: undefined,
        Rule: undefined,
    };
    if (output["ObjectLockEnabled"] !== undefined) {
        contents.ObjectLockEnabled = __expectString(output["ObjectLockEnabled"]);
    }
    if (output["Rule"] !== undefined) {
        contents.Rule = deserializeAws_restXmlObjectLockRule(output["Rule"], context);
    }
    return contents;
};
const deserializeAws_restXmlObjectLockLegalHold = (output, context) => {
    let contents = {
        Status: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    return contents;
};
const deserializeAws_restXmlObjectLockRetention = (output, context) => {
    let contents = {
        Mode: undefined,
        RetainUntilDate: undefined,
    };
    if (output["Mode"] !== undefined) {
        contents.Mode = __expectString(output["Mode"]);
    }
    if (output["RetainUntilDate"] !== undefined) {
        contents.RetainUntilDate = new Date(output["RetainUntilDate"]);
    }
    return contents;
};
const deserializeAws_restXmlObjectLockRule = (output, context) => {
    let contents = {
        DefaultRetention: undefined,
    };
    if (output["DefaultRetention"] !== undefined) {
        contents.DefaultRetention = deserializeAws_restXmlDefaultRetention(output["DefaultRetention"], context);
    }
    return contents;
};
const deserializeAws_restXmlObjectVersion = (output, context) => {
    let contents = {
        ETag: undefined,
        Size: undefined,
        StorageClass: undefined,
        Key: undefined,
        VersionId: undefined,
        IsLatest: undefined,
        LastModified: undefined,
        Owner: undefined,
    };
    if (output["ETag"] !== undefined) {
        contents.ETag = __expectString(output["ETag"]);
    }
    if (output["Size"] !== undefined) {
        contents.Size = parseInt(output["Size"]);
    }
    if (output["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(output["StorageClass"]);
    }
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["VersionId"] !== undefined) {
        contents.VersionId = __expectString(output["VersionId"]);
    }
    if (output["IsLatest"] !== undefined) {
        contents.IsLatest = __parseBoolean(output["IsLatest"]);
    }
    if (output["LastModified"] !== undefined) {
        contents.LastModified = new Date(output["LastModified"]);
    }
    if (output["Owner"] !== undefined) {
        contents.Owner = deserializeAws_restXmlOwner(output["Owner"], context);
    }
    return contents;
};
const deserializeAws_restXmlObjectVersionList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlObjectVersion(entry, context);
    });
};
const deserializeAws_restXmlOwner = (output, context) => {
    let contents = {
        DisplayName: undefined,
        ID: undefined,
    };
    if (output["DisplayName"] !== undefined) {
        contents.DisplayName = __expectString(output["DisplayName"]);
    }
    if (output["ID"] !== undefined) {
        contents.ID = __expectString(output["ID"]);
    }
    return contents;
};
const deserializeAws_restXmlOwnershipControls = (output, context) => {
    let contents = {
        Rules: undefined,
    };
    if (output.Rule === "") {
        contents.Rules = [];
    }
    if (output["Rule"] !== undefined) {
        contents.Rules = deserializeAws_restXmlOwnershipControlsRules(__getArrayIfSingleItem(output["Rule"]), context);
    }
    return contents;
};
const deserializeAws_restXmlOwnershipControlsRule = (output, context) => {
    let contents = {
        ObjectOwnership: undefined,
    };
    if (output["ObjectOwnership"] !== undefined) {
        contents.ObjectOwnership = __expectString(output["ObjectOwnership"]);
    }
    return contents;
};
const deserializeAws_restXmlOwnershipControlsRules = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlOwnershipControlsRule(entry, context);
    });
};
const deserializeAws_restXmlPart = (output, context) => {
    let contents = {
        PartNumber: undefined,
        LastModified: undefined,
        ETag: undefined,
        Size: undefined,
    };
    if (output["PartNumber"] !== undefined) {
        contents.PartNumber = parseInt(output["PartNumber"]);
    }
    if (output["LastModified"] !== undefined) {
        contents.LastModified = new Date(output["LastModified"]);
    }
    if (output["ETag"] !== undefined) {
        contents.ETag = __expectString(output["ETag"]);
    }
    if (output["Size"] !== undefined) {
        contents.Size = parseInt(output["Size"]);
    }
    return contents;
};
const deserializeAws_restXmlParts = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlPart(entry, context);
    });
};
const deserializeAws_restXmlPolicyStatus = (output, context) => {
    let contents = {
        IsPublic: undefined,
    };
    if (output["IsPublic"] !== undefined) {
        contents.IsPublic = __parseBoolean(output["IsPublic"]);
    }
    return contents;
};
const deserializeAws_restXmlProgress = (output, context) => {
    let contents = {
        BytesScanned: undefined,
        BytesProcessed: undefined,
        BytesReturned: undefined,
    };
    if (output["BytesScanned"] !== undefined) {
        contents.BytesScanned = parseInt(output["BytesScanned"]);
    }
    if (output["BytesProcessed"] !== undefined) {
        contents.BytesProcessed = parseInt(output["BytesProcessed"]);
    }
    if (output["BytesReturned"] !== undefined) {
        contents.BytesReturned = parseInt(output["BytesReturned"]);
    }
    return contents;
};
const deserializeAws_restXmlProgressEvent = (output, context) => {
    let contents = {
        Details: undefined,
    };
    if (output["Details"] !== undefined) {
        contents.Details = deserializeAws_restXmlProgress(output["Details"], context);
    }
    return contents;
};
const deserializeAws_restXmlPublicAccessBlockConfiguration = (output, context) => {
    let contents = {
        BlockPublicAcls: undefined,
        IgnorePublicAcls: undefined,
        BlockPublicPolicy: undefined,
        RestrictPublicBuckets: undefined,
    };
    if (output["BlockPublicAcls"] !== undefined) {
        contents.BlockPublicAcls = __parseBoolean(output["BlockPublicAcls"]);
    }
    if (output["IgnorePublicAcls"] !== undefined) {
        contents.IgnorePublicAcls = __parseBoolean(output["IgnorePublicAcls"]);
    }
    if (output["BlockPublicPolicy"] !== undefined) {
        contents.BlockPublicPolicy = __parseBoolean(output["BlockPublicPolicy"]);
    }
    if (output["RestrictPublicBuckets"] !== undefined) {
        contents.RestrictPublicBuckets = __parseBoolean(output["RestrictPublicBuckets"]);
    }
    return contents;
};
const deserializeAws_restXmlQueueConfiguration = (output, context) => {
    let contents = {
        Id: undefined,
        QueueArn: undefined,
        Events: undefined,
        Filter: undefined,
    };
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["Queue"] !== undefined) {
        contents.QueueArn = __expectString(output["Queue"]);
    }
    if (output.Event === "") {
        contents.Events = [];
    }
    if (output["Event"] !== undefined) {
        contents.Events = deserializeAws_restXmlEventList(__getArrayIfSingleItem(output["Event"]), context);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlNotificationConfigurationFilter(output["Filter"], context);
    }
    return contents;
};
const deserializeAws_restXmlQueueConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlQueueConfiguration(entry, context);
    });
};
const deserializeAws_restXmlRecordsEvent = (output, context) => {
    let contents = {
        Payload: undefined,
    };
    if (output["Payload"] !== undefined) {
        contents.Payload = context.base64Decoder(output["Payload"]);
    }
    return contents;
};
const deserializeAws_restXmlRedirect = (output, context) => {
    let contents = {
        HostName: undefined,
        HttpRedirectCode: undefined,
        Protocol: undefined,
        ReplaceKeyPrefixWith: undefined,
        ReplaceKeyWith: undefined,
    };
    if (output["HostName"] !== undefined) {
        contents.HostName = __expectString(output["HostName"]);
    }
    if (output["HttpRedirectCode"] !== undefined) {
        contents.HttpRedirectCode = __expectString(output["HttpRedirectCode"]);
    }
    if (output["Protocol"] !== undefined) {
        contents.Protocol = __expectString(output["Protocol"]);
    }
    if (output["ReplaceKeyPrefixWith"] !== undefined) {
        contents.ReplaceKeyPrefixWith = __expectString(output["ReplaceKeyPrefixWith"]);
    }
    if (output["ReplaceKeyWith"] !== undefined) {
        contents.ReplaceKeyWith = __expectString(output["ReplaceKeyWith"]);
    }
    return contents;
};
const deserializeAws_restXmlRedirectAllRequestsTo = (output, context) => {
    let contents = {
        HostName: undefined,
        Protocol: undefined,
    };
    if (output["HostName"] !== undefined) {
        contents.HostName = __expectString(output["HostName"]);
    }
    if (output["Protocol"] !== undefined) {
        contents.Protocol = __expectString(output["Protocol"]);
    }
    return contents;
};
const deserializeAws_restXmlReplicaModifications = (output, context) => {
    let contents = {
        Status: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    return contents;
};
const deserializeAws_restXmlReplicationConfiguration = (output, context) => {
    let contents = {
        Role: undefined,
        Rules: undefined,
    };
    if (output["Role"] !== undefined) {
        contents.Role = __expectString(output["Role"]);
    }
    if (output.Rule === "") {
        contents.Rules = [];
    }
    if (output["Rule"] !== undefined) {
        contents.Rules = deserializeAws_restXmlReplicationRules(__getArrayIfSingleItem(output["Rule"]), context);
    }
    return contents;
};
const deserializeAws_restXmlReplicationRule = (output, context) => {
    let contents = {
        ID: undefined,
        Priority: undefined,
        Prefix: undefined,
        Filter: undefined,
        Status: undefined,
        SourceSelectionCriteria: undefined,
        ExistingObjectReplication: undefined,
        Destination: undefined,
        DeleteMarkerReplication: undefined,
    };
    if (output["ID"] !== undefined) {
        contents.ID = __expectString(output["ID"]);
    }
    if (output["Priority"] !== undefined) {
        contents.Priority = parseInt(output["Priority"]);
    }
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlReplicationRuleFilter(output["Filter"], context);
    }
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    if (output["SourceSelectionCriteria"] !== undefined) {
        contents.SourceSelectionCriteria = deserializeAws_restXmlSourceSelectionCriteria(output["SourceSelectionCriteria"], context);
    }
    if (output["ExistingObjectReplication"] !== undefined) {
        contents.ExistingObjectReplication = deserializeAws_restXmlExistingObjectReplication(output["ExistingObjectReplication"], context);
    }
    if (output["Destination"] !== undefined) {
        contents.Destination = deserializeAws_restXmlDestination(output["Destination"], context);
    }
    if (output["DeleteMarkerReplication"] !== undefined) {
        contents.DeleteMarkerReplication = deserializeAws_restXmlDeleteMarkerReplication(output["DeleteMarkerReplication"], context);
    }
    return contents;
};
const deserializeAws_restXmlReplicationRuleAndOperator = (output, context) => {
    let contents = {
        Prefix: undefined,
        Tags: undefined,
    };
    if (output["Prefix"] !== undefined) {
        contents.Prefix = __expectString(output["Prefix"]);
    }
    if (output.Tag === "") {
        contents.Tags = [];
    }
    if (output["Tag"] !== undefined) {
        contents.Tags = deserializeAws_restXmlTagSet(__getArrayIfSingleItem(output["Tag"]), context);
    }
    return contents;
};
const deserializeAws_restXmlReplicationRuleFilter = (output, context) => {
    if (output["Prefix"] !== undefined) {
        return {
            Prefix: __expectString(output["Prefix"]),
        };
    }
    if (output["Tag"] !== undefined) {
        return {
            Tag: deserializeAws_restXmlTag(output["Tag"], context),
        };
    }
    if (output["And"] !== undefined) {
        return {
            And: deserializeAws_restXmlReplicationRuleAndOperator(output["And"], context),
        };
    }
    return { $unknown: Object.entries(output)[0] };
};
const deserializeAws_restXmlReplicationRules = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlReplicationRule(entry, context);
    });
};
const deserializeAws_restXmlReplicationTime = (output, context) => {
    let contents = {
        Status: undefined,
        Time: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    if (output["Time"] !== undefined) {
        contents.Time = deserializeAws_restXmlReplicationTimeValue(output["Time"], context);
    }
    return contents;
};
const deserializeAws_restXmlReplicationTimeValue = (output, context) => {
    let contents = {
        Minutes: undefined,
    };
    if (output["Minutes"] !== undefined) {
        contents.Minutes = parseInt(output["Minutes"]);
    }
    return contents;
};
const deserializeAws_restXmlRoutingRule = (output, context) => {
    let contents = {
        Condition: undefined,
        Redirect: undefined,
    };
    if (output["Condition"] !== undefined) {
        contents.Condition = deserializeAws_restXmlCondition(output["Condition"], context);
    }
    if (output["Redirect"] !== undefined) {
        contents.Redirect = deserializeAws_restXmlRedirect(output["Redirect"], context);
    }
    return contents;
};
const deserializeAws_restXmlRoutingRules = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlRoutingRule(entry, context);
    });
};
const deserializeAws_restXmlS3KeyFilter = (output, context) => {
    let contents = {
        FilterRules: undefined,
    };
    if (output.FilterRule === "") {
        contents.FilterRules = [];
    }
    if (output["FilterRule"] !== undefined) {
        contents.FilterRules = deserializeAws_restXmlFilterRuleList(__getArrayIfSingleItem(output["FilterRule"]), context);
    }
    return contents;
};
const deserializeAws_restXmlSelectObjectContentEventStream = (output, context) => {
    if (output["Records"] !== undefined) {
        return {
            Records: deserializeAws_restXmlRecordsEvent(output["Records"], context),
        };
    }
    if (output["Stats"] !== undefined) {
        return {
            Stats: deserializeAws_restXmlStatsEvent(output["Stats"], context),
        };
    }
    if (output["Progress"] !== undefined) {
        return {
            Progress: deserializeAws_restXmlProgressEvent(output["Progress"], context),
        };
    }
    if (output["Cont"] !== undefined) {
        return {
            Cont: deserializeAws_restXmlContinuationEvent(output["Cont"], context),
        };
    }
    if (output["End"] !== undefined) {
        return {
            End: deserializeAws_restXmlEndEvent(output["End"], context),
        };
    }
    return { $unknown: Object.entries(output)[0] };
};
const deserializeAws_restXmlServerSideEncryptionByDefault = (output, context) => {
    let contents = {
        SSEAlgorithm: undefined,
        KMSMasterKeyID: undefined,
    };
    if (output["SSEAlgorithm"] !== undefined) {
        contents.SSEAlgorithm = __expectString(output["SSEAlgorithm"]);
    }
    if (output["KMSMasterKeyID"] !== undefined) {
        contents.KMSMasterKeyID = __expectString(output["KMSMasterKeyID"]);
    }
    return contents;
};
const deserializeAws_restXmlServerSideEncryptionConfiguration = (output, context) => {
    let contents = {
        Rules: undefined,
    };
    if (output.Rule === "") {
        contents.Rules = [];
    }
    if (output["Rule"] !== undefined) {
        contents.Rules = deserializeAws_restXmlServerSideEncryptionRules(__getArrayIfSingleItem(output["Rule"]), context);
    }
    return contents;
};
const deserializeAws_restXmlServerSideEncryptionRule = (output, context) => {
    let contents = {
        ApplyServerSideEncryptionByDefault: undefined,
        BucketKeyEnabled: undefined,
    };
    if (output["ApplyServerSideEncryptionByDefault"] !== undefined) {
        contents.ApplyServerSideEncryptionByDefault = deserializeAws_restXmlServerSideEncryptionByDefault(output["ApplyServerSideEncryptionByDefault"], context);
    }
    if (output["BucketKeyEnabled"] !== undefined) {
        contents.BucketKeyEnabled = __parseBoolean(output["BucketKeyEnabled"]);
    }
    return contents;
};
const deserializeAws_restXmlServerSideEncryptionRules = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlServerSideEncryptionRule(entry, context);
    });
};
const deserializeAws_restXmlSourceSelectionCriteria = (output, context) => {
    let contents = {
        SseKmsEncryptedObjects: undefined,
        ReplicaModifications: undefined,
    };
    if (output["SseKmsEncryptedObjects"] !== undefined) {
        contents.SseKmsEncryptedObjects = deserializeAws_restXmlSseKmsEncryptedObjects(output["SseKmsEncryptedObjects"], context);
    }
    if (output["ReplicaModifications"] !== undefined) {
        contents.ReplicaModifications = deserializeAws_restXmlReplicaModifications(output["ReplicaModifications"], context);
    }
    return contents;
};
const deserializeAws_restXmlSSEKMS = (output, context) => {
    let contents = {
        KeyId: undefined,
    };
    if (output["KeyId"] !== undefined) {
        contents.KeyId = __expectString(output["KeyId"]);
    }
    return contents;
};
const deserializeAws_restXmlSseKmsEncryptedObjects = (output, context) => {
    let contents = {
        Status: undefined,
    };
    if (output["Status"] !== undefined) {
        contents.Status = __expectString(output["Status"]);
    }
    return contents;
};
const deserializeAws_restXmlSSES3 = (output, context) => {
    let contents = {};
    return contents;
};
const deserializeAws_restXmlStats = (output, context) => {
    let contents = {
        BytesScanned: undefined,
        BytesProcessed: undefined,
        BytesReturned: undefined,
    };
    if (output["BytesScanned"] !== undefined) {
        contents.BytesScanned = parseInt(output["BytesScanned"]);
    }
    if (output["BytesProcessed"] !== undefined) {
        contents.BytesProcessed = parseInt(output["BytesProcessed"]);
    }
    if (output["BytesReturned"] !== undefined) {
        contents.BytesReturned = parseInt(output["BytesReturned"]);
    }
    return contents;
};
const deserializeAws_restXmlStatsEvent = (output, context) => {
    let contents = {
        Details: undefined,
    };
    if (output["Details"] !== undefined) {
        contents.Details = deserializeAws_restXmlStats(output["Details"], context);
    }
    return contents;
};
const deserializeAws_restXmlStorageClassAnalysis = (output, context) => {
    let contents = {
        DataExport: undefined,
    };
    if (output["DataExport"] !== undefined) {
        contents.DataExport = deserializeAws_restXmlStorageClassAnalysisDataExport(output["DataExport"], context);
    }
    return contents;
};
const deserializeAws_restXmlStorageClassAnalysisDataExport = (output, context) => {
    let contents = {
        OutputSchemaVersion: undefined,
        Destination: undefined,
    };
    if (output["OutputSchemaVersion"] !== undefined) {
        contents.OutputSchemaVersion = __expectString(output["OutputSchemaVersion"]);
    }
    if (output["Destination"] !== undefined) {
        contents.Destination = deserializeAws_restXmlAnalyticsExportDestination(output["Destination"], context);
    }
    return contents;
};
const deserializeAws_restXmlTag = (output, context) => {
    let contents = {
        Key: undefined,
        Value: undefined,
    };
    if (output["Key"] !== undefined) {
        contents.Key = __expectString(output["Key"]);
    }
    if (output["Value"] !== undefined) {
        contents.Value = __expectString(output["Value"]);
    }
    return contents;
};
const deserializeAws_restXmlTagSet = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlTag(entry, context);
    });
};
const deserializeAws_restXmlTargetGrant = (output, context) => {
    let contents = {
        Grantee: undefined,
        Permission: undefined,
    };
    if (output["Grantee"] !== undefined) {
        contents.Grantee = deserializeAws_restXmlGrantee(output["Grantee"], context);
    }
    if (output["Permission"] !== undefined) {
        contents.Permission = __expectString(output["Permission"]);
    }
    return contents;
};
const deserializeAws_restXmlTargetGrants = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlTargetGrant(entry, context);
    });
};
const deserializeAws_restXmlTiering = (output, context) => {
    let contents = {
        Days: undefined,
        AccessTier: undefined,
    };
    if (output["Days"] !== undefined) {
        contents.Days = parseInt(output["Days"]);
    }
    if (output["AccessTier"] !== undefined) {
        contents.AccessTier = __expectString(output["AccessTier"]);
    }
    return contents;
};
const deserializeAws_restXmlTieringList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlTiering(entry, context);
    });
};
const deserializeAws_restXmlTopicConfiguration = (output, context) => {
    let contents = {
        Id: undefined,
        TopicArn: undefined,
        Events: undefined,
        Filter: undefined,
    };
    if (output["Id"] !== undefined) {
        contents.Id = __expectString(output["Id"]);
    }
    if (output["Topic"] !== undefined) {
        contents.TopicArn = __expectString(output["Topic"]);
    }
    if (output.Event === "") {
        contents.Events = [];
    }
    if (output["Event"] !== undefined) {
        contents.Events = deserializeAws_restXmlEventList(__getArrayIfSingleItem(output["Event"]), context);
    }
    if (output["Filter"] !== undefined) {
        contents.Filter = deserializeAws_restXmlNotificationConfigurationFilter(output["Filter"], context);
    }
    return contents;
};
const deserializeAws_restXmlTopicConfigurationList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlTopicConfiguration(entry, context);
    });
};
const deserializeAws_restXmlTransition = (output, context) => {
    let contents = {
        Date: undefined,
        Days: undefined,
        StorageClass: undefined,
    };
    if (output["Date"] !== undefined) {
        contents.Date = new Date(output["Date"]);
    }
    if (output["Days"] !== undefined) {
        contents.Days = parseInt(output["Days"]);
    }
    if (output["StorageClass"] !== undefined) {
        contents.StorageClass = __expectString(output["StorageClass"]);
    }
    return contents;
};
const deserializeAws_restXmlTransitionList = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restXmlTransition(entry, context);
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
        const parsedObj = xmlParse(encoded, {
            attributeNamePrefix: "",
            ignoreAttributes: false,
            parseNodeValue: false,
            trimValues: false,
            tagValueProcessor: (val) => (val.trim() === "" ? "" : decodeHTML(val)),
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
const loadRestXmlErrorCode = (output, data) => {
    if (data.Code !== undefined) {
        return data.Code;
    }
    if (output.statusCode == 404) {
        return "NotFound";
    }
    return "";
};