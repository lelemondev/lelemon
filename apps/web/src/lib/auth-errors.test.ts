import { describe, it, expect } from 'vitest';
import {
  getAuthErrorMessage,
  getOAuthErrorMessage,
  checkPasswordStrength,
  isValidEmail,
} from './auth-errors';

describe('auth-errors', () => {
  describe('getAuthErrorMessage', () => {
    it('handles network errors', () => {
      expect(getAuthErrorMessage('network error')).toBe(
        'Unable to connect to the server. Please check your internet connection and try again.'
      );
      expect(getAuthErrorMessage('fetch failed')).toBe(
        'Unable to connect to the server. Please check your internet connection and try again.'
      );
    });

    it('handles timeout errors', () => {
      expect(getAuthErrorMessage('Request timeout')).toBe(
        'The request timed out. Please try again.'
      );
    });

    it('handles invalid credentials', () => {
      expect(getAuthErrorMessage('invalid email or password')).toBe(
        'Invalid email or password. Please check your credentials and try again.'
      );
      expect(getAuthErrorMessage('Invalid credentials')).toBe(
        'Invalid email or password. Please check your credentials and try again.'
      );
    });

    it('handles duplicate email', () => {
      expect(getAuthErrorMessage('email already registered')).toBe(
        'This email is already registered. Try signing in instead, or use a different email.'
      );
    });

    it('handles invalid email format', () => {
      expect(getAuthErrorMessage('invalid email format')).toBe(
        'Please enter a valid email address.'
      );
    });

    it('handles missing required fields', () => {
      expect(getAuthErrorMessage('email, password and name are required')).toBe(
        'Please fill in all required fields.'
      );
      expect(getAuthErrorMessage('email and password are required')).toBe(
        'Please enter your email and password.'
      );
    });

    it('handles weak password errors', () => {
      expect(getAuthErrorMessage('password must be at least 12 characters')).toBe(
        'Password must be at least 12 characters with uppercase, lowercase, and a number.'
      );
      expect(getAuthErrorMessage('weak password')).toBe(
        'Please choose a stronger password with at least 12 characters, including uppercase, lowercase, and numbers.'
      );
    });

    it('handles OAuth not configured', () => {
      expect(getAuthErrorMessage('oauth not configured')).toBe(
        'Google sign-in is not available at the moment. Please use email and password.'
      );
    });

    it('handles server errors', () => {
      expect(getAuthErrorMessage('internal server error')).toBe(
        'Something went wrong on our end. Please try again in a few moments.'
      );
      expect(getAuthErrorMessage('Error 500')).toBe(
        'Something went wrong on our end. Please try again in a few moments.'
      );
    });

    it('handles unauthorized errors', () => {
      expect(getAuthErrorMessage('unauthorized')).toBe(
        'Your session has expired. Please sign in again.'
      );
    });

    it('handles user not found', () => {
      expect(getAuthErrorMessage('user not found')).toBe(
        'Account not found. Please check your email or sign up for a new account.'
      );
    });

    it('handles rate limiting', () => {
      expect(getAuthErrorMessage('rate limit exceeded')).toBe(
        'Too many attempts. Please wait a moment before trying again.'
      );
      expect(getAuthErrorMessage('too many requests')).toBe(
        'Too many attempts. Please wait a moment before trying again.'
      );
    });

    it('returns original error as fallback', () => {
      expect(getAuthErrorMessage('Some custom error')).toBe('Some custom error');
    });

    it('returns default message for empty error', () => {
      expect(getAuthErrorMessage('')).toBe('An unexpected error occurred. Please try again.');
    });

    it('is case insensitive', () => {
      expect(getAuthErrorMessage('INVALID EMAIL OR PASSWORD')).toBe(
        'Invalid email or password. Please check your credentials and try again.'
      );
    });
  });

  describe('getOAuthErrorMessage', () => {
    it('handles invalid_state', () => {
      expect(getOAuthErrorMessage('invalid_state')).toBe(
        'Authentication session expired. Please try signing in again.'
      );
    });

    it('handles access_denied', () => {
      expect(getOAuthErrorMessage('access_denied')).toBe(
        'Access was denied. You need to grant permission to sign in with Google.'
      );
    });

    it('handles no_code', () => {
      expect(getOAuthErrorMessage('no_code')).toBe(
        'Authentication failed. No authorization code received.'
      );
    });

    it('handles auth_failed', () => {
      expect(getOAuthErrorMessage('auth_failed')).toBe(
        'Google authentication failed. Please try again or use email sign-in.'
      );
    });

    it('handles server_error', () => {
      expect(getOAuthErrorMessage('server_error')).toBe(
        'Google encountered an error. Please try again later.'
      );
    });

    it('handles temporarily_unavailable', () => {
      expect(getOAuthErrorMessage('temporarily_unavailable')).toBe(
        'Google sign-in is temporarily unavailable. Please try again later.'
      );
    });

    it('handles unknown errors with code', () => {
      expect(getOAuthErrorMessage('unknown_error')).toBe(
        'Authentication error: unknown_error. Please try again.'
      );
    });
  });

  describe('checkPasswordStrength', () => {
    it('returns weak for empty password', () => {
      const result = checkPasswordStrength('');
      expect(result.score).toBe(0);
      expect(result.label).toBe('weak');
      expect(result.feedback).toEqual([]);
    });

    it('returns weak for short password', () => {
      const result = checkPasswordStrength('abc');
      expect(result.score).toBeLessThanOrEqual(1);
      expect(result.label).toBe('weak');
      expect(result.feedback).toContain('9 more characters needed');
    });

    it('requires uppercase letter', () => {
      const result = checkPasswordStrength('abcdefghijkl1');
      expect(result.feedback).toContain('Add an uppercase letter');
    });

    it('requires lowercase letter', () => {
      const result = checkPasswordStrength('ABCDEFGHIJKL1');
      expect(result.feedback).toContain('Add a lowercase letter');
    });

    it('requires number', () => {
      const result = checkPasswordStrength('Abcdefghijkl');
      expect(result.feedback).toContain('Add a number');
    });

    it('returns strong for password meeting all requirements', () => {
      const result = checkPasswordStrength('SecurePass123');
      expect(result.score).toBe(4);
      expect(result.label).toBe('strong');
      expect(result.color).toBe('#22c55e');
      expect(result.feedback).toEqual([]);
    });

    it('returns fair for score 2', () => {
      // Only length + lowercase (missing uppercase and number)
      const result = checkPasswordStrength('abcdefghijkl');
      expect(result.score).toBe(2);
      expect(result.label).toBe('fair');
      expect(result.color).toBe('#f97316');
    });

    it('returns good for score 3', () => {
      // Missing number only
      const result = checkPasswordStrength('Abcdefghijkl');
      expect(result.score).toBe(3);
      expect(result.label).toBe('good');
      expect(result.color).toBe('#eab308');
    });

    it('counts characters needed correctly', () => {
      const result = checkPasswordStrength('Ab1');
      expect(result.feedback).toContain('9 more characters needed');

      const result2 = checkPasswordStrength('Ab1234567');
      expect(result2.feedback).toContain('3 more characters needed');
    });
  });

  describe('isValidEmail', () => {
    it('validates correct emails', () => {
      expect(isValidEmail('test@example.com')).toBe(true);
      expect(isValidEmail('user.name@domain.co')).toBe(true);
      expect(isValidEmail('user+tag@example.org')).toBe(true);
      expect(isValidEmail('a@b.io')).toBe(true);
    });

    it('rejects invalid emails', () => {
      expect(isValidEmail('')).toBe(false);
      expect(isValidEmail('notanemail')).toBe(false);
      expect(isValidEmail('missing@domain')).toBe(false);
      expect(isValidEmail('@nodomain.com')).toBe(false);
      expect(isValidEmail('no spaces@test.com')).toBe(false);
      expect(isValidEmail('test@')).toBe(false);
    });
  });
});
