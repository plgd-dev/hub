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
