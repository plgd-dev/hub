import React, { FC, lazy, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import isEqual from 'lodash/isEqual'
import ReactDOM from 'react-dom'
import { useRecoilState } from 'recoil'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import AppContext from '@shared-ui/app/share/AppContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import { useIsMounted } from '@shared-ui/common/hooks'
import { updateLinkedHubData } from '@/containers/DeviceProvisioning/rest'
import { buildCATranslations } from '@shared-ui/components/Organisms/CaPoolModal/utils'
import { useBeforeUnload } from '@shared-ui/common/hooks/useBeforeUnload'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { messages as t } from '../LinkedHubs.i18n'
import { HubDataType, Props } from './LinkedHubsDetailPage.types'
import { useHubDetail } from '../../hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import PageLayout from '@/containers/Common/PageLayout'
import DetailHeader from '@/containers/DeviceProvisioning/LinkedHubs/DetailHeader'
import testId from '@/testId'
import { messages as dpsT } from '@/containers/DeviceProvisioning/DeviceProvisioning.i18n'
import { dirtyFormState } from '@/store/recoil.store'
import { pages } from '@/routes'
import { formatDataForSave } from '@/containers/DeviceProvisioning/LinkedHubs/utils'

const Tab1 = lazy(() => import('./Tabs/Tab1/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3/Tab3'))

const LinkedHubsDetailPage: FC<Props> = () => {
    const { formatMessage: _ } = useIntl()
    const { hubId, tab: tabRoute } = useParams()
    const isMounted = useIsMounted()
    const { collapsed } = useContext(AppContext)
    const navigate = useNavigate()
    const tab = tabRoute || ''

    const { data, loading, error, refresh } = useHubDetail(hubId!, !!hubId)
    // transform gateways string[] => {value: string}[]
    const [defaultData, setDefaultData] = useState<any>(undefined)

    useEffect(() => {
        if (data) {
            setDefaultData({
                ...data,
                gateways: data?.gateways ? data.gateways.map((gateway: string) => ({ value: gateway })) : [],
            })
        }
    }, [data])

    const [activeTabItem, setActiveTabItem] = useState(tab ? pages.DPS.LINKED_HUBS.DETAIL.TABS.indexOf(tab) : 0)
    const [pageLoading, setPageLoading] = useState(false)
    const [formData, setFormData] = useState<any>(defaultData)
    const defaultFormState = useMemo(
        () => ({
            tab1: false,
            tab2Content1: false,
            tab2Content2: false,
            tab3Content1: false,
            tab3Content2: false,
            tab3Content3: false,
            tab3Content4: false,
        }),
        []
    )
    const [formDirty, setFormDirty] = useState(defaultFormState)
    const [formError, setFormError] = useState(defaultFormState)
    const [resetIndex, setResetIndex] = useState(0)
    const [dirtyState, setDirtyState] = useRecoilState(dirtyFormState)
    const [notFound, setNotFound] = useState(false)

    useEffect(() => {
        setFormData(defaultData)
    }, [defaultData])

    const isDirtyData = useMemo(() => !isEqual(defaultData, formData), [defaultData, formData])
    const isDirty = useMemo(() => Object.values(formDirty).some((i) => i), [formDirty])
    const hasError = useMemo(() => Object.values(formError).some((i) => i), [formError])

    useEffect(() => {
        const dirty = isDirty || isDirtyData
        if (dirtyState !== dirty) {
            setDirtyState(dirty)
        }
    }, [dirtyState, isDirty, isDirtyData, setDirtyState])

    useBeforeUnload({
        when: (isDirty || isDirtyData) && process.env.REACT_APP_DIRTY_FORM_CHECKER !== 'false',
        message: _(g.promptDefaultMessage),
    })

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: generatePath(pages.DPS.LINK) },
            { label: _(t.linkedHubs), link: generatePath(pages.DPS.LINKED_HUBS.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(t.linkedHubsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    useEffect(() => {
        if (pages.DPS.LINKED_HUBS.DETAIL.TABS.indexOf(tab) === -1) {
            setNotFound(true)
        }
    }, [tab])

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(generatePath(pages.DPS.LINKED_HUBS.DETAIL.LINK, { hubId: hubId, tab: pages.DPS.LINKED_HUBS.DETAIL.TABS[i], section: '' }))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const context = useMemo(
        () => ({
            ...getFormContextDefault(_(g.default)),
            updateData: (newFormData: HubDataType) => setFormData(newFormData),
            setFormError,
            setFormDirty,
            i18n: buildCATranslations(_, t, g),
            compactFormComponentsView: true,
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const onSubmit = async () => {
        setPageLoading(true)

        try {
            await updateLinkedHubData(hubId as string, formatDataForSave(formData))

            Notification.success(
                { title: _(t.linkedHubUpdated), message: _(t.linkedHubUpdatedMessage) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_UPDATED }
            )

            refresh()

            setPageLoading(false)
        } catch (error: any) {
            Notification.error(
                { title: _(t.linkedHubsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_ERROR }
            )
            setPageLoading(false)
        }
    }

    const handleReset = useCallback(() => {
        setFormData(defaultData)
        setFormDirty(defaultFormState)
        setFormError(defaultFormState)
        setResetIndex((prev) => prev + 1)
    }, [defaultData, defaultFormState])

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={hubId!} refresh={() => {}} />}
            loading={loading || pageLoading}
            notFound={notFound}
            title={data?.name}
            xPadding={false}
        >
            <FormContext.Provider value={context}>
                {formData && (
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
                                name: _(t.details),
                                id: 0,
                                dataTestId: testId.dps.linkedHubs.detail.tabDetails,
                                content: <Tab1 defaultFormData={formData} resetIndex={resetIndex} />,
                            },
                            {
                                name: _(t.certificateAuthority),
                                id: 1,
                                dataTestId: testId.dps.linkedHubs.detail.tabCertificateAuthority,
                                content: <Tab2 defaultFormData={formData} loading={loading} />,
                            },
                            {
                                name: _(t.authorization),
                                id: 2,
                                dataTestId: testId.dps.linkedHubs.detail.tabAuthorization,
                                content: <Tab3 defaultFormData={formData} loading={loading} />,
                            },
                        ]}
                    />
                )}
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
                        show={isDirty || isDirtyData}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </PageLayout>
    )
}

LinkedHubsDetailPage.displayName = 'LinkedHubsDetailPage'

export default LinkedHubsDetailPage
