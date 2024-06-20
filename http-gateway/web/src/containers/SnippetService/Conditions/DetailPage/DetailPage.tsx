import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import ReactDOM from 'react-dom'
import cloneDeep from 'lodash/cloneDeep'
import { useMediaQuery } from 'react-responsive'

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
import { useVersion } from '@shared-ui/common/hooks/use-version'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

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
    const { data: conditionData, loading, error, refresh } = useConditionsDetail(conditionId || '', !!conditionId)

    const { Selector, data } = useVersion({
        i18n: { version: _(g.version), selectVersion: _(confT.selectVersion) },
        versionData: conditionData,
        refresh,
    })

    const [pageLoading, setPageLoading] = useState(false)
    const [activeItem, setActiveItem] = useState('0')

    const { refsByKey, setRef } = useRefs()
    const { collapsed } = useContext(AppContext)
    const isMounted = useIsMounted()

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

            setTimeout(() => {
                element?.scrollIntoView({ behavior: 'smooth' })
            }, 0)
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

            // FormSelect with multiple values
            dataForSave.deviceIdFilter = dataForSave.deviceIdFilter.map((device: string | OptionType) => (typeof device === 'string' ? device : device.value))

            dataForSave.version = (parseInt(dataForSave.version, 10) + 1).toString()

            await updateConditionApi(formData.id || '', dataForSave)

            Notification.success(
                { title: _(confT.conditionUpdated), message: _(confT.conditionUpdatedMessage) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_UPDATE_SUCCESS }
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
                { title: _(confT.conditionUpdateError), message: e.message },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_PAGE_UPDATE_ERROR }
            )
            setPageLoading(false)
        }
    }

    const isDesktopOrLaptop = useMediaQuery(
        {
            query: '(min-width: 1200px)',
        },
        undefined,
        () => {}
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={conditionId!} loading={loading || pageLoading} name={data?.name} />}
            headlineCustomContent={<Selector />}
            loading={loading || pageLoading}
            title={data?.name}
            xPadding={false}
        >
            <Spacer style={{ height: '100%', overflow: 'hidden' }} type='pt-4 pl-10'>
                <Row
                    style={
                        isDesktopOrLaptop
                            ? { height: '100%' }
                            : { display: 'flex', flexDirection: 'column', flexWrap: 'nowrap', overflow: 'hidden', height: '100%' }
                    }
                >
                    <Column xl={3}>
                        <Spacer type={`mb-4${isDesktopOrLaptop ? '' : ' pr-10'}`}>
                            <ContentMenu
                                activeItem={activeItem}
                                handleItemClick={handleItemClick}
                                handleSubItemClick={(subItem, parentItem) => handleItemClick(subItem)}
                                menu={menu}
                            />
                        </Spacer>
                    </Column>
                    {isDesktopOrLaptop && <Column xl={1}></Column>}
                    <Column style={isDesktopOrLaptop ? { height: '100%' } : { flex: '1 1 auto', overflow: 'hidden' }} xl={8}>
                        <Spacer style={{ height: '100%', overflow: 'auto' }} type='pr-10'>
                            <FormContext.Provider value={context}>
                                <Loadable condition={!!formData && !loading && !!data && Object.keys(data).length > 0 && Object.keys(formData).length > 0}>
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
                    </Column>
                </Row>
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
        </PageLayout>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
