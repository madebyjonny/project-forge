// Tables owned by the OAuth/MCP layer: registered clients, tokens, consents and
// signing keys. Managed by the auth library — you shouldn't need to touch these.
import { t, table } from "../../core/schema";
import { sessions } from "./auth";
import { users } from "./user";

const { stringArray, json } = t.raw;
const nullableTimestamps = () => ({ createdAt: t.timestamp("created_at"), updatedAt: t.timestamp("updated_at") });

export const jwks = table("jwks", {
  id: t.id(),
  publicKey: t.text("public_key").notNull(),
  privateKey: t.text("private_key").notNull(),
  createdAt: t.timestamp("created_at").notNull(),
  expiresAt: t.timestamp("expires_at"),
  alg: t.string("alg"),
  crv: t.string("crv"),
});

export const oauthClients = table("oauth_clients", {
  id: t.id(),
  clientId: t.string("client_id").notNull().unique(),
  clientSecret: t.string("client_secret"),
  clientDiscoveryId: t.string("client_discovery_id"),
  disabled: t.boolean("disabled"),
  skipConsent: t.boolean("skip_consent"),
  enableEndSession: t.boolean("enable_end_session"),
  subjectType: t.string("subject_type"),
  scopes: stringArray("scopes"),
  clientCredentialsScopes: stringArray("client_credentials_scopes"),
  userId: t.string("user_id").references(() => users.id, { onDelete: "cascade" }),
  name: t.string("name"),
  uri: t.string("uri"),
  icon: t.string("icon"),
  contacts: stringArray("contacts"),
  tos: t.string("tos"),
  policy: t.string("policy"),
  softwareId: t.string("software_id"),
  softwareVersion: t.string("software_version"),
  softwareStatement: t.text("software_statement"),
  redirectUris: stringArray("redirect_uris").notNull(),
  postLogoutRedirectUris: stringArray("post_logout_redirect_uris"),
  backchannelLogoutUri: t.string("backchannel_logout_uri"),
  backchannelLogoutSessionRequired: t.boolean("backchannel_logout_session_required"),
  tokenEndpointAuthMethod: t.string("token_endpoint_auth_method"),
  applicationType: t.string("application_type"),
  jwks: t.text("jwks"),
  jwksUri: t.string("jwks_uri"),
  grantTypes: stringArray("grant_types"),
  responseTypes: stringArray("response_types"),
  requirePKCE: t.boolean("require_pkce"),
  dpopBoundAccessTokens: t.boolean("dpop_bound_access_tokens"),
  referenceId: t.string("reference_id"),
  metadata: json("metadata"),
  ...nullableTimestamps(),
});

export const oauthResources = table("oauth_resources", {
  id: t.id(),
  identifier: t.string("identifier").notNull().unique(),
  name: t.string("name").notNull(),
  accessTokenTtl: t.integer("access_token_ttl"),
  refreshTokenTtl: t.integer("refresh_token_ttl"),
  signingAlgorithm: t.string("signing_algorithm"),
  signingKeyId: t.string("signing_key_id"),
  allowedScopes: stringArray("allowed_scopes"),
  customClaims: json("custom_claims"),
  dpopBoundAccessTokensRequired: t.boolean("dpop_bound_access_tokens_required"),
  disabled: t.boolean("disabled"),
  policyVersion: t.integer("policy_version"),
  metadata: json("metadata"),
  ...nullableTimestamps(),
});

export const oauthClientResources = table("oauth_client_resources", {
  id: t.id(),
  clientId: t.string("client_id").notNull().references(() => oauthClients.clientId, { onDelete: "cascade" }),
  resourceId: t.string("resource_id").notNull().references(() => oauthResources.identifier, { onDelete: "cascade" }),
  metadata: json("metadata"),
  createdAt: t.timestamp("created_at"),
});

const tokenColumns = () => ({
  clientId: t.string("client_id").notNull().references(() => oauthClients.clientId, { onDelete: "cascade" }),
  sessionId: t.string("session_id").references(() => sessions.id, { onDelete: "set null" }),
  referenceId: t.string("reference_id"),
  authorizationCodeId: t.string("authorization_code_id"),
  resources: stringArray("resources"),
  requestedUserInfoClaims: stringArray("requested_user_info_claims"),
  expiresAt: t.timestamp("expires_at"),
  createdAt: t.timestamp("created_at"),
  revoked: t.timestamp("revoked"),
  confirmation: json("confirmation"),
  scopes: stringArray("scopes").notNull(),
});

export const oauthRefreshTokens = table("oauth_refresh_tokens", {
  id: t.id(),
  token: t.string("token").notNull().unique(),
  userId: t.string("user_id").notNull().references(() => users.id, { onDelete: "cascade" }),
  rotatedAt: t.timestamp("rotated_at"),
  rotationReplayResponse: t.text("rotation_replay_response"),
  rotationReplayExpiresAt: t.timestamp("rotation_replay_expires_at"),
  authTime: t.timestamp("auth_time"),
  ...tokenColumns(),
});

export const oauthAccessTokens = table("oauth_access_tokens", {
  id: t.id(),
  token: t.string("token").unique(),
  userId: t.string("user_id").references(() => users.id, { onDelete: "cascade" }),
  refreshId: t.string("refresh_id").references(() => oauthRefreshTokens.id, { onDelete: "cascade" }),
  ...tokenColumns(),
});

export const oauthConsents = table("oauth_consents", {
  id: t.id(),
  clientId: t.string("client_id").notNull().references(() => oauthClients.clientId, { onDelete: "cascade" }),
  userId: t.string("user_id").references(() => users.id, { onDelete: "cascade" }),
  referenceId: t.string("reference_id"),
  resources: stringArray("resources"),
  requestedUserInfoClaims: stringArray("requested_user_info_claims"),
  scopes: stringArray("scopes").notNull(),
  ...nullableTimestamps(),
});

export const oauthClientAssertions = table("oauth_client_assertions", {
  id: t.id(),
  expiresAt: t.timestamp("expires_at").notNull(),
});

/** Model-name → table map handed to the auth adapter. */
export const oauthSchema = {
  jwks,
  oauthClient: oauthClients,
  oauthResource: oauthResources,
  oauthClientResource: oauthClientResources,
  oauthRefreshToken: oauthRefreshTokens,
  oauthAccessToken: oauthAccessTokens,
  oauthConsent: oauthConsents,
  oauthClientAssertion: oauthClientAssertions,
};
