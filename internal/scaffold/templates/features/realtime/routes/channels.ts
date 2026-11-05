import { channels } from "~/core/realtime";

// Who may listen on which channel. Unknown channels are always denied.
export default channels({
  "public.{name}": () => true,
  "user.{id}": (user, { id }) => user?.id === id,
});
