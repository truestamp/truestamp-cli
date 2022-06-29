export var errorUtil;
(function(errorUtil1) {
    var errToObj = errorUtil1.errToObj = (message)=>typeof message === "string" ? {
            message
        } : message || {};
    var toString = errorUtil1.toString = (message)=>typeof message === "string" ? message : message?.message;
})(errorUtil || (errorUtil = {}));
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvem9kQHYzLjE3LjMvaGVscGVycy9lcnJvclV0aWwudHMiXSwic291cmNlc0NvbnRlbnQiOlsiZXhwb3J0IG5hbWVzcGFjZSBlcnJvclV0aWwge1xuICBleHBvcnQgdHlwZSBFcnJNZXNzYWdlID0gc3RyaW5nIHwgeyBtZXNzYWdlPzogc3RyaW5nIH07XG4gIGV4cG9ydCBjb25zdCBlcnJUb09iaiA9IChtZXNzYWdlPzogRXJyTWVzc2FnZSkgPT5cbiAgICB0eXBlb2YgbWVzc2FnZSA9PT0gXCJzdHJpbmdcIiA/IHsgbWVzc2FnZSB9IDogbWVzc2FnZSB8fCB7fTtcbiAgZXhwb3J0IGNvbnN0IHRvU3RyaW5nID0gKG1lc3NhZ2U/OiBFcnJNZXNzYWdlKTogc3RyaW5nIHwgdW5kZWZpbmVkID0+XG4gICAgdHlwZW9mIG1lc3NhZ2UgPT09IFwic3RyaW5nXCIgPyBtZXNzYWdlIDogbWVzc2FnZT8ubWVzc2FnZTtcbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLElBQVUsU0FBUyxDQU16Qjs7UUFKYyxRQUFRLGNBQVIsUUFBUSxHQUFHLENBQUMsT0FBb0IsR0FDM0MsT0FBTyxPQUFPLEtBQUssUUFBUSxHQUFHO1lBQUUsT0FBTztTQUFFLEdBQUcsT0FBTyxJQUFJLEVBQUUsQUFEdEM7UUFFUixRQUFRLGNBQVIsUUFBUSxHQUFHLENBQUMsT0FBb0IsR0FDM0MsT0FBTyxPQUFPLEtBQUssUUFBUSxHQUFHLE9BQU8sR0FBRyxPQUFPLEVBQUUsT0FBTyxBQURyQztHQUpOLFNBQVMsS0FBVCxTQUFTIn0=