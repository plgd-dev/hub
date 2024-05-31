import React, { FC, useState } from 'react'
import { useIntl } from 'react-intl'

import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'

import notificationId from '@/notificationId'
import { deleteCertificatesApi } from '@/containers/Certificates/rest'
import CertificatesList from '@/containers/Certificates/ListPage/CertificatesList'
import { Props } from './MultiCerts.types'
import { messages as t } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecords.i18n'
import { useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'

const MultiCerts: FC<Props> = (props) => {
    const { certificates, loading, refresh } = props

    const { formatMessage: _ } = useIntl()
    const i18n = useCaI18n()

    const [deleting, setDeleting] = useState(false)

    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: _(t.certificateDetail),
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    return (
        <>
            <CertificatesList
                data={certificates}
                deleting={deleting}
                loading={loading}
                notificationIds={{
                    deleteError: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB3_DELETE_ERROR,
                    deleteSuccess: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB3_DELETE_SUCCESS,
                    parsingError: notificationId.HUB_DEVICES_DETAILS_PAGE_TAB3_PARSING_ERROR,
                }}
                onDelete={deleteCertificatesApi}
                onView={(_id, certData) => {
                    setCaModalData({
                        title: _(t.certificateDetail),
                        subTitle: certData.name,
                        data: certData.data || certData.name,
                        dataChain: certData.dataChain,
                    })
                }}
                refresh={refresh}
                setDeleting={setDeleting}
            />

            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18n}
                onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
                title={caModalData.title}
            />
        </>
    )
}

MultiCerts.displayName = 'MultiCerts'

export default MultiCerts
