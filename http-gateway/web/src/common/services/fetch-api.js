import axios from 'axios'

import { security } from './security'

export const fetchApi = async (url, options = {}) => {
  const { audience, scope, body, ...fetchOptions } = options
  const defaultAudience = security.getDefaultAudience()
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

  return axios({
    ...oAuthSettings,
    url,
    data: body,
  })
}
