let getAccessTokenSilently = null
let defaultAudience = null

// This singleton contains the method getAccessTokenSilently exposed globally, so that we can use this in our interceptors.
export const security = {
  getAccessTokenSilently: () => getAccessTokenSilently,
  setAccessTokenSilently: func => (getAccessTokenSilently = func),
  getDefaultAudience: () => defaultAudience,
  setDefaultAudience: audience => (defaultAudience = audience),
}
