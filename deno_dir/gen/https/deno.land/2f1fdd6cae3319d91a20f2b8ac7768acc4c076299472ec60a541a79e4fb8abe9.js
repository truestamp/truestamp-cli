export default function configDir() {
    switch (Deno.build.os) {
        case "linux": {
            const xdg = Deno.env.get("XDG_CONFIG_HOME");
            if (xdg)
                return xdg;
            const home = Deno.env.get("HOME");
            if (home)
                return `${home}/.config`;
            break;
        }
        case "darwin": {
            const home = Deno.env.get("HOME");
            if (home)
                return `${home}/Library/Preferences`;
            break;
        }
        case "windows":
            return Deno.env.get("FOLDERID_RoamingAppData") ?? null;
    }
    return null;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQVdBLE1BQU0sQ0FBQyxPQUFPLFVBQVUsU0FBUztJQUMvQixRQUFRLElBQUksQ0FBQyxLQUFLLENBQUMsRUFBRSxFQUFFO1FBQ3JCLEtBQUssT0FBTyxDQUFDLENBQUM7WUFDWixNQUFNLEdBQUcsR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsQ0FBQyxDQUFDO1lBQzVDLElBQUksR0FBRztnQkFBRSxPQUFPLEdBQUcsQ0FBQztZQUVwQixNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxNQUFNLENBQUMsQ0FBQztZQUNsQyxJQUFJLElBQUk7Z0JBQUUsT0FBTyxHQUFHLElBQUksVUFBVSxDQUFDO1lBQ25DLE1BQU07U0FDUDtRQUVELEtBQUssUUFBUSxDQUFDLENBQUM7WUFDYixNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxNQUFNLENBQUMsQ0FBQztZQUNsQyxJQUFJLElBQUk7Z0JBQUUsT0FBTyxHQUFHLElBQUksc0JBQXNCLENBQUM7WUFDL0MsTUFBTTtTQUNQO1FBRUQsS0FBSyxTQUFTO1lBQ1osT0FBTyxJQUFJLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyx5QkFBeUIsQ0FBQyxJQUFJLElBQUksQ0FBQztLQUMxRDtJQUVELE9BQU8sSUFBSSxDQUFDO0FBQ2QsQ0FBQyJ9