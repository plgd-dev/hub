import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate, useParams } from 'react-router-dom'

import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import Tabs from '@shared-ui/components/Atomic/Tabs/Tabs'

import { messages as g } from '@/containers/Global.i18n'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../ProvisioningRecords.i18n'
import { useProvisioningRecordsDetail } from '../../hooks'
import DetailHeader from '../DetailHeader/DetailHeader'
import PageLayout from '@/containers/Common/PageLayout'
import testId from '@/testId'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3'))

const ProvisioningRecordsListPage: FC<any> = (props) => {
    const { defaultActiveTab } = props

    const { formatMessage: _ } = useIntl()
    const { recordId } = useParams()
    const navigate = useNavigate()

    const { data, loading, error, refresh } = useProvisioningRecordsDetail(recordId)

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)

    const isOnline = true

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(t.provisioningRecords), link: '/device-provisioning/provisioning-records' },
            { label: data?.enrollmentGroupData?.name! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data?.enrollmentGroupData]
    )

    const getTabRoute = (i: number) => {
        switch (i) {
            case 1: {
                return '/credentials'
            }
            case 2: {
                return '/acls'
            }
            default:
            case 0: {
                return ''
            }
        }
    }

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(`/device-provisioning/provisioning-records/${recordId}${getTabRoute(i)}`, { replace: true })

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
            {!!data && (
                <Tabs
                    fullHeight
                    isAsync
                    activeItem={activeTabItem}
                    onItemChange={handleTabChange}
                    style={{
                        height: '100%',
                    }}
                    tabs={[
                        {
                            name: _(t.details),
                            id: 0,
                            dataTestId: testId.dps.provisioningRecords.detail.tabDetails,
                            content: <Tab1 data={data} />,
                        },
                        {
                            name: _(t.credentials),
                            id: 1,
                            dataTestId: testId.dps.provisioningRecords.detail.tabCredentials,
                            content: <Tab2 data={data} />,
                            status: data && data.credential ? getStatusFromCode(data.credential.status.coapCode) : undefined,
                        },
                        {
                            name: _(t.acls),
                            id: 2,
                            dataTestId: testId.dps.provisioningRecords.detail.tabAcls,
                            content: <Tab3 data={data} />,
                            status: data && data.acl ? getStatusFromCode(data.acl.status.coapCode) : undefined,
                        },
                    ]}
                />
            )}
        </PageLayout>
    )
}

ProvisioningRecordsListPage.displayName = 'ProvisioningRecordsListPage'

export default ProvisioningRecordsListPage
