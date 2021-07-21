import { SENSITIVE_STRING } from "../../smithy-client/mod.ts";
export var Encryption;
(function (Encryption) {
    Encryption.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.KMSKeyId && { KMSKeyId: SENSITIVE_STRING }),
    });
})(Encryption || (Encryption = {}));
export var MetadataEntry;
(function (MetadataEntry) {
    MetadataEntry.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(MetadataEntry || (MetadataEntry = {}));
export var S3Location;
(function (S3Location) {
    S3Location.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.Encryption && { Encryption: Encryption.filterSensitiveLog(obj.Encryption) }),
    });
})(S3Location || (S3Location = {}));
export var OutputLocation;
(function (OutputLocation) {
    OutputLocation.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.S3 && { S3: S3Location.filterSensitiveLog(obj.S3) }),
    });
})(OutputLocation || (OutputLocation = {}));
export var FileHeaderInfo;
(function (FileHeaderInfo) {
    FileHeaderInfo["IGNORE"] = "IGNORE";
    FileHeaderInfo["NONE"] = "NONE";
    FileHeaderInfo["USE"] = "USE";
})(FileHeaderInfo || (FileHeaderInfo = {}));
export var CSVInput;
(function (CSVInput) {
    CSVInput.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(CSVInput || (CSVInput = {}));
export var JSONType;
(function (JSONType) {
    JSONType["DOCUMENT"] = "DOCUMENT";
    JSONType["LINES"] = "LINES";
})(JSONType || (JSONType = {}));
export var JSONInput;
(function (JSONInput) {
    JSONInput.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(JSONInput || (JSONInput = {}));
export var ParquetInput;
(function (ParquetInput) {
    ParquetInput.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ParquetInput || (ParquetInput = {}));
export var InputSerialization;
(function (InputSerialization) {
    InputSerialization.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(InputSerialization || (InputSerialization = {}));
export var QuoteFields;
(function (QuoteFields) {
    QuoteFields["ALWAYS"] = "ALWAYS";
    QuoteFields["ASNEEDED"] = "ASNEEDED";
})(QuoteFields || (QuoteFields = {}));
export var CSVOutput;
(function (CSVOutput) {
    CSVOutput.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(CSVOutput || (CSVOutput = {}));
export var JSONOutput;
(function (JSONOutput) {
    JSONOutput.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(JSONOutput || (JSONOutput = {}));
export var OutputSerialization;
(function (OutputSerialization) {
    OutputSerialization.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(OutputSerialization || (OutputSerialization = {}));
export var SelectParameters;
(function (SelectParameters) {
    SelectParameters.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(SelectParameters || (SelectParameters = {}));
export var RestoreRequestType;
(function (RestoreRequestType) {
    RestoreRequestType["SELECT"] = "SELECT";
})(RestoreRequestType || (RestoreRequestType = {}));
export var RestoreRequest;
(function (RestoreRequest) {
    RestoreRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.OutputLocation && { OutputLocation: OutputLocation.filterSensitiveLog(obj.OutputLocation) }),
    });
})(RestoreRequest || (RestoreRequest = {}));
export var RestoreObjectRequest;
(function (RestoreObjectRequest) {
    RestoreObjectRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.RestoreRequest && { RestoreRequest: RestoreRequest.filterSensitiveLog(obj.RestoreRequest) }),
    });
})(RestoreObjectRequest || (RestoreObjectRequest = {}));
export var ContinuationEvent;
(function (ContinuationEvent) {
    ContinuationEvent.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ContinuationEvent || (ContinuationEvent = {}));
export var EndEvent;
(function (EndEvent) {
    EndEvent.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(EndEvent || (EndEvent = {}));
export var Progress;
(function (Progress) {
    Progress.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(Progress || (Progress = {}));
export var ProgressEvent;
(function (ProgressEvent) {
    ProgressEvent.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ProgressEvent || (ProgressEvent = {}));
export var RecordsEvent;
(function (RecordsEvent) {
    RecordsEvent.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(RecordsEvent || (RecordsEvent = {}));
export var Stats;
(function (Stats) {
    Stats.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(Stats || (Stats = {}));
export var StatsEvent;
(function (StatsEvent) {
    StatsEvent.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(StatsEvent || (StatsEvent = {}));
export var SelectObjectContentEventStream;
(function (SelectObjectContentEventStream) {
    SelectObjectContentEventStream.visit = (value, visitor) => {
        if (value.Records !== undefined)
            return visitor.Records(value.Records);
        if (value.Stats !== undefined)
            return visitor.Stats(value.Stats);
        if (value.Progress !== undefined)
            return visitor.Progress(value.Progress);
        if (value.Cont !== undefined)
            return visitor.Cont(value.Cont);
        if (value.End !== undefined)
            return visitor.End(value.End);
        return visitor._(value.$unknown[0], value.$unknown[1]);
    };
    SelectObjectContentEventStream.filterSensitiveLog = (obj) => {
        if (obj.Records !== undefined)
            return { Records: RecordsEvent.filterSensitiveLog(obj.Records) };
        if (obj.Stats !== undefined)
            return { Stats: StatsEvent.filterSensitiveLog(obj.Stats) };
        if (obj.Progress !== undefined)
            return { Progress: ProgressEvent.filterSensitiveLog(obj.Progress) };
        if (obj.Cont !== undefined)
            return { Cont: ContinuationEvent.filterSensitiveLog(obj.Cont) };
        if (obj.End !== undefined)
            return { End: EndEvent.filterSensitiveLog(obj.End) };
        if (obj.$unknown !== undefined)
            return { [obj.$unknown[0]]: "UNKNOWN" };
    };
})(SelectObjectContentEventStream || (SelectObjectContentEventStream = {}));
export var SelectObjectContentOutput;
(function (SelectObjectContentOutput) {
    SelectObjectContentOutput.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.Payload && { Payload: "STREAMING_CONTENT" }),
    });
})(SelectObjectContentOutput || (SelectObjectContentOutput = {}));
export var RequestProgress;
(function (RequestProgress) {
    RequestProgress.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(RequestProgress || (RequestProgress = {}));
export var ScanRange;
(function (ScanRange) {
    ScanRange.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ScanRange || (ScanRange = {}));
export var SelectObjectContentRequest;
(function (SelectObjectContentRequest) {
    SelectObjectContentRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.SSECustomerKey && { SSECustomerKey: SENSITIVE_STRING }),
    });
})(SelectObjectContentRequest || (SelectObjectContentRequest = {}));
export var UploadPartOutput;
(function (UploadPartOutput) {
    UploadPartOutput.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.SSEKMSKeyId && { SSEKMSKeyId: SENSITIVE_STRING }),
    });
})(UploadPartOutput || (UploadPartOutput = {}));
export var UploadPartRequest;
(function (UploadPartRequest) {
    UploadPartRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.SSECustomerKey && { SSECustomerKey: SENSITIVE_STRING }),
    });
})(UploadPartRequest || (UploadPartRequest = {}));
export var CopyPartResult;
(function (CopyPartResult) {
    CopyPartResult.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(CopyPartResult || (CopyPartResult = {}));
export var UploadPartCopyOutput;
(function (UploadPartCopyOutput) {
    UploadPartCopyOutput.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.SSEKMSKeyId && { SSEKMSKeyId: SENSITIVE_STRING }),
    });
})(UploadPartCopyOutput || (UploadPartCopyOutput = {}));
export var UploadPartCopyRequest;
(function (UploadPartCopyRequest) {
    UploadPartCopyRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.SSECustomerKey && { SSECustomerKey: SENSITIVE_STRING }),
        ...(obj.CopySourceSSECustomerKey && { CopySourceSSECustomerKey: SENSITIVE_STRING }),
    });
})(UploadPartCopyRequest || (UploadPartCopyRequest = {}));
export var WriteGetObjectResponseRequest;
(function (WriteGetObjectResponseRequest) {
    WriteGetObjectResponseRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.SSEKMSKeyId && { SSEKMSKeyId: SENSITIVE_STRING }),
    });
})(WriteGetObjectResponseRequest || (WriteGetObjectResponseRequest = {}));
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kZWxzXzEuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJtb2RlbHNfMS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFjQSxPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQTRCOUQsTUFBTSxLQUFXLFVBQVUsQ0FRMUI7QUFSRCxXQUFpQixVQUFVO0lBSVosNkJBQWtCLEdBQUcsQ0FBQyxHQUFlLEVBQU8sRUFBRSxDQUFDLENBQUM7UUFDM0QsR0FBRyxHQUFHO1FBQ04sR0FBRyxDQUFDLEdBQUcsQ0FBQyxRQUFRLElBQUksRUFBRSxRQUFRLEVBQUUsZ0JBQWdCLEVBQUUsQ0FBQztLQUNwRCxDQUFDLENBQUM7QUFDTCxDQUFDLEVBUmdCLFVBQVUsS0FBVixVQUFVLFFBUTFCO0FBaUJELE1BQU0sS0FBVyxhQUFhLENBTzdCO0FBUEQsV0FBaUIsYUFBYTtJQUlmLGdDQUFrQixHQUFHLENBQUMsR0FBa0IsRUFBTyxFQUFFLENBQUMsQ0FBQztRQUM5RCxHQUFHLEdBQUc7S0FDUCxDQUFDLENBQUM7QUFDTCxDQUFDLEVBUGdCLGFBQWEsS0FBYixhQUFhLFFBTzdCO0FBK0NELE1BQU0sS0FBVyxVQUFVLENBUTFCO0FBUkQsV0FBaUIsVUFBVTtJQUlaLDZCQUFrQixHQUFHLENBQUMsR0FBZSxFQUFPLEVBQUUsQ0FBQyxDQUFDO1FBQzNELEdBQUcsR0FBRztRQUNOLEdBQUcsQ0FBQyxHQUFHLENBQUMsVUFBVSxJQUFJLEVBQUUsVUFBVSxFQUFFLFVBQVUsQ0FBQyxrQkFBa0IsQ0FBQyxHQUFHLENBQUMsVUFBVSxDQUFDLEVBQUUsQ0FBQztLQUNyRixDQUFDLENBQUM7QUFDTCxDQUFDLEVBUmdCLFVBQVUsS0FBVixVQUFVLFFBUTFCO0FBWUQsTUFBTSxLQUFXLGNBQWMsQ0FROUI7QUFSRCxXQUFpQixjQUFjO0lBSWhCLGlDQUFrQixHQUFHLENBQUMsR0FBbUIsRUFBTyxFQUFFLENBQUMsQ0FBQztRQUMvRCxHQUFHLEdBQUc7UUFD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