# User Management and Profile Change Notes

Updated: 2026-07-07

## User Management

- The user list hides the built-in `super_admin` bootstrap account by default.
- Administrator-role users can be edited, deleted, and included in password resets through the regular user-management flow.
- The table supports pagination, keyword search, status filtering, compact row actions, and row selection.
- Batch actions now include role changes, password resets, email changes, and deletion.
- Batch email changes require one unique, valid email per selected user. The frontend validates before submit; the backend remains authoritative.
- Deletion requires a confirmation modal and no longer rejects administrator-role users.
- Single-user editing supports email, roles, and status for managed users.
- CSV import includes a downloadable template, field guidance, client-side preview of the first rows, and backend error feedback.

## Profile Center

- The profile page now uses a compact two-column layout on desktop.
- Account info stays in the left panel; email, password, passkey, and OAuth operations are grouped into tabs.
- Email change flow:
  1. Enter a new email.
  2. Send a verification code to that new email.
  3. Enter the 6-digit code.
  4. Submit the email change.
- Email-change verification codes are valid for 5 minutes.
- The frontend validates email format and code length synchronously before submitting.

## Validation Notes

- Frontend validation reduces invalid requests, but backend validation is the final enforcement point.
- Sensitive account operations rely on authenticated endpoints and backend RBAC checks.
