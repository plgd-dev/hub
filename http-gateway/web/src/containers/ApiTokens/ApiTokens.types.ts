export type OpenidConfigurationReturnType = {
    issuer: string
    jwks_uri: string
    plgd_tokens_endpoint: string
    token_endpoint: string
}

export type CreateTokenReturnType = {
    accessToken: string
    scope: string
    token_type: string
}
