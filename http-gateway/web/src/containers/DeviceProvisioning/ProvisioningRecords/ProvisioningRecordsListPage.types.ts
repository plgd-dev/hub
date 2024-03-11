export interface DataType {
    deviceId: string
    enrollmentGroupId: string
    creationDate: string
    attestation: Attestation
    acl: ACL
    cloud: Cloud
    localEndpoints: string[]
    id: string
    credential: PokedexCredential
    ownership: Ownership
    plgdTime: PlgdTime
    enrollmentGroupData: EnrollmentGroupData
}

export interface ACL {
    status: PlgdTime
    accessControlList: AccessControlList[]
}

export interface AccessControlList {
    resources: Resource[]
    deviceSubject?: DeviceSubject
    permissions: string[]
    connectionSubject?: ConnectionSubject
}

export interface ConnectionSubject {
    type: string
}

export interface DeviceSubject {
    deviceId: string
}

export interface Resource {
    interfaces: Interface[]
    wildcard: Wildcard
    href?: string
}

export enum Interface {
    Empty = '*',
}

export enum Wildcard {
    NoncfgAll = 'NONCFG_ALL',
    None = 'NONE',
}

export interface PlgdTime {
    date: string
    coapCode: number
}

export interface Attestation {
    date: string
    x509: AttestationX509
}

export interface AttestationX509 {
    certificatePem: string
    commonName: string
}

export interface Cloud {
    status: PlgdTime
    providerName: string
    gateways: Gateway[]
    selectedGateway: number
}

export interface Gateway {
    id: string
    uri: string
}

export interface PokedexCredential {
    preSharedKey: PreSharedKey
    credentials: CredentialElement[]
    status: PlgdTime
    identityCertificatePem: string
}

export interface CredentialElement {
    subject: string
    privateData?: PrivateData
    id: string
    type: string[]
    usage: string
    publicData?: PublicData
}

export interface PrivateData {
    data: string
    encoding: string
    handle: string
}

export interface PublicData {
    data: string
    encoding: string
}

export interface PreSharedKey {
    key: string
    subjectId: string
}

export interface EnrollmentGroupData {
    id: string
    owner: string
    attestationMechanism: AttestationMechanism
    hubIds: string[]
    hubsData: HubData[]
    preSharedKey: string
    name: string
}

export interface AttestationMechanism {
    x509: AttestationMechanismX509
}

export interface AttestationMechanismX509 {
    certificateChain: string
    leadCertificateName: string
    expiredCertificateEnabled: boolean
}

export interface Ownership {
    status: PlgdTime
    owner: string
}

export interface HubData {
    id: string
    gateways: string[]
    certificateAuthority: CertificateAuthority
    authorization: Authorization
    name: string
    hubId: string
}

export interface Authorization {
    provider: Provider
    ownerClaim: string
}

export interface Provider {
    authority: string
    clientId: string
    scopes: string[]
    clientSecret: string
    http: HTTP
    name: string
}

export interface HTTP {
    timeout: string
    tls: TLS
    maxIdleConns: number
    maxConnsPerHost: number
    maxIdleConnsPerHost: number
    idleConnTimeout: string
}

export interface TLS {
    cert: string
    useSystemCaPool: boolean
    caPool: string[]
    key: string
}

export interface CertificateAuthority {
    grpc: Grpc
}

export interface Grpc {
    tls: TLS
    address: string
    keepAlive: KeepAlive
}

export interface KeepAlive {
    permitWithoutStream: boolean
    time: string
    timeout: string
}
