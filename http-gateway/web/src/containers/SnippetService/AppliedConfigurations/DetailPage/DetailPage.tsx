import React, { FC, lazy, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'

import PageLayout from '@/containers/Common/PageLayout'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'
import Tabs from '@shared-ui/components/Atomic/Tabs'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import { useAppliedConfigurationDetail } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import DetailHeader from './DetailHeader'
import { cancelPendingCommandApi } from '@/containers/PendingCommands/rest'
import testId from '@/testId'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))

const DetailPage: FC<any> = () => {
    const { appliedConfigurationId, tab: tabRoute } = useParams()

    const { formatMessage: _ } = useIntl()
    const { data, loading, error, refresh } = useAppliedConfigurationDetail(appliedConfigurationId || '', !!appliedConfigurationId)

    const tab = tabRoute || ''

    const [activeTabItem, setActiveTabItem] = useState(tab ? pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.TABS.indexOf(tab) : 0)
    const [canceling, setCanceling] = useState(false)
    const [notFound, setNotFound] = useState(false)

    const navigate = useNavigate()

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.snippetService), link: generatePath(pages.SNIPPET_SERVICE.LINK) },
            { label: _(confT.appliedConfiguration), link: generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    useEffect(() => {
        if (pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.TABS.indexOf(tab) === -1 || (!data && !loading)) {
            setNotFound(true)
        }
    }, [data, loading, tab])

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(
            generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.LINK, {
                appliedConfigurationId,
                tab: pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.DETAIL.TABS[i],
            })
        )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const cancelCommand = async (resource: ResourceType) => {
        try {
            setCanceling(true)

            const { correlationId, href } = resource

            if (href && data?.deviceId) {
                await cancelPendingCommandApi({
                    deviceId: data?.deviceId || '',
                    href,
                    correlationId,
                })

                refresh()
            }

            setCanceling(false)
        } catch (error) {
            setCanceling(false)
            Notification.error(
                {
                    title: _(confT.cancelCommandError),
                    message: getApiErrorMessage(error),
                },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_DETAIL_CANCEL_COMMAND }
            )
        }
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <DetailHeader
                    conditionId={data?.conditionId?.id || _(confT.onDemand)}
                    conditionName={data?.conditionName !== -1 ? data?.conditionName : undefined}
                    configurationId={data?.configurationId?.id || ''}
                    configurationName={data?.configurationName || ''}
                    id={data?.id || ''}
                    loading={loading || canceling}
                />
            }
            loading={loading || canceling}
            notFound={notFound}
            title={data?.name}
            xPadding={false}
        >
            <Loadable condition={!loading && !!data}>
                <Tabs
                    fullHeight
                    innerPadding
                    isAsync
                    activeItem={activeTabItem}
                    onItemChange={handleTabChange}
                    style={{
                        height: '100%',
                    }}
                    tabs={[
                        {
                            name: _(g.general),
                            id: 0,
                            dataTestId: testId.snippetService.appliedConfigurations.detail.tabGeneral,
                            content: <Tab1 data={data} />,
                        },
                        {
                            name: _(confT.listOfResources),
                            id: 1,
                            dataTestId: testId.snippetService.appliedConfigurations.detail.tabListOfResources,
                            content: <Tab2 cancelCommand={cancelCommand} data={data} />,
                        },
                    ]}
                />
            </Loadable>
        </PageLayout>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
