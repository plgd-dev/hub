import React, { FC, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { messages as t } from '../LinkedHubs.i18n'
import { Props } from './LinkedHubsDetailPage.types'
import { useHubDetail } from '../../hooks'
import notificationId from '@/notificationId'
import LinkedHubsDetail from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetail'

const LinkedHubsDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultActiveTab } = props
    const { hubId } = useParams()

    const { data, loading, error, updateData } = useHubDetail(hubId!, !!hubId)

    useEffect(() => {
        if (error) {
            Notification.error({ title: _(t.linkedHubsError), message: error }, { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_ERROR })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    console.log('LinkedHubsDetailPage render')

    return <LinkedHubsDetail data={data} defaultActiveTab={defaultActiveTab} loading={loading} updateData={updateData} />
}

LinkedHubsDetailPage.displayName = 'LinkedHubsDetailPage'

export default LinkedHubsDetailPage
