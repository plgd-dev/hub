export const toastTypes = {
  ERROR: 'error',
  SUCCESS: 'success',
  WARNING: 'warning',
  INFO: 'info',
}

// Default hide time of a toast
export const TOAST_HIDE_TIME = 5000

// Estimated hide time of a browser notification
export const BROWSER_NOTIFICATION_HIDE_TIME = 5000

// Maximum number of visible toasts
export const MAX_NUMBER_OF_VISIBLE_TOASTS = 5

// Maximum number of all toasts, including the "invisible" or "queued" toast
export const MAX_NUMBER_OF_ALL_TOASTS = 10

// Maximum number of all browser notifications
export const MAX_NUMBER_OF_BROWSER_NOTIFICATIONS = 3

// Event key for the browser notifications
export const BROWSER_NOTIFICATIONS_EVENT_KEY = 'browser-notifications'

// List of permissions available in the browser notifications
export const browserNotificationPermissions = {
  GRANTED: 'granted',
  DENIED: 'denied',
  DEFAULT: 'default',
}
