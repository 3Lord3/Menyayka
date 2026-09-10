export const EMAIL_PATTERN = /^\S+@\S+\.\S+$/;
export const PASSWORD_MIN_LENGTH = 8;

// the pause after which typing in a field counts as finished: the debounce applies to the
// email/password fields of auth forms so the error doesn't flash on every letter
export const VALIDATE_DEBOUNCE_MS = 600;
