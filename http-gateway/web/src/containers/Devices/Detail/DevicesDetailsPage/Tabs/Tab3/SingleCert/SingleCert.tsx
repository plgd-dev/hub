import React, { FC, useEffect, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import { useIntl } from 'react-intl'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import CaPoolModalContent from '@shared-ui/components/Organisms/CaPoolModal/components/CaPoolModalContent'
import { buildCATranslations } from '@shared-ui/components/Organisms/CaPoolModal/utils'
import { parseCertificate } from '@shared-ui/common/services/certificates'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Button from '@shared-ui/components/Atomic/Button'
import { IconTrash } from '@shared-ui/components/Atomic/Icon'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { messages as trans } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import notificationId from '@/notificationId'
import { deleteCertificatesApi } from '@/containers/Certificates/rest'
import { Props } from './SingleCert.types'

const SingleCert: FC<Props> = (props) => {
    const { certificate, loading, handleTabChange, refresh } = props

    const { formatMessage: _ } = useIntl()

    const [certData, setCertData] = useState<any>(undefined)
    const [deleting, setDeleting] = useState<boolean>(false)

    const { ref, height } = useResizeDetector({
        refreshRate: 500,
    })

    useEffect(() => {
        const parseData = async (cert: any) => {
            try {
                return await parseCertificate(cert, 0)
            } catch (e: any) {
                let error = e
                if (!(error instanceof Error)) {
                    error = new Error(e)
                }

                Notification.error(
                    { title: _(t.certificationParsingError), message: error.message },
                    { notificationId: notificationId.HUB_DPS_CERTIFICATES_LIST_CERT_PARSE_ERROR }
                )
            }
        }

        if (certificate && certificate.credential.certificatePem) {
            parseData(certificate?.credential.certificatePem).then((d) => setCertData(d))
        }

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [certificate])

    const handleDelete = async () => {
        try {
            setDeleting(true)
            await deleteCertificatesApi([certificate.id])

            Notification.success(
                { title: _(t.certificateDeleted), message: _(t.certificateDeletedMessage) },
                { notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB3_DELETE_SUCCESS }
            )

            refresh()
            setDeleting(false)
            handleTabChange(0)
        } catch (e: any) {
            setDeleting(false)

            Notification.error(
                { title: _(t.certificatesDeleteError), message: getApiErrorMessage(e) },
                { notificationId: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB3_DELETE_ERROR }
            )
        }
    }

    return (
        <Spacer style={{ flex: ' 1 1 auto', height: '100%', display: 'flex', flexDirection: 'column' }} type='pl-10'>
            <Spacer style={{ display: 'flex', justifyContent: 'flex-end' }} type='mb-4 pr-10'>
                <Button disabled={deleting || loading} icon={<IconTrash />} onClick={handleDelete} variant='tertiary'>
                    {_(t.deleteCertificate)}
                </Button>
            </Spacer>
            <div ref={ref} style={{ flex: ' 1 1 auto' }}>
                <Loadable condition={!!certData}>
                    <CaPoolModalContent data={certData?.data || []} i18n={buildCATranslations(_, trans, g)} maxHeight={height} />
                </Loadable>
            </div>
        </Spacer>
    )
}

SingleCert.displayName = 'SingleCert'

export default SingleCert
