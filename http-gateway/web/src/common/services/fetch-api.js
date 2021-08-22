import axios from 'axios'
import { time } from 'units-converter'

import { security } from './security'

const CANCEL_REQUEST_DEADLINE_MS = 10000
const COMMAND_TIMEOUT_THRESHOLD_MS = 500

export const errorCodes = {
  COMMAND_EXPIRED: 'CommandExpired',
  DEADLINE_EXCEEDED: 'DeadlineExceeded',
  INVALID_ARGUMENT: 'InvalidArgument',
}

export const fetchApi = async (url, options = {}) => {
  const { audience, scope, body, timeToLive, ...fetchOptions } = options
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
  const cancelDeadlineMs = timeToLive
    ? time(timeToLive)
        .from('ns')
        .to('ms').value + COMMAND_TIMEOUT_THRESHOLD_MS
    : CANCEL_REQUEST_DEADLINE_MS

  // We are returning a Promise because we want to be able to cancel the request after a certain time
  return new Promise((resolve, reject) => {
    // Timer for the request cancellation
    const deadlineTimer = setTimeout(() => {
      // Cancel the request
      cancelTokenSource.cancel()
    }, cancelDeadlineMs)

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
        clearTimeout(deadlineTimer)

        // A middleware for checking if the error was caused by cancellation of the request, if so, throw a DeadlineExceeded error
        if (axios.isCancel(error)) {
          const cancelError =
            cancelDeadlineMs < CANCEL_REQUEST_DEADLINE_MS
              ? errorCodes.COMMAND_EXPIRED
              : errorCodes.DEADLINE_EXCEEDED
          return reject(new Error(cancelError))
        }

        // Rethrow the error
        return reject(error)
      })
  })
}
