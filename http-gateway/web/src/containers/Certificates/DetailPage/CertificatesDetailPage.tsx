import React, { FC, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import { useResizeDetector } from 'react-resize-detector'

import { buildCATranslations } from '@shared-ui/components/Organisms/CaPoolModal/utils'
import { parseCertificate } from '@shared-ui/common/services/certificates'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import CaPoolModalContent from '@shared-ui/components/Organisms/CaPoolModal/components/CaPoolModalContent'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import PageLayout from '@/containers/Common/PageLayout'
import { Props } from './CertificatesDetailPage.types'
import DetailHeader from '../DetailHeader'
import { useCertificatesDetail } from '@/containers/Certificates/hooks'
import { messages as t } from '@/containers/Certificates/Certificates.i18n'
import { messages as trans } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'

const CertificatesDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { certificateId } = useParams()

    const { data, error, loading } = useCertificatesDetail(certificateId!)
    const [parsedCert, setParsedCert] = useState<any>(undefined)
    const [pageLoading, setPageLoading] = useState<any>(undefined)

    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    useEffect(() => {
        if (data) {
            setPageLoading(true)
            parseCertificate(data.credential.certificatePem, 0)
                .then((r) => {
                    setPageLoading(false)
                    setParsedCert(r)
                })
                .catch((e) => {
                    Notification.error({ title: _(t.certificatesError), message: e }, { notificationId: notificationId.HUB_DPS_CERTIFICATES_DETAIL_PAGE_ERROR })
                })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data])

    useEffect(() => {
        error &&
            Notification.error({ title: _(t.certificatesError), message: error }, { notificationId: notificationId.HUB_DPS_CERTIFICATES_DETAIL_PAGE_ERROR })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const breadcrumbs = useMemo(
        () => [{ label: _(t.certificates), link: '/certificates' }, { label: certificateId! }],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={certificateId!} refresh={() => {}} />}
            loading={loading || pageLoading}
            notFound={!data && !loading}
            title={data?.commonName}
            xPadding={false}
        >
            <Spacer ref={ref} style={{ flex: ' 1 1 auto' }} type='pl-10'>
                <Loadable condition={!!parsedCert?.data}>
                    <CaPoolModalContent data={parsedCert?.data} i18n={buildCATranslations(_, trans, g)} maxHeight={height} />
                </Loadable>
            </Spacer>
        </PageLayout>
    )
}

CertificatesDetailPage.displayName = 'EnrollmentGroupsDetailPage'

export default CertificatesDetailPage
