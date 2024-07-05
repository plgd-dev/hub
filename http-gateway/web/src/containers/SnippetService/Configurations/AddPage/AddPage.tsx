import React, { FC, lazy, useContext, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'
import ReactDOM from 'react-dom'

import Loadable from '@shared-ui/components/Atomic/Loadable'
import { FormContext } from '@shared-ui/common/context/FormContext'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { useFormData, useIsMounted } from '@shared-ui/common/hooks'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import AppContext from '@shared-ui/app/share/AppContext'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '../../SnippetService.i18n'
import { pages } from '@/routes'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { DEFAULT_CONFIGURATIONS_DATA } from '@/containers/SnippetService/constants'
import { createConfigurationApi } from '@/containers/SnippetService/rest'
import { dirtyFormState } from '@/store/recoil.store'
import testId from '@/testId'
import { formatConfigurationResources } from '@/containers/SnippetService/utils'

const Tab1 = lazy(() => import('../DetailPage/Tabs/Tab1'))

const AddPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()

    const defaultFormState = useMemo(
        () => ({
            tab1: false,
        }),
        []
    )

    const [loadingState, setLoadingState] = useState(false)
    const [resourcesError, setResourcesError] = useState(false)
    const { collapsed } = useContext(AppContext)
    const isMounted = useIsMounted()

    const { handleReset, context, resetIndex, dirty, formData, hasError, setFormData, rehydrated } = useFormData({
        defaultFormState,
        defaultData: DEFAULT_CONFIGURATIONS_DATA,
        dirtyFormState,
        i18n: { promptDefaultMessage: _(g.promptDefaultMessage), default: _(g.default) },
        localStorageKey: 'snippet-service-create-configuration',
    })

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.snippetService), link: generatePath(pages.SNIPPET_SERVICE.LINK) },
            { label: _(confT.configurations), link: generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.LINK) },
            { label: _(confT.createNewConfiguration) },
        ],
        [_]
    )

    const onSubmit = async () => {
        try {
            setLoadingState(true)

            const createdData = await createConfigurationApi(formatConfigurationResources(formData))

            Notification.success(
                {
                    title: _(confT.addConfigurationSuccess),
                    message: _(confT.addConfigurationSuccessMessage, { name: formData.name }),
                },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_ADD_SUCCESS }
            )

            setFormData(DEFAULT_CONFIGURATIONS_DATA)
            setLoadingState(false)

            navigate(generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, { configurationId: createdData.data.id, tab: '' }))
        } catch (error: any) {
            Notification.error(
                { title: _(confT.addConfigurationError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONFIGURATIONS_LIST_PAGE_ADD_ERROR }
            )
            setLoadingState(false)
        }
    }

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loadingState} notFound={false} title={_(confT.createNewConfiguration)} xPadding={false}>
            <Loadable condition={rehydrated}>
                <FormContext.Provider value={context}>
                    <Tabs
                        fullHeight
                        innerPadding
                        isAsync
                        activeItem={0}
                        onItemChange={() => {}}
                        style={{
                            height: '100%',
                        }}
                        tabs={[
                            {
                                name: _(confT.generalAndResources),
                                id: 0,
                                dataTestId: testId.snippetService.configurations.detail.tabGeneral,
                                content: (
                                    <Tab1
                                        defaultFormData={formData}
                                        isActiveTab={true}
                                        loading={loadingState}
                                        resetIndex={resetIndex}
                                        setResourcesError={setResourcesError}
                                    />
                                ),
                            },
                            {
                                name: _(confT.conditions),
                                id: 1,
                                dataTestId: testId.snippetService.configurations.detail.tabConditions,
                                content: <div />,
                                disabled: true,
                            },
                            {
                                name: _(confT.appliedConfiguration),
                                id: 2,
                                dataTestId: testId.snippetService.configurations.detail.tabAppliedConfiguration,
                                content: <div />,
                                disabled: true,
                            },
                        ]}
                    />
                </FormContext.Provider>
            </Loadable>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button
                                disabled={hasError || resourcesError || !dirty || !rehydrated}
                                loading={loadingState}
                                loadingText={_(g.loading)}
                                onClick={onSubmit}
                                variant='primary'
                            >
                                {_(g.create)}
                            </Button>
                        }
                        actionSecondary={
                            <Button disabled={loadingState || !dirty || !rehydrated} onClick={handleReset} variant='secondary'>
                                {_(g.reset)}
                            </Button>
                        }
                        leftPanelCollapsed={collapsed}
                        show={true}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </PageLayout>
    )
}

AddPage.displayName = 'AddPage'

export default AddPage
