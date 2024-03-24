import { useIntl } from 'react-intl'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'

export const tabRoutes = ['', 'certificate-authority', 'authorization']

export const DEFAULT_FORM_DATA = {
    certificateAuthority: {
        grpc: {
            keepAlive: {
                time: '10000000000',
                timeout: '20000000000',
            },
            tls: {
                caPool: [],
                key: '',
                cert: '',
                useSystemCaPool: true,
            },
        },
    },
    authorization: {
        provider: {
            http: {
                maxIdleConns: 16,
                maxConnsPerHost: 32,
                maxIdleConnsPerHost: 16,
                idleConnTimeout: '30000000000',
                timeout: '10000000000',
                tls: {
                    caPool: [],
                    key: '',
                    cert: '',
                    useSystemCaPool: true,
                },
            },
            scopes: [],
        },
    },
}

export function useCaI18n() {
    const { formatMessage: _ } = useIntl()

    return {
        algorithm: _(t.algorithm),
        authorityInfoAIA: _(t.authorityInfoAIA),
        authorityKeyID: _(t.authorityKeyID),
        basicConstraints: _(t.basicConstraints),
        certificateAuthority: _(t.certificateAuthority),
        certificatePolicies: _(t.certificatePolicies),
        commonName: _(t.commonName),
        country: _(t.country),
        dNSName: _(t.dNSName),
        download: _(t.download),
        embeddedSCTs: _(t.embeddedSCTs),
        exponent: _(t.exponent),
        extendedKeyUsages: _(t.extendedKeyUsages),
        fingerprints: _(t.fingerprints),
        issuerName: _(t.issuerName),
        keySize: _(t.keySize),
        keyUsages: _(t.keyUsages),
        location: _(t.location),
        logID: _(t.logID),
        menuTitle: _(t.certificates),
        method: _(t.method),
        miscellaneous: _(t.miscellaneous),
        modules: _(t.modules),
        no: _(g.no),
        notAfter: _(t.notAfter),
        notBefore: _(t.notBefore),
        organization: _(t.organization),
        policy: _(t.policy),
        publicKeyInfo: _(t.publicKeyInfo),
        purposes: _(t.purposes),
        serialNumber: _(t.serialNumber),
        signatureAlgorithm: _(t.signatureAlgorithm),
        subjectAltNames: _(t.subjectAltNames),
        subjectKeyID: _(t.subjectKeyID),
        subjectName: _(t.subjectName),
        timestamp: _(t.timestamp),
        validity: _(t.validity),
        value: _(t.value),
        version: _(g.version),
        yes: _(g.yes),
    }
}
