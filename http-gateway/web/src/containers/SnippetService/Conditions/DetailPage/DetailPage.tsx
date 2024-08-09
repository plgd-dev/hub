import React, { FC, lazy, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import ReactDOM from 'react-dom'
import cloneDeep from 'lodash/cloneDeep'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { useFormData, useIsMounted } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import AppContext from '@shared-ui/app/share/AppContext'
import { useVersion } from '@shared-ui/common/hooks/use-version'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import Tabs from '@shared-ui/components/Atomic/Tabs'

import { useConditionsDetail } from '@/containers/SnippetService/hooks'
import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import notificationId from '@/notificationId'
import DetailHeader from './DetailHeader'
import { messages as g } from '@/containers/Global.i18n'
import { updateConditionApi } from '@/containers/SnippetService/rest'
import { dirtyFormState } from '@/store/recoil.store'
import testId from '@/testId'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3'))

const DetailPage: FC<any> = () => {
    const { conditionId, tab: tabRoute } = useParams()
    const { formatMessage: _ } = useIntl()
    const { data: conditionData, loading, error, refresh } = useConditionsDetail(conditionId || '', !!conditionId)

    const tab = tabRoute || ''

    const { Selector, data, setSearchParams } = useVersion({
        i18n: { version: _(g.version), selectVersion: _(confT.selectVersion) },
        versionData: conditionData,
        refresh,
        dataTestId: testId.snippetService.conditions.detail.versionSelector,
    })

    const [pageLoading, setPageLoading] = useState(false)
    const [filterError, setFilterError] = useState(false)
    const [notFound, setNotFound] = useState(false)
    const [activeTabItem, setActiveTabItem] = useState(tab ? pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.TABS.indexOf(tab) : 0)

    const { collapsed } = useContext(AppContext)
    const isMounted = useIsMounted()
    const navigate = useNavigate()

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.snippetService), link: generatePath(pages.SNIPPET_SERVICE.LINK) },
            { label: _(confT.conditions), link: generatePath(pages.SNIPPET_SERVICE.CONDITIONS.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    useEffect(() => {
        if (pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.TABS.indexOf(tab) === -1 || conditionData?.length === 0) {
            setNotFound(true)
        }
    }, [conditionData?.length, tab])

    const defaultFormState = useMemo(
        () => ({
            tab1: false,
            tab2: false,
            tab3: false,
        }),
        []
    )

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, { conditionId, tab: pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.TABS[i] }))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const { handleReset, context, resetIndex, dirty, formData, hasError } = useFormData({
        defaultFormState,
        data,
        defaultData: data,
        dirtyFormState,
        i18n: { promptDefaultMessage: _(g.promptDefaultMessage), default: _(g.default) },
    })

    const onSubmit = async () => {
        setPageLoading(true)

        try {
            // DATA FOR SAVE
            const dataForSave = cloneDeep(formData)
            delete dataForSave.id

            // FormSelect with multiple values
            if (dataForSave.deviceIdFilter) {
                dataForSave.deviceIdFilter = dataForSave.deviceIdFilter.map((device: string | OptionType) =>
                    typeof device === 'string' ? device : device.value
                )
            } else {
                dataForSave.deviceIdFilter = []
            }

            dataForSave.version = (parseInt(dataForSave.version, 10) + 1).toString()

            await updateConditionApi(formData.id || '', dataForSave)

            Notification.success(
                { title: _(confT.conditionUpdated), message: _(confT.conditionUpdatedMessage) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_UPDATE_SUCCESS }
            )

            setSearchParams({ version: dataForSave.version })
            refresh()

            setPageLoading(false)
        } catch (error: any) {
            let e = error
            if (!(error instanceof Error)) {
                e = new Error(error)
            }
            Notification.error(
                { title: _(confT.conditionUpdateError), message: e.message },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_UPDATE_ERROR }
            )
            setPageLoading(false)
        }
    }

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={conditionId!} loading={loading || pageLoading} name={data?.name} />}
            headlineCustomContent={<Selector />}
            loading={loading || pageLoading}
            notFound={notFound}
            title={data?.name}
            xPadding={false}
        >
            <FormContext.Provider value={context}>
                <Loadable condition={!!formData && !loading && !!data && Object.keys(data).length > 0 && Object.keys(formData).length > 0}>
                    {/* <DetailForm formData={formData} resetIndex={resetIndex} setFilterError={setFilterError} />*/}
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
                                dataTestId: testId.snippetService.conditions.detail.tabGeneral,
                                content: <Tab1 defaultFormData={formData} resetIndex={resetIndex} />,
                            },
                            {
                                name: _(g.filters),
                                id: 1,
                                dataTestId: testId.snippetService.conditions.detail.tabFilters,
                                content: <Tab2 defaultFormData={formData} resetIndex={resetIndex} setFilterError={setFilterError} />,
                            },
                            {
                                name: _(confT.APIAccessToken),
                                id: 2,
                                dataTestId: testId.snippetService.conditions.detail.tabApiAccessToken,
                                content: <Tab3 defaultFormData={formData} resetIndex={resetIndex} />,
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
                            <Button
                                dataTestId={testId.snippetService.conditions.detail.bottomPanelSave}
                                disabled={hasError || filterError}
                                loading={loading}
                                loadingText={_(g.loading)}
                                onClick={onSubmit}
                                variant='primary'
                            >
                                {_(g.saveChanges)}
                            </Button>
                        }
                        actionSecondary={
                            <Button
                                dataTestId={testId.snippetService.conditions.detail.bottomPanelReset}
                                disabled={loading}
                                onClick={handleReset}
                                variant='secondary'
                            >
                                {_(g.reset)}
                            </Button>
                        }
                        dataTestId={testId.snippetService.conditions.detail.bottomPanel}
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
