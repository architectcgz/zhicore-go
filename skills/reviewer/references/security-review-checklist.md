# Security Review Checklist

Read this file when the change affects authentication, authorization, input handling, secrets management, session handling, or any trust boundary.

This checklist follows OWASP Top 10 Proactive Controls and common security principles. Not every item applies to every change; focus on items relevant to the touched code paths.

## C1: Define Security Requirements

- Check whether security-sensitive features (authentication, authorization, payment, data export, account deletion) have explicit security requirements documented
- Verify that new security requirements are validated against threat models or risk assessments
- Flag changes that weaken existing security boundaries without explicit justification

## C2: Leverage Security Frameworks and Libraries

- Check whether the code uses well-established security libraries (e.g., bcrypt/scrypt for password hashing, jsonwebtoken for JWT, helmet for HTTP headers) instead of rolling custom implementations
- Verify that cryptographic operations use standard libraries with secure defaults (e.g., crypto.randomBytes, not Math.random())
- Flag any custom authentication, encryption, or signature verification logic

## C3: Secure Database Access

- Check for SQL injection risks: verify that all dynamic queries use parameterized statements, prepared statements, or ORM query builders
- Verify that ORM methods (e.g., Sequelize `findAll`, GORM `Where`) use bound parameters, not string concatenation
- Check NoSQL injection risks: verify that MongoDB queries do not directly interpolate user input into query objects
- Verify that database credentials are not hard-coded or logged

## C4: Encode and Escape Data

- Check for XSS risks: verify that user input rendered in HTML is escaped (e.g., Vue template binding `{{ }}`, React JSX, not `v-html` or `dangerouslySetInnerHTML` with unvalidated content)
- Check for command injection: verify that shell commands do not interpolate user input without proper escaping or use of array-based APIs
- Check for path traversal: verify that file paths constructed from user input are validated or canonicalized before use
- Verify that JSON responses do not contain unescaped user content that could break out of the JSON structure

## C5: Validate All Inputs

- Check whether user input is validated at trust boundaries (API endpoints, form handlers, file uploads)
- Verify that validation includes type, format, length, range, and allowed characters
- Check whether validation failures return actionable error messages without exposing internal structure
- Verify that file uploads validate MIME type, extension, and size limits, and do not trust client-provided filenames or types

## C6: Implement Digital Identity

- Check authentication mechanism: verify that passwords are hashed with a strong algorithm (bcrypt, scrypt, argon2), not MD5/SHA1/plain
- Verify that password policies (minimum length, complexity) are enforced server-side
- Check session management: verify that session IDs are generated with sufficient entropy, invalidated on logout, and have appropriate expiration
- Check JWT usage: verify that tokens are signed, have expiration (`exp`), and do not contain sensitive data in the payload
- Verify that authentication tokens are not stored in `localStorage` (use `httpOnly` cookies or memory for frontend apps)
- Check whether multi-factor authentication (MFA) is required for sensitive operations

## C7: Enforce Access Controls

- Check authorization boundaries: verify that every protected resource checks user permissions before granting access
- Verify that authorization checks cannot be bypassed by manipulating IDs, paths, or query parameters (IDOR prevention)
- Check for horizontal privilege escalation: verify that users cannot access other users' resources by changing IDs
- Check for vertical privilege escalation: verify that normal users cannot invoke admin-only endpoints
- Verify that default permissions are deny-all, not allow-all

## C8: Protect Data Everywhere

- Check whether sensitive data (passwords, tokens, credit cards, PII) is encrypted in transit (HTTPS, TLS) and at rest (database encryption, encrypted file storage)
- Verify that secrets (API keys, database passwords, signing keys) are not hard-coded in source code or config files committed to version control
- Check whether secrets are loaded from environment variables, secret management services (e.g., AWS Secrets Manager, HashiCorp Vault), or encrypted config
- Verify that sensitive data is not logged (passwords, tokens, credit card numbers, SSNs)
- Check whether API responses redact sensitive fields when returning to clients

## C9: Implement Security Logging and Monitoring

- Check whether authentication failures, authorization failures, suspicious activity, and security-critical operations (password change, permission grant, data export) are logged
- Verify that logs include sufficient context (timestamp, user ID, IP, resource, action, outcome) to investigate incidents
- Check whether sensitive data is excluded from logs (passwords, tokens, full credit card numbers)
- Verify that security events trigger alerts or monitoring dashboards for timely incident response

## C10: Handle All Errors and Exceptions Securely

- Check whether error messages expose internal details (stack traces, SQL queries, file paths, library versions) that could aid attackers
- Verify that authentication and authorization failures return generic error messages ("Invalid credentials", "Access denied") without revealing whether a user exists
- Check whether unhandled exceptions are caught at framework/middleware level and logged without exposing to users
- Verify that API error responses distinguish client errors (4xx) from server errors (5xx) but do not leak implementation details

## Additional Security Concerns

- **CORS**: Verify that CORS `Access-Control-Allow-Origin` is not set to `*` for credentialed endpoints; use explicit allowed origins
- **CSRF**: Verify that state-changing operations (POST/PUT/DELETE) require CSRF tokens or use SameSite cookie attributes
- **Rate Limiting**: Check whether authentication endpoints, password reset, and other abuse-prone endpoints have rate limiting
- **Content Security Policy (CSP)**: Verify that CSP headers are configured to prevent inline script execution and restrict resource origins
- **Secure Headers**: Check for security headers (X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security)
- **Dependency Vulnerabilities**: Verify that dependencies are up-to-date and do not have known CVEs (use `npm audit`, `go mod audit`, Snyk, Dependabot)
- **Sensitive Operations**: Verify that high-risk operations (account deletion, payment, permission change) require re-authentication or MFA
