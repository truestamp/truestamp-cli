import { fromFileUrl } from "../path.ts";
export function exists(path, callback) {
    path = path instanceof URL ? fromFileUrl(path) : path;
    Deno.lstat(path).then(() => callback(true), () => callback(false));
}
export function existsSync(path) {
    path = path instanceof URL ? fromFileUrl(path) : path;
    try {
        Deno.lstatSync(path);
        return true;
    }
    catch (err) {
        if (err instanceof Deno.errors.NotFound) {
            return false;
        }
        throw err;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX2V4aXN0cy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9mc19leGlzdHMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLFlBQVksQ0FBQztBQVN6QyxNQUFNLFVBQVUsTUFBTSxDQUFDLElBQWtCLEVBQUUsUUFBdUI7SUFDaEUsSUFBSSxHQUFHLElBQUksWUFBWSxHQUFHLENBQUMsQ0FBQyxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDO0lBQ3RELElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLENBQUMsSUFBSSxDQUFDLEdBQUcsRUFBRSxDQUFDLFFBQVEsQ0FBQyxJQUFJLENBQUMsRUFBRSxHQUFHLEVBQUUsQ0FBQyxRQUFRLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQztBQUNyRSxDQUFDO0FBTUQsTUFBTSxVQUFVLFVBQVUsQ0FBQyxJQUFrQjtJQUMzQyxJQUFJLEdBQUcsSUFBSSxZQUFZLEdBQUcsQ0FBQyxDQUFDLENBQUMsV0FBVyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUM7SUFDdEQsSUFBSTtRQUNGLElBQUksQ0FBQyxTQUFTLENBQUMsSUFBSSxDQUFDLENBQUM7UUFDckIsT0FBTyxJQUFJLENBQUM7S0FDYjtJQUFDLE9BQU8sR0FBRyxFQUFFO1FBQ1osSUFBSSxHQUFHLFlBQVksSUFBSSxDQUFDLE1BQU0sQ0FBQyxRQUFRLEVBQUU7WUFDdkMsT0FBTyxLQUFLLENBQUM7U0FDZDtRQUNELE1BQU0sR0FBRyxDQUFDO0tBQ1g7QUFDSCxDQUFDIn0=