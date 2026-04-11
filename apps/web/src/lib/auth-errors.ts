/**
 * Maps API error messages to user-friendly messages
 */
export function getAuthErrorMessage(error: string): string {
  // Normalize error string
  const normalizedError = error.toLowerCase().trim();

  // Network errors
  if (normalizedError.includes('network') || normalizedError.includes('fetch')) {
    return 'Unable to connect to the server. Please check your internet connection and try again.';
  }

  if (normalizedError.includes('timeout')) {
    return 'The request timed out. Please try again.';
  }

  // Auth errors from backend
  if (normalizedError.includes('invalid email or password') || normalizedError.includes('invalid credentials')) {
    return 'Invalid email or password. Please check your credentials and try again.';
  }

  if (normalizedError.includes('email already registered')) {
    return 'This email is already registered. Try signing in instead, or use a different email.';
  }

  if (normalizedError.includes('invalid email format')) {
    return 'Please enter a valid email address.';
  }

  if (normalizedError.includes('email, password and name are required')) {
    return 'Please fill in all required fields.';
  }

  if (normalizedError.includes('email and password are required')) {
    return 'Please enter your email and password.';
  }

  // Password errors
  if (normalizedError.includes('password must be at least 12 characters')) {
    return 'Password must be at least 12 characters with uppercase, lowercase, and a number.';
  }

  if (normalizedError.includes('weak password')) {
    return 'Please choose a stronger password with at least 12 characters, including uppercase, lowercase, and numbers.';
  }

  // OAuth errors
  if (normalizedError.includes('oauth not configured')) {
    return 'Google sign-in is not available at the moment. Please use email and password.';
  }

  // Server errors
  if (normalizedError.includes('internal server error') || normalizedError.includes('500')) {
    return 'Something went wrong on our end. Please try again in a few moments.';
  }

  // Unauthorized
  if (normalizedError.includes('unauthorized')) {
    return 'Your session has expired. Please sign in again.';
  }

  // User not found
  if (normalizedError.includes('user not found')) {
    return 'Account not found. Please check your email or sign up for a new account.';
  }

  // Rate limiting
  if (normalizedError.includes('rate limit') || normalizedError.includes('too many requests')) {
    return 'Too many attempts. Please wait a moment before trying again.';
  }

  // Default fallback
  return error || 'An unexpected error occurred. Please try again.';
}

/**
 * Maps OAuth error codes from URL params to friendly messages
 */
export function getOAuthErrorMessage(errorCode: string): string {
  switch (errorCode) {
    case 'invalid_state':
      return 'Authentication session expired. Please try signing in again.';
    case 'access_denied':
      return 'Access was denied. You need to grant permission to sign in with Google.';
    case 'no_code':
      return 'Authentication failed. No authorization code received.';
    case 'auth_failed':
      return 'Google authentication failed. Please try again or use email sign-in.';
    case 'server_error':
      return 'Google encountered an error. Please try again later.';
    case 'temporarily_unavailable':
      return 'Google sign-in is temporarily unavailable. Please try again later.';
    default:
      return `Authentication error: ${errorCode}. Please try again.`;
  }
}

/**
 * Validates password strength and returns feedback
 */
export interface PasswordStrength {
  score: number; // 0-4
  label: 'weak' | 'fair' | 'good' | 'strong';
  color: string;
  feedback: string[];
}

export function checkPasswordStrength(password: string): PasswordStrength {
  const feedback: string[] = [];
  let score = 0;

  if (password.length === 0) {
    return { score: 0, label: 'weak', color: 'gray', feedback: [] };
  }

  // Length check
  if (password.length >= 12) {
    score += 1;
  } else {
    feedback.push(`${12 - password.length} more characters needed`);
  }

  // Uppercase check
  if (/[A-Z]/.test(password)) {
    score += 1;
  } else {
    feedback.push('Add an uppercase letter');
  }

  // Lowercase check
  if (/[a-z]/.test(password)) {
    score += 1;
  } else {
    feedback.push('Add a lowercase letter');
  }

  // Number check
  if (/[0-9]/.test(password)) {
    score += 1;
  } else {
    feedback.push('Add a number');
  }

  // Determine label and color
  let label: PasswordStrength['label'];
  let color: string;

  if (score <= 1) {
    label = 'weak';
    color = '#ef4444'; // red
  } else if (score === 2) {
    label = 'fair';
    color = '#f97316'; // orange
  } else if (score === 3) {
    label = 'good';
    color = '#eab308'; // yellow
  } else {
    label = 'strong';
    color = '#22c55e'; // green
  }

  return { score, label, color, feedback };
}

/**
 * Validates email format
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}
