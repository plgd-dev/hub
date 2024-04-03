import React, { useCallback, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'

import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import { messages as g } from '@/containers/Global.i18n'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../ProvisioningRecords.i18n'
import { useProvisioningRecordsDetail } from '../../hooks'
import DetailHeader from '../DetailHeader/DetailHeader'
import PageLayout from '@/containers/Common/PageLayout'
import { pages } from '@/routes'
import DetailPage from '@/containers/DeviceProvisioning/ProvisioningRecords/DetailPage/DetailPage'
import { getProvisioningRecordStatus } from '@/containers/DeviceProvisioning/utils'

const ProvisioningRecordsListPage = () => {
    const { formatMessage: _ } = useIntl()
    const { recordId, tab: tabRoute } = useParams()
    const navigate = useNavigate()
    const tab = tabRoute || ''

    const { data, loading, error, refresh } = useProvisioningRecordsDetail(recordId)

    const isOnline = useMemo(() => data && getProvisioningRecordStatus(data), [data])

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(t.provisioningRecords), link: pages.DPS.PROVISIONING_RECORDS.LINK },
            { label: data?.enrollmentGroupData?.name! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data?.enrollmentGroupData]
    )

    const handleTabChange = useCallback((i: ItemType) => {
        navigate(generatePath(pages.DPS.PROVISIONING_RECORDS.DETAIL, { recordId, tab: pages.DPS.PROVISIONING_RECORDS.TABS[parseInt(i.id)] }))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    if (error) {
        return <div>{error}</div>
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <DetailHeader enrollmentGroupData={data?.enrollmentGroupData} enrollmentGroupId={data?.enrollmentGroupId} id={recordId} refresh={refresh} />
            }
            headlineStatusTag={<StatusTag variant={isOnline ? 'success' : 'error'}>{isOnline ? _(g.online) : _(g.offline)}</StatusTag>}
            loading={loading}
            title={data?.enrollmentGroupData?.name || '-'}
        >
            <Loadable condition={!!data && !loading}>
                <DetailPage currentTab={tab} onItemClick={handleTabChange} provisioningRecord={data} />
            </Loadable>
        </PageLayout>
    )
}

ProvisioningRecordsListPage.displayName = 'ProvisioningRecordsListPage'

export default ProvisioningRecordsListPage
