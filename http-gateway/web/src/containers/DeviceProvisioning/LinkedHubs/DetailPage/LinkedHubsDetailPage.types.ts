export type TLSType = {
    caPool: string[]
    cert: string
    key: string
    useSystemCaPool: boolean
}

export type GRPCData = {
    address: string
    keepAlive: {
        time: string
        timeout: string
        permitWithoutStream: boolean
    }
    tls: TLSType
}

export type AuthorizationDataType = {
    ownerClaim: string
    provider: {
        authority: string
        clientId: string
        clientSecret: string
        name: string
        scopes: string[]
        http: {
            idleConnTimeout: string
            maxConnsPerHost: number
            maxIdleConns: number
            maxIdleConnsPerHost: number
            timeout: string
            tls: TLSType
        }
    }
}

export type HubDataType = {
    id: string
    certificateAuthority: {
        grpc: GRPCData
    }
    authorization: AuthorizationDataType
    gateways: { value: string; id?: string }[]
    name: string
}

export type Props = {
    defaultActiveTab?: number
}

export type Inputs = {
    name: string
    coapGateway: string
    certificateAuthority: {
        grpc: GRPCData
    }
    authorization: AuthorizationDataType
}
