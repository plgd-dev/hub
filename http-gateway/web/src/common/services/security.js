let generalConfig = {}
let deviceOAuthConfig = {}
let webOAuthConfig = {}
let getAccessToken = null
let userManager = null

// This singleton contains the method getAccessTokenSilently exposed globally, so that we can use this in our interceptors.
export const security = {
  getAccessToken: () => getAccessToken,
  setAccessToken: func => (getAccessToken = func),
  getGeneralConfig: () => generalConfig,
  setGeneralConfig: config => (generalConfig = config),
  getWebOAuthConfig: () => webOAuthConfig,
  setWebOAuthConfig: config => (webOAuthConfig = config),
  getDeviceOAuthConfig: () => deviceOAuthConfig,
  setDeviceOAuthConfig: config => (deviceOAuthConfig = config),
  getUserManager: () => userManager,
  setUserManager: data => (userManager = data),
}
