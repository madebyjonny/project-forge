import { HTTPException } from "hono/http-exception";
import type { ContentfulStatusCode } from "hono/utils/http-status";

export class HttpError extends HTTPException {
  constructor(status: ContentfulStatusCode, message: string, public details?: Record<string, unknown>) {
    super(status, { message });
  }
}

export class NotFoundError extends HttpError {
  constructor(message = "Not found") {
    super(404, message);
  }
}

export class UnauthorizedError extends HttpError {
  constructor(message = "Unauthenticated") {
    super(401, message);
  }
}

export class ForbiddenError extends HttpError {
  constructor(message = "Forbidden") {
    super(403, message);
  }
}

export class ValidationError extends HttpError {
  constructor(public errors: Record<string, string[]>) {
    super(422, "The given data was invalid.", { errors });
  }
}

/** Throw an HTTP error from anywhere (≈ Laravel's abort()). */
export function abort(status: ContentfulStatusCode, message?: string): never {
  throw new HttpError(status, message ?? "Error");
}
