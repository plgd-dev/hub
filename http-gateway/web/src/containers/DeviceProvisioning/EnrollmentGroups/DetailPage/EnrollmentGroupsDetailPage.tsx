import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import Tabs from '@shared-ui/components/Atomic/Tabs'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import testId from '@/testId'
import { Props } from './EnrollmentGroupsDetailPage.types'
import DetailHeader from '../DetailHeader'
import Tab1 from './Tabs/Tab1'
import { useEnrollmentGroupDetail, useHubDetail } from '@/containers/DeviceProvisioning/hooks'
import notificationId from '@/notificationId'

const EnrollmentGroupsDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultActiveTab } = props
    const { enrollmentId } = useParams()

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)

    const { data, loading, error } = useEnrollmentGroupDetail(enrollmentId!)
    const { data: hubData, loading: hubLoading, error: hubError } = useHubDetail(data?.hubId!, !!data?.hubId)

    useEffect(() => {
        const errorF = error || hubError

        if (errorF) {
            Notification.error(
                { title: _(t.enrollmentGroupsError), message: errorF },
                { notificationId: notificationId.HUB_DPS_ENROLLMENT_GROUP_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        // navigate(`/devices/${id}${i === 1 ? '/resources' : ''}`, { replace: true })

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(dpsT.enrollmentGroups), link: '/device-provisioning/enrollment-groups' },
            { label: enrollmentId! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={enrollmentId!} refresh={() => {}} />}
            loading={loading || hubLoading}
            title={enrollmentId}
        >
            <Tabs
                fullHeight
                activeItem={activeTabItem}
                onItemChange={handleTabChange}
                tabs={[
                    {
                        name: _(t.enrollmentConfiguration),
                        id: 0,
                        dataTestId: testId.dps.enrollmentGroups.detail.tabEnrollmentConfiguration,
                        content: <Tab1 data={data} hubData={hubData} />,
                    },
                    {
                        name: _(t.deviceCredentials),
                        id: 1,
                        dataTestId: testId.dps.enrollmentGroups.detail.tabDeviceCredentials,
                        content: <div>Tab2</div>,
                    },
                ]}
            />
        </PageLayout>
    )
}

EnrollmentGroupsDetailPage.displayName = 'EnrollmentGroupsDetailPage'

export default EnrollmentGroupsDetailPage
