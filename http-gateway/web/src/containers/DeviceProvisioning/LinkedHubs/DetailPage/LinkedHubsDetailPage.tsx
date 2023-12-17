import React, { FC, useCallback, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import Tabs from '@shared-ui/components/Atomic/Tabs'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import testId from '@/testId'
import { Props } from './LinkedHubsDetailPage.types'
import DetailHeader from '../DetailHeader'

const LinkedHubsDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultActiveTab } = props
    const { hubId } = useParams()

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)

    console.log(hubId)

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        // navigate(`/devices/${id}${i === 1 ? '/resources' : ''}`, { replace: true })

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(t.linkedHubs), link: '/device-provisioning/linked-hubs' },
            { label: hubId! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <PageLayout breadcrumbs={breadcrumbs} header={<DetailHeader id={hubId!} refresh={() => {}} />} loading={false} title={hubId}>
            <Tabs
                activeItem={activeTabItem}
                fullHeight={true}
                onItemChange={handleTabChange}
                tabs={[
                    {
                        name: _(t.certificateAuthorityConfiguration),
                        id: 0,
                        dataTestId: testId.dps.linkedHubs.detail.tabCertificateAuthorityConfiguration,
                        content: <div>Tab1</div>,
                    },
                    {
                        name: _(t.authorization),
                        id: 1,
                        dataTestId: testId.dps.linkedHubs.detail.tabAuthorization,
                        content: <div>Tab2</div>,
                    },
                ]}
            />
        </PageLayout>
    )
}

LinkedHubsDetailPage.displayName = 'LinkedHubsDetailPage'

export default LinkedHubsDetailPage
