// Authorization (≈ Gates & Policies), backed by CASL. Policies live in app/policies.
import { AbilityBuilder, createMongoAbility, detectSubjectType, type MongoAbility } from "@casl/ability";
import policies from "../app/policies";
import type { Feature } from "./app";
import { ForbiddenError } from "./errors";
import { user, type AuthUser, type Ctx } from "./http";
import { alias } from "./middleware";
import { MODEL } from "./model";

export type Ability = MongoAbility;
type Rules = Pick<AbilityBuilder<Ability>, "can" | "cannot">;
export type Policy = (user: AuthUser | null, rules: Rules) => void;

/** Define a policy: grant or deny abilities for the given user. */
export function policy(define: Policy): Policy {
  return define;
}

export function abilityFor(u: AuthUser | null): Ability {
  const builder = new AbilityBuilder<Ability>(createMongoAbility);
  for (const p of policies) p(u, builder);
  return builder.build({ detectSubjectType: (o: any) => o?.[MODEL] ?? detectSubjectType(o) });
}

/** Can the current user perform `action` on a model record or subject name? */
export function can(c: Ctx, action: string, subject: object | string): boolean {
  return abilityFor(user(c)).can(action, subject as any);
}

/** Like can(), but throws a 403 (≈ $this->authorize()). */
export function authorize(c: Ctx, action: string, subject: object | string): void {
  if (!can(c, action, subject)) throw new ForbiddenError();
}

/** Serialisable rules for the frontend to hide what the user can't do. */
export function rulesFor(c: Ctx) {
  return abilityFor(user(c)).rules;
}

export default function policiesFeature(): Feature {
  return {
    name: "policies",
    setup() {
      // r.post("/posts", "can:create,Post", PostController.store)
      alias("can", (c, _next, [action = "", subject = ""]) => authorize(c, action, subject));
    },
  };
}
