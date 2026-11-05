// Turns the auth layer into an OAuth 2.1 authorization server for MCP clients.
// Better Auth's mcp() plugin issues audience-bound tokens; cimd() lets clients
// identify themselves with a Client ID Metadata Document URL.
import { cimd } from "@better-auth/cimd";
import { fetchClientMetadataResource } from "@better-auth/cimd/node";
import { mcp } from "@better-auth/mcp";
import { jwt } from "better-auth/plugins";
import config from "../config/mcp";
import { oauthSchema } from "../app/models/oauth";
import type { AuthExtension } from "./auth";

export function mcpAuth(): AuthExtension {
  const registration = config.dynamicClientRegistration
    ? {
        allowDynamicClientRegistration: true,
        allowUnauthenticatedClientRegistration: true,
        clientRegistrationDefaultApplicationType: "infer" as const,
        clientRegistrationDefaultScopes: config.scopes,
      }
    : {};

  return {
    plugins: [
      jwt({ disableSettingJwtHeader: true }),
      mcp({
        loginPage: config.loginPage,
        consentPage: config.consentPage,
        resource: config.resource,
        scopes: config.scopes,
        ...registration,
      }),
      cimd({ fetchClientMetadataResource, metadataProfile: "mcp-2026-07-28" }),
    ],
    // Session JWTs share the token signing key; never hand them out.
    disabledPaths: ["/token"],
    schema: oauthSchema,
  };
}
