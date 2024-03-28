import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { parseCertificate } from '@shared-ui/common/services/certificates'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as t } from '../Certificates.i18n'
import notificationId from '@/notificationId'
import { useCertificatesList } from '@/containers/Certificates/hooks'
import { deleteCertificatesApi } from '@/containers/Certificates/rest'
import { pages } from '@/routes'
import CertificatesList from './CertificatesList'

const CertificatesListPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const { data, error, loading, refresh } = useCertificatesList()

    const [displayData, setDisplayData] = useState<any>(undefined)

    useEffect(() => {
        const parseCerts = async (certs: any) => {
            const parsed = certs?.map(async (certsData: { credential: { certificatePem: string } }, key: number) => {
                try {
                    return await parseCertificate(certsData?.credential.certificatePem, key, certsData)
                } catch (e: any) {
                    let error = e
                    if (!(error instanceof Error)) {
                        error = new Error(e)
                    }

                    console.error(error)
                    Notification.error(
                        { title: _(t.certificationParsingError), message: error.message },
                        { notificationId: notificationId.HUB_DPS_CERTIFICATES_LIST_CERT_PARSE_ERROR }
                    )
                }
            })

            return await Promise.all(parsed)
        }

        if (data) {
            parseCerts(data).then((d) => {
                setDisplayData(d)
            })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data])

    const navigate = useNavigate()

    const [deleting, setDeleting] = useState(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
    const breadcrumbs = useMemo(() => [{ label: _(t.certificate) }], [])

    useEffect(() => {
        error && Notification.error({ title: _(t.certificatesError), message: error }, { notificationId: notificationId.HUB_DPS_CERTIFICATES_LIST_PAGE_ERROR })
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loading || deleting || !displayData} title={_(t.certificates)}>
            <CertificatesList
                data={data}
                deleting={deleting}
                loading={loading}
                notificationIds={{
                    deleteError: notificationId.HUB_DPS_CERTIFICATES_LIST_DELETE_ERROR,
                    deleteSuccess: notificationId.HUB_DPS_CERTIFICATES_LIST_DELETE_SUCCESS,
                    parsingError: notificationId.HUB_DPS_CERTIFICATES_LIST_CERT_PARSE_ERROR,
                }}
                onDelete={deleteCertificatesApi}
                onView={(id) => navigate(generatePath(pages.CERTIFICATES.DETAIL, { certificateId: id }))}
                refresh={refresh}
                setDeleting={setDeleting}
            />
        </PageLayout>
    )
}

CertificatesListPage.displayName = 'CertificatesListPage'

export default CertificatesListPage
