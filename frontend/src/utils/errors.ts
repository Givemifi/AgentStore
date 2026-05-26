import axios from 'axios';

// Generic messages for common HTTP status codes to avoid leaking internal details.
const STATUS_MESSAGES: Record<number, string> = {
  400: 'Invalid request. Please check your input.',
  401: 'Please sign in to continue.',
  403: 'You do not have permission to perform this action.',
  404: 'The requested resource was not found.',
  409: 'This action conflicts with the current state. Please refresh and try again.',
  422: 'The submitted data is invalid. Please check your input.',
  429: 'Too many requests. Please try again later.',
  500: 'An unexpected error occurred. Please try again.',
  502: 'Service temporarily unavailable. Please try again.',
  503: 'Service temporarily unavailable. Please try again.',
};

export function getErrorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const status = err.response?.status;
    const data = err.response?.data;

    if (data && typeof data === 'object' && 'error' in data) {
      return String(data.error);
    }

    if (typeof data === 'string' && data.trim()) {
      return data;
    }

    // For errors without a backend message, use generic status-based messages.
    if (status && STATUS_MESSAGES[status]) return STATUS_MESSAGES[status];

    if (err.message) return err.message;
  }
  if (err instanceof Error) return err.message;
  return 'An unexpected error occurred';
}
