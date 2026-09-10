// we don't mask the domain — otherwise it's unclear which mailbox to check; a string without @
// is returned as-is, the helper mustn't crash on an invalid value
export function maskEmail(email: string): string {
  const atIndex = email.indexOf('@');
  if (atIndex === -1) return email;
  const local = email.slice(0, atIndex);
  const domain = email.slice(atIndex);
  // Math.max guards an empty local part ('@example.com') from '*'.repeat(-1)
  if (local.length <= 2) {
    return `${local.charAt(0)}${'*'.repeat(Math.max(0, local.length - 1))}${domain}`;
  }
  return `${local.slice(0, 2)}${'*'.repeat(local.length - 2)}${domain}`;
}
