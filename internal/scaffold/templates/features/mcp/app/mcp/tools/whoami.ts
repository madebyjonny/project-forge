import { tool } from "~/core/mcp";

export default tool(
  "whoami",
  { description: "Show the account this connection is signed in as.", readOnly: true },
  (_input, { user, scopes }) => ({ id: user.id, name: user.name, email: user.email, role: user.role, scopes }),
);
