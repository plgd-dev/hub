let getAccessTokenSilently = null
let generalConfig = {}
let deviceOAuthConfig = {}
let webOAuthConfig = {}

// This singleton contains the method getAccessTokenSilently exposed globally, so that we can use this in our interceptors.
export const security = {
  getAccessTokenSilently: () => getAccessTokenSilently,
  setAccessTokenSilently: func => (getAccessTokenSilently = func),
  getGeneralConfig: () => generalConfig,
  setGeneralConfig: config => (generalConfig = config),
  getWebOAuthConfig: () => webOAuthConfig,
  setWebOAuthConfig: config => (webOAuthConfig = config),
  getDeviceOAuthConfig: () => deviceOAuthConfig,
  setDeviceOAuthConfig: config => (deviceOAuthConfig = config),
}
