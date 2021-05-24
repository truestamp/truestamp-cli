import { escapeAttribute } from "./escape-attribute.ts";
export class XmlNode {
    name;
    children;
    attributes = {};
    constructor(name, children = []) {
        this.name = name;
        this.children = children;
    }
    withName(name) {
        this.name = name;
        return this;
    }
    addAttribute(name, value) {
        this.attributes[name] = value;
        return this;
    }
    addChildNode(child) {
        this.children.push(child);
        return this;
    }
    removeAttribute(name) {
        delete this.attributes[name];
        return this;
    }
    toString() {
        const hasChildren = Boolean(this.children.length);
        let xmlText = `<${this.name}`;
        const attributes = this.attributes;
        for (const attributeName of Object.keys(attributes)) {
            const attribute = attributes[attributeName];
            if (typeof attribute !== "undefined" && attribute !== null) {
                xmlText += ` ${attributeName}="${escapeAttribute("" + attribute)}"`;
            }
        }
        return (xmlText += !hasChildren ? "/>" : `>${this.children.map((c) => c.toString()).join("")}</${this.name}>`);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiWG1sTm9kZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIlhtbE5vZGUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLGVBQWUsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBTXhELE1BQU0sT0FBTyxPQUFPO0lBR0U7SUFBOEI7SUFGMUMsVUFBVSxHQUE0QixFQUFFLENBQUM7SUFFakQsWUFBb0IsSUFBWSxFQUFrQixXQUF5QixFQUFFO1FBQXpELFNBQUksR0FBSixJQUFJLENBQVE7UUFBa0IsYUFBUSxHQUFSLFFBQVEsQ0FBbUI7SUFBRyxDQUFDO0lBRWpGLFFBQVEsQ0FBQyxJQUFZO1FBQ25CLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBQ2pCLE9BQU8sSUFBSSxDQUFDO0lBQ2QsQ0FBQztJQUVELFlBQVksQ0FBQyxJQUFZLEVBQUUsS0FBVTtRQUNuQyxJQUFJLENBQUMsVUFBVSxDQUFDLElBQUksQ0FBQyxHQUFHLEtBQUssQ0FBQztRQUM5QixPQUFPLElBQUksQ0FBQztJQUNkLENBQUM7SUFFRCxZQUFZLENBQUMsS0FBaUI7UUFDNUIsSUFBSSxDQUFDLFFBQVEsQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLENBQUM7UUFDMUIsT0FBTyxJQUFJLENBQUM7SUFDZCxDQUFDO0lBRUQsZUFBZSxDQUFDLElBQVk7UUFDMUIsT0FBTyxJQUFJLENBQUMsVUFBVSxDQUFDLElBQUksQ0FBQyxDQUFDO1FBQzdCLE9BQU8sSUFBSSxDQUFDO0lBQ2QsQ0FBQztJQUVELFFBQVE7UUFDTixNQUFNLFdBQVcsR0FBRyxPQUFPLENBQUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxNQUFNLENBQUMsQ0FBQztRQUNsRCxJQUFJLE9BQU8sR0FBRyxJQUFJLElBQUksQ0FBQyxJQUFJLEVBQUUsQ0FBQztRQUU5QixNQUFNLFVBQVUsR0FBRyxJQUFJLENBQUMsVUFBVSxDQUFDO1FBQ25DLEtBQUssTUFBTSxhQUFhLElBQUksTUFBTSxDQUFDLElBQUksQ0FBQyxVQUFVLENBQUMsRUFBRTtZQUNuRCxNQUFNLFNBQVMsR0FBRyxVQUFVLENBQUMsYUFBYSxDQUFDLENBQUM7WUFDNUMsSUFBSSxPQUFPLFNBQVMsS0FBSyxXQUFXLElBQUksU0FBUyxLQUFLLElBQUksRUFBRTtnQkFDMUQsT0FBTyxJQUFJLElBQUksYUFBYSxLQUFLLGVBQWUsQ0FBQyxFQUFFLEdBQUcsU0FBUyxDQUFDLEdBQUcsQ0FBQzthQUNyRTtTQUNGO1FBRUQsT0FBTyxDQUFDLE9BQU8sSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxJQUFJLElBQUksQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUMsUUFBUSxFQUFFLENBQUMsQ0FBQyxJQUFJLENBQUMsRUFBRSxDQUFDLEtBQUssSUFBSSxDQUFDLElBQUksR0FBRyxDQUFDLENBQUM7SUFDakgsQ0FBQztDQUNGIn0=