import { routes } from "~/core/router";
import HomeController from "~/app/controllers/home.controller";
// forge:imports

// Everything here is prefixed with /api.
export default routes((r) => {
  r.get("/hello/:name?", HomeController.hello);
  // forge:routes
});
