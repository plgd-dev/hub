// Return a message from the error response of an API
export const getApiErrorMessage = error =>
  error?.response?.data?.err || error?.response?.data?.message || error?.message
