import { createApp, serve } from "~/core/app";
import middleware from "~/app/middleware";
import web from "~/routes/web";
import api from "~/routes/api";
import features from "./features";

export const app = createApp({
  features,
  middleware,
  routes: {
    web: { prefix: "/", define: web },
    api: { prefix: "/api", define: api },
  },
});

export default serve(app, features);
