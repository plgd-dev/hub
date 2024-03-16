export type Props = {
    defaultFormData: {
        id: string
        owner: string
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
    name: string
    hubIds: string[]
    preSharedKey: string
}
