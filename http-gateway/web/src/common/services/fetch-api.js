import axios from 'axios'

import { security } from './security'

const CANCEL_REQUEST_DEADLINE_MS = 10000

export const fetchApi = async (url, options = {}) => {
  const { audience, scope, body, ...fetchOptions } = options
  const defaultAudience = security.getDefaultAudience()
  // Access token must be gathered and set as a Bearer header in all requests
  const accessToken = await security.getAccessTokenSilently()({
    audience: audience || defaultAudience,
    scope,
  })
  const oAuthSettings = {
    ...fetchOptions,
    headers: {
      'Content-Type': 'application/json',
      ...fetchOptions.headers,
      // Add the Authorization header to the existing headers
      Authorization: `Bearer ${accessToken}`,
    },
  }
  // Cancel token source needed for cancelling the request
  const cancelTokenSource = axios.CancelToken.source()

  // We are returning a Promise because we want to be able to cancel the request after a certain time
  return new Promise((resolve, reject) => {
    // Timer for the request cancellation
    const deadlineTimer = setTimeout(() => {
      // Cancel the request
      cancelTokenSource.cancel()
    }, CANCEL_REQUEST_DEADLINE_MS)

    axios({
      ...oAuthSettings,
      url,
      data: body,
      cancelToken: cancelTokenSource.token,
    })
      .then(response => {
        clearTimeout(deadlineTimer)
        return resolve(response)
      })
      .catch(error => {
        // A middleware for checking if the error was caused by cancellation of the request, if so, throw a DeadlineExceeded error
        if (axios.isCancel(error)) {
          return reject(new Error('DeadlineExceeded'))
        }

        // Rethrow the error
        return reject(error)
      })
  })
}
