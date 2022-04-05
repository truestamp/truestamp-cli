export var Status;
(function (Status) {
    Status[Status["Unknown"] = -1] = "Unknown";
    Status[Status["SqliteOk"] = 0] = "SqliteOk";
    Status[Status["SqliteError"] = 1] = "SqliteError";
    Status[Status["SqliteInternal"] = 2] = "SqliteInternal";
    Status[Status["SqlitePerm"] = 3] = "SqlitePerm";
    Status[Status["SqliteAbort"] = 4] = "SqliteAbort";
    Status[Status["SqliteBusy"] = 5] = "SqliteBusy";
    Status[Status["SqliteLocked"] = 6] = "SqliteLocked";
    Status[Status["SqliteNoMem"] = 7] = "SqliteNoMem";
    Status[Status["SqliteReadOnly"] = 8] = "SqliteReadOnly";
    Status[Status["SqliteInterrupt"] = 9] = "SqliteInterrupt";
    Status[Status["SqliteIOErr"] = 10] = "SqliteIOErr";
    Status[Status["SqliteCorrupt"] = 11] = "SqliteCorrupt";
    Status[Status["SqliteNotFound"] = 12] = "SqliteNotFound";
    Status[Status["SqliteFull"] = 13] = "SqliteFull";
    Status[Status["SqliteCantOpen"] = 14] = "SqliteCantOpen";
    Status[Status["SqliteProtocol"] = 15] = "SqliteProtocol";
    Status[Status["SqliteEmpty"] = 16] = "SqliteEmpty";
    Status[Status["SqliteSchema"] = 17] = "SqliteSchema";
    Status[Status["SqliteTooBig"] = 18] = "SqliteTooBig";
    Status[Status["SqliteConstraint"] = 19] = "SqliteConstraint";
    Status[Status["SqliteMismatch"] = 20] = "SqliteMismatch";
    Status[Status["SqliteMisuse"] = 21] = "SqliteMisuse";
    Status[Status["SqliteNoLFS"] = 22] = "SqliteNoLFS";
    Status[Status["SqliteAuth"] = 23] = "SqliteAuth";
    Status[Status["SqliteFormat"] = 24] = "SqliteFormat";
    Status[Status["SqliteRange"] = 25] = "SqliteRange";
    Status[Status["SqliteNotADB"] = 26] = "SqliteNotADB";
    Status[Status["SqliteNotice"] = 27] = "SqliteNotice";
    Status[Status["SqliteWarning"] = 28] = "SqliteWarning";
    Status[Status["SqliteRow"] = 100] = "SqliteRow";
    Status[Status["SqliteDone"] = 101] = "SqliteDone";
})(Status || (Status = {}));
export var OpenFlags;
(function (OpenFlags) {
    OpenFlags[OpenFlags["ReadOnly"] = 1] = "ReadOnly";
    OpenFlags[OpenFlags["ReadWrite"] = 2] = "ReadWrite";
    OpenFlags[OpenFlags["Create"] = 4] = "Create";
    OpenFlags[OpenFlags["Uri"] = 64] = "Uri";
    OpenFlags[OpenFlags["Memory"] = 128] = "Memory";
})(OpenFlags || (OpenFlags = {}));
export var Types;
(function (Types) {
    Types[Types["Integer"] = 1] = "Integer";
    Types[Types["Float"] = 2] = "Float";
    Types[Types["Text"] = 3] = "Text";
    Types[Types["Blob"] = 4] = "Blob";
    Types[Types["Null"] = 5] = "Null";
    Types[Types["BigInteger"] = 6] = "BigInteger";
})(Types || (Types = {}));
export var Values;
(function (Values) {
    Values[Values["Error"] = -1] = "Error";
    Values[Values["Null"] = 0] = "Null";
})(Values || (Values = {}));
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29uc3RhbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiY29uc3RhbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQU1BLE1BQU0sQ0FBTixJQUFZLE1Ba0NYO0FBbENELFdBQVksTUFBTTtJQUNoQiwwQ0FBWSxDQUFBO0lBRVosMkNBQVksQ0FBQTtJQUNaLGlEQUFlLENBQUE7SUFDZix1REFBa0IsQ0FBQTtJQUNsQiwrQ0FBYyxDQUFBO0lBQ2QsaURBQWUsQ0FBQTtJQUNmLCtDQUFjLENBQUE7SUFDZCxtREFBZ0IsQ0FBQTtJQUNoQixpREFBZSxDQUFBO0lBQ2YsdURBQWtCLENBQUE7SUFDbEIseURBQW1CLENBQUE7SUFDbkIsa0RBQWdCLENBQUE7SUFDaEIsc0RBQWtCLENBQUE7SUFDbEIsd0RBQW1CLENBQUE7SUFDbkIsZ0RBQWUsQ0FBQTtJQUNmLHdEQUFtQixDQUFBO0lBQ25CLHdEQUFtQixDQUFBO0lBQ25CLGtEQUFnQixDQUFBO0lBQ2hCLG9EQUFpQixDQUFBO0lBQ2pCLG9EQUFpQixDQUFBO0lBQ2pCLDREQUFxQixDQUFBO0lBQ3JCLHdEQUFtQixDQUFBO0lBQ25CLG9EQUFpQixDQUFBO0lBQ2pCLGtEQUFnQixDQUFBO0lBQ2hCLGdEQUFlLENBQUE7SUFDZixvREFBaUIsQ0FBQTtJQUNqQixrREFBZ0IsQ0FBQTtJQUNoQixvREFBaUIsQ0FBQTtJQUNqQixvREFBaUIsQ0FBQTtJQUNqQixzREFBa0IsQ0FBQTtJQUNsQiwrQ0FBZSxDQUFBO0lBQ2YsaURBQWdCLENBQUE7QUFDbEIsQ0FBQyxFQWxDVyxNQUFNLEtBQU4sTUFBTSxRQWtDakI7QUFFRCxNQUFNLENBQU4sSUFBWSxTQU1YO0FBTkQsV0FBWSxTQUFTO0lBQ25CLGlEQUFxQixDQUFBO0lBQ3JCLG1EQUFzQixDQUFBO0lBQ3RCLDZDQUFtQixDQUFBO0lBQ25CLHdDQUFnQixDQUFBO0lBQ2hCLCtDQUFtQixDQUFBO0FBQ3JCLENBQUMsRUFOVyxTQUFTLEtBQVQsU0FBUyxRQU1wQjtBQUVELE1BQU0sQ0FBTixJQUFZLEtBT1g7QUFQRCxXQUFZLEtBQUs7SUFDZix1Q0FBVyxDQUFBO0lBQ1gsbUNBQVMsQ0FBQTtJQUNULGlDQUFRLENBQUE7SUFDUixpQ0FBUSxDQUFBO0lBQ1IsaUNBQVEsQ0FBQTtJQUNSLDZDQUFjLENBQUE7QUFDaEIsQ0FBQyxFQVBXLEtBQUssS0FBTCxLQUFLLFFBT2hCO0FBRUQsTUFBTSxDQUFOLElBQVksTUFHWDtBQUhELFdBQVksTUFBTTtJQUNoQixzQ0FBVSxDQUFBO0lBQ1YsbUNBQVEsQ0FBQTtBQUNWLENBQUMsRUFIVyxNQUFNLEtBQU4sTUFBTSxRQUdqQiIsInNvdXJjZXNDb250ZW50IjpbIi8qKlxuICogU3RhdHVzIGNvZGVzIHdoaWNoIGNhbiBiZSByZXR1cm5lZFxuICogYnkgU1FMaXRlLlxuICpcbiAqIEFsc28gc2VlIGh0dHBzOi8vd3d3LnNxbGl0ZS5vcmcvcmVzY29kZS5odG1sLlxuICovXG5leHBvcnQgZW51bSBTdGF0dXMge1xuICBVbmtub3duID0gLTEsIC8vIFVua25vd24gc3RhdHVzXG5cbiAgU3FsaXRlT2sgPSAwLCAvLyBTdWNjZXNzZnVsIHJlc3VsdFxuICBTcWxpdGVFcnJvciA9IDEsIC8vIEdlbmVyaWMgZXJyb3JcbiAgU3FsaXRlSW50ZXJuYWwgPSAyLCAvLyBJbnRlcm5hbCBsb2dpYyBlcnJvciBpbiBTUUxpdGVcbiAgU3FsaXRlUGVybSA9IDMsIC8vIEFjY2VzcyBwZXJtaXNzaW9uIGRlbmllZFxuICBTcWxpdGVBYm9ydCA9IDQsIC8vIENhbGxiYWNrIHJvdXRpbmUgcmVxdWVzdGVkIGFuIGFib3J0XG4gIFNxbGl0ZUJ1c3kgPSA1LCAvLyBUaGUgZGF0YWJhc2UgZmlsZSBpcyBsb2NrZWRcbiAgU3FsaXRlTG9ja2VkID0gNiwgLy8gQSB0YWJsZSBpbiB0aGUgZGF0YWJhc2UgaXMgbG9ja2VkXG4gIFNxbGl0ZU5vTWVtID0gNywgLy8gQSBtYWxsb2MoKSBmYWlsZWRcbiAgU3FsaXRlUmVhZE9ubHkgPSA4LCAvLyBBdHRlbXB0IHRvIHdyaXRlIGEgcmVhZG9ubHkgZGF0YWJhc2VcbiAgU3FsaXRlSW50ZXJydXB0ID0gOSwgLy8gT3BlcmF0aW9uIHRlcm1pbmF0ZWQgYnkgc3FsaXRlM19pbnRlcnJ1cHQoKVxuICBTcWxpdGVJT0VyciA9IDEwLCAvLyBTb21lIGtpbmQgb2YgZGlzayBJL08gZXJyb3Igb2NjdXJyZWRcbiAgU3FsaXRlQ29ycnVwdCA9IDExLCAvLyBUaGUgZGF0YWJhc2UgZGlzayBpbWFnZSBpcyBtYWxmb3JtZWRcbiAgU3FsaXRlTm90Rm91bmQgPSAxMiwgLy8gVW5rbm93biBvcGNvZGUgaW4gc3FsaXRlM19maWxlX2NvbnRyb2woKVxuICBTcWxpdGVGdWxsID0gMTMsIC8vIEluc2VydGlvbiBmYWlsZWQgYmVjYXVzZSBkYXRhYmFzZSBpcyBmdWxsXG4gIFNxbGl0ZUNhbnRPcGVuID0gMTQsIC8vIFVuYWJsZSB0byBvcGVuIHRoZSBkYXRhYmFzZSBmaWxlXG4gIFNxbGl0ZVByb3RvY29sID0gMTUsIC8vIERhdGFiYXNlIGxvY2sgcHJvdG9jb2wgZXJyb3JcbiAgU3FsaXRlRW1wdHkgPSAxNiwgLy8gSW50ZXJuYWwgdXNlIG9ubHlcbiAgU3FsaXRlU2NoZW1hID0gMTcsIC8vIFRoZSBkYXRhYmFzZSBzY2hlbWEgY2hhbmdlZFxuICBTcWxpdGVUb29CaWcgPSAxOCwgLy8gU3RyaW5nIG9yIEJMT0IgZXhjZWVkcyBzaXplIGxpbWl0XG4gIFNxbGl0ZUNvbnN0cmFpbnQgPSAxOSwgLy8gQWJvcnQgZHVlIHRvIGNvbnN0cmFpbnQgdmlvbGF0aW9uXG4gIFNxbGl0ZU1pc21hdGNoID0gMjAsIC8vIERhdGEgdHlwZSBtaXNtYXRjaFxuICBTcWxpdGVNaXN1c2UgPSAyMSwgLy8gTGlicmFyeSB1c2VkIGluY29ycmVjdGx5XG4gIFNxbGl0ZU5vTEZTID0gMjIsIC8vIFVzZXMgT1MgZmVhdHVyZXMgbm90IHN1cHBvcnRlZCBvbiBob3N0XG4gIFNxbGl0ZUF1dGggPSAyMywgLy8gQXV0aG9yaXphdGlvbiBkZW5pZWRcbiAgU3FsaXRlRm9ybWF0ID0gMjQsIC8vIE5vdCB1c2VkXG4gIFNxbGl0ZVJhbmdlID0gMjUsIC8vIDJuZCBwYXJhbWV0ZXIgdG8gc3FsaXRlM19iaW5kIG91dCBvZiByYW5nZVxuICBTcWxpdGVOb3RBREIgPSAyNiwgLy8gRmlsZSBvcGVuZWQgdGhhdCBpcyBub3QgYSBkYXRhYmFzZSBmaWxlXG4gIFNxbGl0ZU5vdGljZSA9IDI3LCAvLyBOb3RpZmljYXRpb25zIGZyb20gc3FsaXRlM19sb2coKVxuICBTcWxpdGVXYXJuaW5nID0gMjgsIC8vIFdhcm5pbmdzIGZyb20gc3FsaXRlM19sb2coKVxuICBTcWxpdGVSb3cgPSAxMDAsIC8vIHNxbGl0ZTNfc3RlcCgpIGhhcyBhbm90aGVyIHJvdyByZWFkeVxuICBTcWxpdGVEb25lID0gMTAxLCAvLyBzcWxpdGUzX3N0ZXAoKSBoYXMgZmluaXNoZWQgZXhlY3V0aW5nXG59XG5cbmV4cG9ydCBlbnVtIE9wZW5GbGFncyB7XG4gIFJlYWRPbmx5ID0gMHgwMDAwMDAwMSxcbiAgUmVhZFdyaXRlID0gMHgwMDAwMDAwMixcbiAgQ3JlYXRlID0gMHgwMDAwMDAwNCxcbiAgVXJpID0gMHgwMDAwMDA0MCxcbiAgTWVtb3J5ID0gMHgwMDAwMDA4MCxcbn1cblxuZXhwb3J0IGVudW0gVHlwZXMge1xuICBJbnRlZ2VyID0gMSxcbiAgRmxvYXQgPSAyLFxuICBUZXh0ID0gMyxcbiAgQmxvYiA9IDQsXG4gIE51bGwgPSA1LFxuICBCaWdJbnRlZ2VyID0gNixcbn1cblxuZXhwb3J0IGVudW0gVmFsdWVzIHtcbiAgRXJyb3IgPSAtMSxcbiAgTnVsbCA9IDAsXG59XG4iXX0=