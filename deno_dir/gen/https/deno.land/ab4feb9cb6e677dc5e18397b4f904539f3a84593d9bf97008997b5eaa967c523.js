import { S3Client } from "./S3Client.ts";
import { AbortMultipartUploadCommand, } from "./commands/AbortMultipartUploadCommand.ts";
import { CompleteMultipartUploadCommand, } from "./commands/CompleteMultipartUploadCommand.ts";
import { CopyObjectCommand } from "./commands/CopyObjectCommand.ts";
import { CreateBucketCommand, } from "./commands/CreateBucketCommand.ts";
import { CreateMultipartUploadCommand, } from "./commands/CreateMultipartUploadCommand.ts";
import { DeleteBucketAnalyticsConfigurationCommand, } from "./commands/DeleteBucketAnalyticsConfigurationCommand.ts";
import { DeleteBucketCommand, } from "./commands/DeleteBucketCommand.ts";
import { DeleteBucketCorsCommand, } from "./commands/DeleteBucketCorsCommand.ts";
import { DeleteBucketEncryptionCommand, } from "./commands/DeleteBucketEncryptionCommand.ts";
import { DeleteBucketIntelligentTieringConfigurationCommand, } from "./commands/DeleteBucketIntelligentTieringConfigurationCommand.ts";
import { DeleteBucketInventoryConfigurationCommand, } from "./commands/DeleteBucketInventoryConfigurationCommand.ts";
import { DeleteBucketLifecycleCommand, } from "./commands/DeleteBucketLifecycleCommand.ts";
import { DeleteBucketMetricsConfigurationCommand, } from "./commands/DeleteBucketMetricsConfigurationCommand.ts";
import { DeleteBucketOwnershipControlsCommand, } from "./commands/DeleteBucketOwnershipControlsCommand.ts";
import { DeleteBucketPolicyCommand, } from "./commands/DeleteBucketPolicyCommand.ts";
import { DeleteBucketReplicationCommand, } from "./commands/DeleteBucketReplicationCommand.ts";
import { DeleteBucketTaggingCommand, } from "./commands/DeleteBucketTaggingCommand.ts";
import { DeleteBucketWebsiteCommand, } from "./commands/DeleteBucketWebsiteCommand.ts";
import { DeleteObjectCommand, } from "./commands/DeleteObjectCommand.ts";
import { DeleteObjectTaggingCommand, } from "./commands/DeleteObjectTaggingCommand.ts";
import { DeleteObjectsCommand, } from "./commands/DeleteObjectsCommand.ts";
import { DeletePublicAccessBlockCommand, } from "./commands/DeletePublicAccessBlockCommand.ts";
import { GetBucketAccelerateConfigurationCommand, } from "./commands/GetBucketAccelerateConfigurationCommand.ts";
import { GetBucketAclCommand, } from "./commands/GetBucketAclCommand.ts";
import { GetBucketAnalyticsConfigurationCommand, } from "./commands/GetBucketAnalyticsConfigurationCommand.ts";
import { GetBucketCorsCommand, } from "./commands/GetBucketCorsCommand.ts";
import { GetBucketEncryptionCommand, } from "./commands/GetBucketEncryptionCommand.ts";
import { GetBucketIntelligentTieringConfigurationCommand, } from "./commands/GetBucketIntelligentTieringConfigurationCommand.ts";
import { GetBucketInventoryConfigurationCommand, } from "./commands/GetBucketInventoryConfigurationCommand.ts";
import { GetBucketLifecycleConfigurationCommand, } from "./commands/GetBucketLifecycleConfigurationCommand.ts";
import { GetBucketLocationCommand, } from "./commands/GetBucketLocationCommand.ts";
import { GetBucketLoggingCommand, } from "./commands/GetBucketLoggingCommand.ts";
import { GetBucketMetricsConfigurationCommand, } from "./commands/GetBucketMetricsConfigurationCommand.ts";
import { GetBucketNotificationConfigurationCommand, } from "./commands/GetBucketNotificationConfigurationCommand.ts";
import { GetBucketOwnershipControlsCommand, } from "./commands/GetBucketOwnershipControlsCommand.ts";
import { GetBucketPolicyCommand, } from "./commands/GetBucketPolicyCommand.ts";
import { GetBucketPolicyStatusCommand, } from "./commands/GetBucketPolicyStatusCommand.ts";
import { GetBucketReplicationCommand, } from "./commands/GetBucketReplicationCommand.ts";
import { GetBucketRequestPaymentCommand, } from "./commands/GetBucketRequestPaymentCommand.ts";
import { GetBucketTaggingCommand, } from "./commands/GetBucketTaggingCommand.ts";
import { GetBucketVersioningCommand, } from "./commands/GetBucketVersioningCommand.ts";
import { GetBucketWebsiteCommand, } from "./commands/GetBucketWebsiteCommand.ts";
import { GetObjectAclCommand, } from "./commands/GetObjectAclCommand.ts";
import { GetObjectCommand } from "./commands/GetObjectCommand.ts";
import { GetObjectLegalHoldCommand, } from "./commands/GetObjectLegalHoldCommand.ts";
import { GetObjectLockConfigurationCommand, } from "./commands/GetObjectLockConfigurationCommand.ts";
import { GetObjectRetentionCommand, } from "./commands/GetObjectRetentionCommand.ts";
import { GetObjectTaggingCommand, } from "./commands/GetObjectTaggingCommand.ts";
import { GetObjectTorrentCommand, } from "./commands/GetObjectTorrentCommand.ts";
import { GetPublicAccessBlockCommand, } from "./commands/GetPublicAccessBlockCommand.ts";
import { HeadBucketCommand } from "./commands/HeadBucketCommand.ts";
import { HeadObjectCommand } from "./commands/HeadObjectCommand.ts";
import { ListBucketAnalyticsConfigurationsCommand, } from "./commands/ListBucketAnalyticsConfigurationsCommand.ts";
import { ListBucketIntelligentTieringConfigurationsCommand, } from "./commands/ListBucketIntelligentTieringConfigurationsCommand.ts";
import { ListBucketInventoryConfigurationsCommand, } from "./commands/ListBucketInventoryConfigurationsCommand.ts";
import { ListBucketMetricsConfigurationsCommand, } from "./commands/ListBucketMetricsConfigurationsCommand.ts";
import { ListBucketsCommand } from "./commands/ListBucketsCommand.ts";
import { ListMultipartUploadsCommand, } from "./commands/ListMultipartUploadsCommand.ts";
import { ListObjectVersionsCommand, } from "./commands/ListObjectVersionsCommand.ts";
import { ListObjectsCommand } from "./commands/ListObjectsCommand.ts";
import { ListObjectsV2Command, } from "./commands/ListObjectsV2Command.ts";
import { ListPartsCommand } from "./commands/ListPartsCommand.ts";
import { PutBucketAccelerateConfigurationCommand, } from "./commands/PutBucketAccelerateConfigurationCommand.ts";
import { PutBucketAclCommand, } from "./commands/PutBucketAclCommand.ts";
import { PutBucketAnalyticsConfigurationCommand, } from "./commands/PutBucketAnalyticsConfigurationCommand.ts";
import { PutBucketCorsCommand, } from "./commands/PutBucketCorsCommand.ts";
import { PutBucketEncryptionCommand, } from "./commands/PutBucketEncryptionCommand.ts";
import { PutBucketIntelligentTieringConfigurationCommand, } from "./commands/PutBucketIntelligentTieringConfigurationCommand.ts";
import { PutBucketInventoryConfigurationCommand, } from "./commands/PutBucketInventoryConfigurationCommand.ts";
import { PutBucketLifecycleConfigurationCommand, } from "./commands/PutBucketLifecycleConfigurationCommand.ts";
import { PutBucketLoggingCommand, } from "./commands/PutBucketLoggingCommand.ts";
import { PutBucketMetricsConfigurationCommand, } from "./commands/PutBucketMetricsConfigurationCommand.ts";
import { PutBucketNotificationConfigurationCommand, } from "./commands/PutBucketNotificationConfigurationCommand.ts";
import { PutBucketOwnershipControlsCommand, } from "./commands/PutBucketOwnershipControlsCommand.ts";
import { PutBucketPolicyCommand, } from "./commands/PutBucketPolicyCommand.ts";
import { PutBucketReplicationCommand, } from "./commands/PutBucketReplicationCommand.ts";
import { PutBucketRequestPaymentCommand, } from "./commands/PutBucketRequestPaymentCommand.ts";
import { PutBucketTaggingCommand, } from "./commands/PutBucketTaggingCommand.ts";
import { PutBucketVersioningCommand, } from "./commands/PutBucketVersioningCommand.ts";
import { PutBucketWebsiteCommand, } from "./commands/PutBucketWebsiteCommand.ts";
import { PutObjectAclCommand, } from "./commands/PutObjectAclCommand.ts";
import { PutObjectCommand } from "./commands/PutObjectCommand.ts";
import { PutObjectLegalHoldCommand, } from "./commands/PutObjectLegalHoldCommand.ts";
import { PutObjectLockConfigurationCommand, } from "./commands/PutObjectLockConfigurationCommand.ts";
import { PutObjectRetentionCommand, } from "./commands/PutObjectRetentionCommand.ts";
import { PutObjectTaggingCommand, } from "./commands/PutObjectTaggingCommand.ts";
import { PutPublicAccessBlockCommand, } from "./commands/PutPublicAccessBlockCommand.ts";
import { RestoreObjectCommand, } from "./commands/RestoreObjectCommand.ts";
import { SelectObjectContentCommand, } from "./commands/SelectObjectContentCommand.ts";
import { UploadPartCommand } from "./commands/UploadPartCommand.ts";
import { UploadPartCopyCommand, } from "./commands/UploadPartCopyCommand.ts";
import { WriteGetObjectResponseCommand, } from "./commands/WriteGetObjectResponseCommand.ts";
export class S3 extends S3Client {
    abortMultipartUpload(args, optionsOrCb, cb) {
        const command = new AbortMultipartUploadCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    completeMultipartUpload(args, optionsOrCb, cb) {
        const command = new CompleteMultipartUploadCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    copyObject(args, optionsOrCb, cb) {
        const command = new CopyObjectCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    createBucket(args, optionsOrCb, cb) {
        const command = new CreateBucketCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    createMultipartUpload(args, optionsOrCb, cb) {
        const command = new CreateMultipartUploadCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucket(args, optionsOrCb, cb) {
        const command = new DeleteBucketCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketAnalyticsConfiguration(args, optionsOrCb, cb) {
        const command = new DeleteBucketAnalyticsConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketCors(args, optionsOrCb, cb) {
        const command = new DeleteBucketCorsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketEncryption(args, optionsOrCb, cb) {
        const command = new DeleteBucketEncryptionCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketIntelligentTieringConfiguration(args, optionsOrCb, cb) {
        const command = new DeleteBucketIntelligentTieringConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketInventoryConfiguration(args, optionsOrCb, cb) {
        const command = new DeleteBucketInventoryConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketLifecycle(args, optionsOrCb, cb) {
        const command = new DeleteBucketLifecycleCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketMetricsConfiguration(args, optionsOrCb, cb) {
        const command = new DeleteBucketMetricsConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketOwnershipControls(args, optionsOrCb, cb) {
        const command = new DeleteBucketOwnershipControlsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketPolicy(args, optionsOrCb, cb) {
        const command = new DeleteBucketPolicyCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketReplication(args, optionsOrCb, cb) {
        const command = new DeleteBucketReplicationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketTagging(args, optionsOrCb, cb) {
        const command = new DeleteBucketTaggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteBucketWebsite(args, optionsOrCb, cb) {
        const command = new DeleteBucketWebsiteCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteObject(args, optionsOrCb, cb) {
        const command = new DeleteObjectCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteObjects(args, optionsOrCb, cb) {
        const command = new DeleteObjectsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deleteObjectTagging(args, optionsOrCb, cb) {
        const command = new DeleteObjectTaggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    deletePublicAccessBlock(args, optionsOrCb, cb) {
        const command = new DeletePublicAccessBlockCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketAccelerateConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketAccelerateConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketAcl(args, optionsOrCb, cb) {
        const command = new GetBucketAclCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketAnalyticsConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketAnalyticsConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketCors(args, optionsOrCb, cb) {
        const command = new GetBucketCorsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketEncryption(args, optionsOrCb, cb) {
        const command = new GetBucketEncryptionCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketIntelligentTieringConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketIntelligentTieringConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketInventoryConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketInventoryConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketLifecycleConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketLifecycleConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketLocation(args, optionsOrCb, cb) {
        const command = new GetBucketLocationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketLogging(args, optionsOrCb, cb) {
        const command = new GetBucketLoggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketMetricsConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketMetricsConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketNotificationConfiguration(args, optionsOrCb, cb) {
        const command = new GetBucketNotificationConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketOwnershipControls(args, optionsOrCb, cb) {
        const command = new GetBucketOwnershipControlsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketPolicy(args, optionsOrCb, cb) {
        const command = new GetBucketPolicyCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketPolicyStatus(args, optionsOrCb, cb) {
        const command = new GetBucketPolicyStatusCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketReplication(args, optionsOrCb, cb) {
        const command = new GetBucketReplicationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketRequestPayment(args, optionsOrCb, cb) {
        const command = new GetBucketRequestPaymentCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketTagging(args, optionsOrCb, cb) {
        const command = new GetBucketTaggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketVersioning(args, optionsOrCb, cb) {
        const command = new GetBucketVersioningCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getBucketWebsite(args, optionsOrCb, cb) {
        const command = new GetBucketWebsiteCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObject(args, optionsOrCb, cb) {
        const command = new GetObjectCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObjectAcl(args, optionsOrCb, cb) {
        const command = new GetObjectAclCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObjectLegalHold(args, optionsOrCb, cb) {
        const command = new GetObjectLegalHoldCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObjectLockConfiguration(args, optionsOrCb, cb) {
        const command = new GetObjectLockConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObjectRetention(args, optionsOrCb, cb) {
        const command = new GetObjectRetentionCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObjectTagging(args, optionsOrCb, cb) {
        const command = new GetObjectTaggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getObjectTorrent(args, optionsOrCb, cb) {
        const command = new GetObjectTorrentCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getPublicAccessBlock(args, optionsOrCb, cb) {
        const command = new GetPublicAccessBlockCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    headBucket(args, optionsOrCb, cb) {
        const command = new HeadBucketCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    headObject(args, optionsOrCb, cb) {
        const command = new HeadObjectCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listBucketAnalyticsConfigurations(args, optionsOrCb, cb) {
        const command = new ListBucketAnalyticsConfigurationsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listBucketIntelligentTieringConfigurations(args, optionsOrCb, cb) {
        const command = new ListBucketIntelligentTieringConfigurationsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listBucketInventoryConfigurations(args, optionsOrCb, cb) {
        const command = new ListBucketInventoryConfigurationsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listBucketMetricsConfigurations(args, optionsOrCb, cb) {
        const command = new ListBucketMetricsConfigurationsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listBuckets(args, optionsOrCb, cb) {
        const command = new ListBucketsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listMultipartUploads(args, optionsOrCb, cb) {
        const command = new ListMultipartUploadsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listObjects(args, optionsOrCb, cb) {
        const command = new ListObjectsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listObjectsV2(args, optionsOrCb, cb) {
        const command = new ListObjectsV2Command(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listObjectVersions(args, optionsOrCb, cb) {
        const command = new ListObjectVersionsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    listParts(args, optionsOrCb, cb) {
        const command = new ListPartsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketAccelerateConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketAccelerateConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketAcl(args, optionsOrCb, cb) {
        const command = new PutBucketAclCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketAnalyticsConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketAnalyticsConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketCors(args, optionsOrCb, cb) {
        const command = new PutBucketCorsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketEncryption(args, optionsOrCb, cb) {
        const command = new PutBucketEncryptionCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketIntelligentTieringConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketIntelligentTieringConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketInventoryConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketInventoryConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketLifecycleConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketLifecycleConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketLogging(args, optionsOrCb, cb) {
        const command = new PutBucketLoggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketMetricsConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketMetricsConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketNotificationConfiguration(args, optionsOrCb, cb) {
        const command = new PutBucketNotificationConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketOwnershipControls(args, optionsOrCb, cb) {
        const command = new PutBucketOwnershipControlsCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketPolicy(args, optionsOrCb, cb) {
        const command = new PutBucketPolicyCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketReplication(args, optionsOrCb, cb) {
        const command = new PutBucketReplicationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketRequestPayment(args, optionsOrCb, cb) {
        const command = new PutBucketRequestPaymentCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketTagging(args, optionsOrCb, cb) {
        const command = new PutBucketTaggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketVersioning(args, optionsOrCb, cb) {
        const command = new PutBucketVersioningCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putBucketWebsite(args, optionsOrCb, cb) {
        const command = new PutBucketWebsiteCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putObject(args, optionsOrCb, cb) {
        const command = new PutObjectCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putObjectAcl(args, optionsOrCb, cb) {
        const command = new PutObjectAclCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putObjectLegalHold(args, optionsOrCb, cb) {
        const command = new PutObjectLegalHoldCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putObjectLockConfiguration(args, optionsOrCb, cb) {
        const command = new PutObjectLockConfigurationCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putObjectRetention(args, optionsOrCb, cb) {
        const command = new PutObjectRetentionCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putObjectTagging(args, optionsOrCb, cb) {
        const command = new PutObjectTaggingCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    putPublicAccessBlock(args, optionsOrCb, cb) {
        const command = new PutPublicAccessBlockCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    restoreObject(args, optionsOrCb, cb) {
        const command = new RestoreObjectCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    selectObjectContent(args, optionsOrCb, cb) {
        const command = new SelectObjectContentCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    uploadPart(args, optionsOrCb, cb) {
        const command = new UploadPartCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    uploadPartCopy(args, optionsOrCb, cb) {
        const command = new UploadPartCopyCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    writeGetObjectResponse(args, optionsOrCb, cb) {
        const command = new WriteGetObjectResponseCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUzMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJTMy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsUUFBUSxFQUFFLE1BQU0sZUFBZSxDQUFDO0FBQ3pDLE9BQU8sRUFDTCwyQkFBMkIsR0FHNUIsTUFBTSwyQ0FBMkMsQ0FBQztBQUNuRCxPQUFPLEVBQ0wsOEJBQThCLEdBRy9CLE1BQU0sOENBQThDLENBQUM7QUFDdEQsT0FBTyxFQUFFLGlCQUFpQixFQUFtRCxNQUFNLGlDQUFpQyxDQUFDO0FBQ3JILE9BQU8sRUFDTCxtQkFBbUIsR0FHcEIsTUFBTSxtQ0FBbUMsQ0FBQztBQUMzQyxPQUFPLEVBQ0wsNEJBQTRCLEdBRzdCLE1BQU0sNENBQTRDLENBQUM7QUFDcEQsT0FBTyxFQUNMLHlDQUF5QyxHQUcxQyxNQUFNLHlEQUF5RCxDQUFDO0FBQ2pFLE9BQU8sRUFDTCxtQkFBbUIsR0FHcEIsTUFBTSxtQ0FBbUMsQ0FBQztBQUMzQyxPQUFPLEVBQ0wsdUJBQXVCLEdBR3hCLE1BQU0sdUNBQXVDLENBQUM7QUFDL0MsT0FBTyxFQUNMLDZCQUE2QixHQUc5QixNQUFNLDZDQUE2QyxDQUFDO0FBQ3JELE9BQU8sRUFDTCxrREFBa0QsR0FHbkQsTUFBTSxrRUFBa0UsQ0FBQztBQUMxRSxPQUFPLEVBQ0wseUNBQXlDLEdBRzFDLE1BQU0seURBQXlELENBQUM7QUFDakUsT0FBTyxFQUNMLDRCQUE0QixHQUc3QixNQUFNLDRDQUE0QyxDQUFDO0FBQ3BELE9BQU8sRUFDTCx1Q0FBdUMsR0FHeEMsTUFBTSx1REFBdUQsQ0FBQztBQUMvRCxPQUFPLEVBQ0wsb0NBQW9DLEdBR3JDLE1BQU0sb0RBQW9ELENBQUM7QUFDNUQsT0FBTyxFQUNMLHlCQUF5QixHQUcxQixNQUFNLHlDQUF5QyxDQUFDO0FBQ2pELE9BQU8sRUFDTCw4QkFBOEIsR0FHL0IsTUFBTSw4Q0FBOEMsQ0FBQztBQUN0RCxPQUFPLEVBQ0wsMEJBQTBCLEdBRzNCLE1BQU0sMENB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