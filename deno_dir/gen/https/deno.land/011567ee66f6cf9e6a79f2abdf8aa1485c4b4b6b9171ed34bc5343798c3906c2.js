import { blobReader } from "../chunked-blob-reader/mod.ts";
export const blobHasher = async function blobHasher(hashCtor, blob) {
    const hash = new hashCtor();
    await blobReader(blob, (chunk) => {
        hash.update(chunk);
    });
    return hash.digest();
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUczRCxNQUFNLENBQUMsTUFBTSxVQUFVLEdBQXVCLEtBQUssVUFBVSxVQUFVLENBQ3JFLFFBQXlCLEVBQ3pCLElBQVU7SUFFVixNQUFNLElBQUksR0FBRyxJQUFJLFFBQVEsRUFBRSxDQUFDO0lBRTVCLE1BQU0sVUFBVSxDQUFDLElBQUksRUFBRSxDQUFDLEtBQUssRUFBRSxFQUFFO1FBQy9CLElBQUksQ0FBQyxNQUFNLENBQUMsS0FBSyxDQUFDLENBQUM7SUFDckIsQ0FBQyxDQUFDLENBQUM7SUFFSCxPQUFPLElBQUksQ0FBQyxNQUFNLEVBQUUsQ0FBQztBQUN2QixDQUFDLENBQUMifQ==