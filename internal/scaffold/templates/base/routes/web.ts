import { routes } from "~/core/router";
import HomeController from "~/app/controllers/home.controller";
// forge:imports

export default routes((r) => {
  r.get("/", HomeController.index);
  r.get("/health", HomeController.health);
  // forge:routes
});
