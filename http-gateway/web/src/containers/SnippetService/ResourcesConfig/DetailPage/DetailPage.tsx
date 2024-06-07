import React, { FC, lazy, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import ReactDOM from 'react-dom'

import { getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { FormContext } from '@shared-ui/common/context/FormContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import AppContext from '@shared-ui/app/share/AppContext'
import { useFormData, useIsMounted } from '@shared-ui/common/hooks'

import PageLayout from '@/containers/Common/PageLayout'
import { pages } from '@/routes'
import { messages as confT } from '../../SnippetService.i18n'
import { useResourcesConfigApplied, useResourcesConfigConditions, useResourcesConfigDetail } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import DetailHeader from './DetailHeader'
import testId from '@/testId'
import { messages as g } from '@/containers/Global.i18n'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3'))

const DetailPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const { resourcesConfigId, tab: tabRoute } = useParams()
    const { data, loading, error, refresh } = useResourcesConfigDetail(resourcesConfigId || '', !!resourcesConfigId)
    const { data: conditionsData, loading: conditionsLoading } = useResourcesConfigConditions(resourcesConfigId || '', !!resourcesConfigId)
    const { data: appliedConfigurationData, loading: appliedConfigurationLoading } = useResourcesConfigApplied(resourcesConfigId || '', !!resourcesConfigId)
    const { collapsed } = useContext(AppContext)
    const tab = tabRoute || ''

    const defaultFormState = useMemo(
        () => ({
            tab1: false,
            tab2: false,
            tab3: false,
        }),
        []
    )

    console.log({ conditionsData })
    console.log({ appliedConfigurationData })

    const { handleReset, context, resetIndex, dirty, formData, hasError } = useFormData({
        defaultFormState,
        data,
        i18n: { promptDefaultMessage: _(g.promptDefaultMessage), default: _(g.default) },
    })

    const [pageLoading, setPageLoading] = useState(false)
    const [notFound, setNotFound] = useState(false)
    const [activeTabItem, setActiveTabItem] = useState(tab ? pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.TABS.indexOf(tab) : 0)

    const isMounted = useIsMounted()
    const navigate = useNavigate()

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(confT.resourcesConfigurationError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    useEffect(() => {
        if (pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.TABS.indexOf(tab) === -1) {
            setNotFound(true)
        }
    }, [tab])

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.conditions), link: generatePath(pages.CONDITIONS.LINK) },
            { label: _(confT.resourcesConfiguration), link: generatePath(pages.CONDITIONS.RESOURCES_CONFIG.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(generatePath(pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.LINK, { resourcesConfigId, tab: pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.TABS[i] }))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const onSubmit = async () => {
        setPageLoading(true)

        try {
            // DATA FOR SAVE

            Notification.success(
                { title: _(confT.resourcesConfigurationUpdated), message: _(confT.resourcesConfigurationUpdatedMessage) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_DETAIL_PAGE_UPDATE_SUCCESS }
            )

            handleReset()
            refresh()

            setPageLoading(false)
        } catch (error: any) {
            let e = error
            if (!(error instanceof Error)) {
                e = new Error(error)
            }
            Notification.error(
                { title: _(confT.resourcesConfigurationUpdateError), message: e.message },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_DETAIL_PAGE_UPDATE_ERROR }
            )
            setPageLoading(false)
        }
    }

    const loadingState = useMemo(
        () => loading || conditionsLoading || appliedConfigurationLoading || pageLoading,
        [loading, conditionsLoading, appliedConfigurationLoading, pageLoading]
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={resourcesConfigId!} loading={loadingState} refresh={refresh} />}
            loading={loadingState}
            notFound={notFound}
            title={data?.name}
            xPadding={false}
        >
            <FormContext.Provider value={context}>
                <Loadable condition={!!formData}>
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
                                name: _(confT.generalAndResources),
                                id: 0,
                                dataTestId: testId.snippetService.resourcesConfig.detail.tabGeneral,
                                content: (
                                    <Tab1
                                        defaultFormData={formData}
                                        isActiveTab={activeTabItem === 0}
                                        loading={loading || pageLoading}
                                        resetIndex={resetIndex}
                                    />
                                ),
                            },
                            {
                                name: _(confT.conditions),
                                id: 1,
                                dataTestId: testId.snippetService.resourcesConfig.detail.tabConditions,
                                content: <Tab2 data={conditionsData} isActiveTab={activeTabItem === 1} loading={conditionsLoading} />,
                            },
                            {
                                name: _(confT.appliedDeviceConfiguration),
                                id: 2,
                                dataTestId: testId.snippetService.resourcesConfig.detail.tabDeviceAppliedConfiguration,
                                content: <Tab3 />,
                            },
                        ]}
                    />
                </Loadable>
            </FormContext.Provider>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button disabled={hasError} loading={loading} loadingText={_(g.loading)} onClick={onSubmit} variant='primary'>
                                {_(g.saveChanges)}
                            </Button>
                        }
                        actionSecondary={
                            <Button disabled={loading} onClick={handleReset} variant='secondary'>
                                {_(g.reset)}
                            </Button>
                        }
                        leftPanelCollapsed={collapsed}
                        show={dirty}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </PageLayout>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
