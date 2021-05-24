export function destroyer(stream, err) {
    if (typeof stream.destroy === "function") {
        return stream.destroy(err);
    }
    if (typeof stream.close === "function") {
        return stream.close();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVzdHJveS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImRlc3Ryb3kudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBb0JBLE1BQU0sVUFBVSxTQUFTLENBQUMsTUFBYyxFQUFFLEdBQWtCO0lBSzFELElBQ0UsT0FBUSxNQUFnQyxDQUFDLE9BQU8sS0FBSyxVQUFVLEVBQy9EO1FBQ0EsT0FBUSxNQUFnQyxDQUFDLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQztLQUN2RDtJQUlELElBQUksT0FBUSxNQUFjLENBQUMsS0FBSyxLQUFLLFVBQVUsRUFBRTtRQUUvQyxPQUFRLE1BQWMsQ0FBQyxLQUFLLEVBQUUsQ0FBQztLQUNoQztBQUNILENBQUMifQ==