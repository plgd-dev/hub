import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import ReactDOM from 'react-dom'
import cloneDeep from 'lodash/cloneDeep'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import Row from '@shared-ui/components/Atomic/Grid/Row'
import { ItemType, SubItemItem } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import { useRefs } from '@shared-ui/common/hooks/useRefs'
import { useFormData, useIsMounted } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import AppContext from '@shared-ui/app/share/AppContext'

import { useConditionsDetail } from '@/containers/SnippetService/hooks'
import PageLayout from '@/containers/Common/PageLayout'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import notificationId from '@/notificationId'
import DetailHeader from './DetailHeader'
import { messages as g } from '@/containers/Global.i18n'
import DetailForm from './DetailForm'
import { updateConditionApi } from '@/containers/SnippetService/rest'

const DetailPage: FC<any> = () => {
    const { conditionId } = useParams()
    const { formatMessage: _ } = useIntl()
    const { data, loading, error, refresh } = useConditionsDetail(conditionId!, !!conditionId)

    const [pageLoading, setPageLoading] = useState(false)
    const [activeItem, setActiveItem] = useState('0')

    const { refsByKey, setRef } = useRefs()
    const { collapsed } = useContext(AppContext)
    const isMounted = useIsMounted()

    const navigate = useNavigate()

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.conditions), link: generatePath(pages.CONDITIONS.LINK) },
            { label: _(confT.conditions), link: generatePath(pages.CONDITIONS.CONDITIONS.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_RESOURCES_CONFIGURATION_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const menu = useMemo(
        () => [
            { id: '0', link: '#general', title: _(g.general) },
            {
                id: '10',
                link: '#filters',
                title: _(g.filters),
                children: [
                    { id: '1', link: '#filtersDeviceId', title: _(g.deviceId) },
                    { id: '2', link: '#filtersResourcesType', title: _(confT.resourceType) },
                    { id: '3', link: '#filterHref', title: _(g.href) },
                    { id: '4', link: '#filterJqExpression', title: _(confT.JqExpression) },
                ],
            },
            { id: '5', link: '#APIAccessToken', title: _(confT.APIAccessToken) },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    const refs = Object.values(refsByKey).filter(Boolean)

    const handleItemClick = useCallback(
        (item: ItemType | SubItemItem) => {
            setActiveItem(item.id)
            const id = parseInt(item.id)
            const element = refs[id] as HTMLElement

            element?.scrollIntoView({ behavior: 'smooth' })
        },
        [refs]
    )

    const defaultFormState = useMemo(
        () => ({
            tab1: false,
        }),
        []
    )

    const { handleReset, context, resetIndex, dirty, formData, hasError } = useFormData({
        defaultFormState,
        data,
        i18n: { promptDefaultMessage: _(g.promptDefaultMessage), default: _(g.default) },
    })

    const onSubmit = async () => {
        setPageLoading(true)

        try {
            // DATA FOR SAVE
            const dataForSave = cloneDeep(formData)
            delete dataForSave.id

            await updateConditionApi(formData.id || '', dataForSave)

            Notification.success(
                { title: _(confT.conditionUpdated), message: _(confT.conditionUpdatedMessage) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_UPDATE_SUCCESS }
            )

            handleReset()
            refresh()

            setPageLoading(false)

            // temp
            navigate(generatePath(pages.CONDITIONS.CONDITIONS.LINK))
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
            loading={loading || pageLoading}
            title={data?.name}
            xPadding={false}
        >
            <Spacer style={{ height: '100%', overflow: 'hidden' }} type='pt-4 pl-10'>
                <Row style={{ height: '100%' }}>
                    <Column xl={3}>
                        <Spacer type='mb-4'>
                            <ContentMenu
                                activeItem={activeItem}
                                handleItemClick={handleItemClick}
                                handleSubItemClick={(subItem, parentItem) => handleItemClick(subItem)}
                                menu={menu}
                            />
                        </Spacer>
                    </Column>
                    <Column xl={1}></Column>
                    <Column style={{ height: '100%' }} xl={8}>
                        <Spacer style={{ height: '100%', overflow: 'auto' }} type='pr-10'>
                            <FormContext.Provider value={context}>
                                <Loadable condition={!!formData && !loading && !!data}>
                                    <DetailForm
                                        formData={formData}
                                        refs={{
                                            general: (element: HTMLElement) => setRef(element, '0'),
                                            filterDeviceId: (element: HTMLElement) => setRef(element, '1'),
                                            filterResourceType: (element: HTMLElement) => setRef(element, '2'),
                                            filterResourceHref: (element: HTMLElement) => setRef(element, '3'),
                                            filterJqExpression: (element: HTMLElement) => setRef(element, '4'),
                                            accessToken: (element: HTMLElement) => setRef(element, '5'),
                                        }}
                                        resetIndex={resetIndex}
                                    />
                                </Loadable>
                            </FormContext.Provider>
                        </Spacer>

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
                    </Column>
                </Row>
            </Spacer>
        </PageLayout>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
