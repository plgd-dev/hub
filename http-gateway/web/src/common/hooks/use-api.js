import { useEffect, useState } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
import axios from 'axios'

export const useApi = (url, options = {}) => {
  const { getAccessTokenSilently } = useAuth0()
  const [state, setState] = useState({
    error: null,
    loading: true,
    data: null,
  })
  const [refreshIndex, setRefreshIndex] = useState(0)

  useEffect(
    () => {
      ;(async () => {
        try {
          const { audience, scope, ...fetchOptions } = options
          const accessToken = await getAccessTokenSilently({ audience, scope })
          const oAuthSettings = {
            ...fetchOptions,
            headers: {
              'Content-Type': 'application/json',
              ...fetchOptions.headers,
              // Add the Authorization header to the existing headers
              Authorization: `Bearer ${accessToken}`,
            },
          }

          const { data } = await axios({
            ...oAuthSettings,
            url,
          })

          setState({
            ...state,
            data,
            error: null,
            loading: false,
          })
        } catch (error) {
          setState({
            ...state,
            data: null,
            error,
            loading: false,
          })
        }
      })()
    },
    [refreshIndex] // eslint-disable-line
  )

  return {
    ...state,
    refresh: () => setRefreshIndex(refreshIndex + 1),
  }
}
