export type Props = {
    defaultFormData: {
        id: string
        hubsData: {
            authorization: any
            certificateAuthority: any
            coapGateway: string
            id: string
            name: string
        }[]
    } & Inputs
    resetIndex?: number
}

export type Inputs = {
    attestationMechanism: {
        x509: {
            certificateChain: string
            expiredCertificateEnabled: boolean
            leadCertificateName: string
        }
    }
    owner: string
    name: string
    hubIds: string[]
    preSharedKey: string
}
