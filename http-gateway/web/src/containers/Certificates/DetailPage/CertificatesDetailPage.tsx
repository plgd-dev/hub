import React, { FC, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as dpsT } from '../../DeviceProvisioning/DeviceProvisioning.i18n'
import { Props } from './CertificatesDetailPage.types'
import DetailHeader from '../DetailHeader'

const CertificatesDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { certificateId } = useParams()

    console.log(certificateId)

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(dpsT.enrollmentGroups), link: '/device-provisioning/certificates' },
            { label: certificateId! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} header={<DetailHeader id={certificateId!} refresh={() => {}} />} loading={false} title={certificateId}>
            Content
        </PageLayout>
    )
}

CertificatesDetailPage.displayName = 'EnrollmentGroupsDetailPage'

export default CertificatesDetailPage
