export type Props = {
    defaultFormData: any
}

export type Inputs = {
    hubName: string
    endpoint: string
    certificate: string
    certificateName?: string
    certificateAuthorityAddress?: string
    caPool?: string
    clientCertificate?: string
    clientCertificatePrivateKey?: string
    keepAliveTime?: number
    keepAliveTimeout?: number
}
