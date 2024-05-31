export type CertificatesType = {
    id: string
    owner: string
    commonName: string
    publicKey: string
    creationDate: string
    credential: Credential
}

export type Credential = {
    date: string
    certificatePem: string
    validUntilDate: string
}

export type ParsedCertificatesType = {
    id: string
    name: string
    dataChain: string
    data: {}[]
} & CertificatesType
