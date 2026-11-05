import { controller, userOrFail } from "~/core/http";
import { rulesFor } from "~/core/policy";

// Sign up / sign in / sign out live under /api/auth/* (handled by the auth layer).
export default controller({
  me: (c) => c.json({ user: userOrFail(c), abilities: rulesFor(c) }),
});
