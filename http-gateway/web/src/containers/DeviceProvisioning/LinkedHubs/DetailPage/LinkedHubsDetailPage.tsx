import React, { FC, lazy, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import isEqual from 'lodash/isEqual'
import ReactDOM from 'react-dom'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import AppContext from '@shared-ui/app/share/AppContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import { useIsMounted } from '@shared-ui/common/hooks'

import { messages as t } from '../LinkedHubs.i18n'
import { HubDataType, Props } from './LinkedHubsDetailPage.types'
import { useHubDetail } from '../../hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import PageLayout from '@/containers/Common/PageLayout'
import DetailHeader from '@/containers/DeviceProvisioning/LinkedHubs/DetailHeader'
import testId from '@/testId'
import { messages as dpsT } from '@/containers/DeviceProvisioning/DeviceProvisioning.i18n'
import { getTabRoute } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { updateLinkedHubData } from '@/containers/DeviceProvisioning/rest'

const Tab1 = lazy(() => import('./Tabs/Tab1/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3/Tab3'))

const LinkedHubsDetailPage: FC<Props> = (props) => {
    const { defaultActiveTab } = props

    const { formatMessage: _ } = useIntl()
    const { hubId } = useParams()
    const isMounted = useIsMounted()
    const { collapsed } = useContext(AppContext)
    const navigate = useNavigate()

    const { data, loading, error, refresh } = useHubDetail(hubId!, !!hubId)

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)
    const [pageLoading, setPageLoading] = useState(false)
    const [formData, setFormData] = useState(data)
    const [formError, setFormError] = useState({
        tab1: false,
        tab2Content1: false,
        tab2Content2: false,
        tab3Content1: false,
        tab3Content2: false,
        tab3Content3: false,
        tab3Content4: false,
    })

    useEffect(() => {
        setFormData(data)
    }, [data])

    const isDirty = useMemo(() => !isEqual(data, formData), [data, formData])

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(t.linkedHubs), link: '/device-provisioning/linked-hubs' },
            { label: hubId! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    useEffect(() => {
        if (error) {
            Notification.error({ title: _(t.linkedHubsError), message: error }, { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_ERROR })
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)

        navigate(`/device-provisioning/linked-hubs/${hubId}${getTabRoute(i)}`, { replace: true })

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const context = useMemo(
        () => ({
            ...getFormContextDefault(_(g.default)),
            updateData: (newFormData: HubDataType) => setFormData(newFormData),
            setFormError,
            i18n: {
                algorithm: _(t.algorithm),
                authorityInfoAIA: _(t.authorityInfoAIA),
                authorityKeyID: _(t.authorityKeyID),
                basicConstraints: _(t.basicConstraints),
                certificateAuthority: _(t.certificateAuthority),
                certificatePolicies: _(t.certificatePolicies),
                commonName: _(t.commonName),
                country: _(t.country),
                dNSName: _(t.dNSName),
                download: _(t.download),
                embeddedSCTs: _(t.embeddedSCTs),
                exponent: _(t.exponent),
                extendedKeyUsages: _(t.extendedKeyUsages),
                fingerprints: _(t.fingerprints),
                issuerName: _(t.issuerName),
                keySize: _(t.keySize),
                keyUsages: _(t.keyUsages),
                location: _(t.location),
                logID: _(t.logID),
                menuTitle: _(t.certificates),
                method: _(t.method),
                miscellaneous: _(t.miscellaneous),
                modules: _(t.modules),
                no: _(g.no),
                notAfter: _(t.notAfter),
                notBefore: _(t.notBefore),
                organization: _(t.organization),
                policy: _(t.policy),
                publicKeyInfo: _(t.publicKeyInfo),
                purposes: _(t.purposes),
                serialNumber: _(t.serialNumber),
                signatureAlgorithm: _(t.signatureAlgorithm),
                subjectAltNames: _(t.subjectAltNames),
                subjectKeyID: _(t.subjectKeyID),
                subjectName: _(t.subjectName),
                timestamp: _(t.timestamp),
                validity: _(t.validity),
                value: _(t.value),
                version: _(g.version),
                yes: _(g.yes),
            },
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const onSubmit = async () => {
        setPageLoading(true)

        try {
            await updateLinkedHubData(hubId!, formData)
            refresh()

            setPageLoading(false)
        } catch (error: any) {
            if (!(error instanceof Error)) {
                error = new Error(error)
            }
            Notification.error(
                { title: _(t.linkedHubsError), message: error.message },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_DETAIL_PAGE_ERROR }
            )
            setPageLoading(false)
        }
    }

    const handleReset = useCallback(() => {
        setFormData(data)
    }, [data])

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={hubId!} refresh={() => {}} />}
            loading={loading || pageLoading}
            title={data?.name}
            xPadding={false}
        >
            <FormContext.Provider value={context}>
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
                            content: <Tab1 defaultFormData={formData} />,
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
            </FormContext.Provider>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button
                                disabled={
                                    formError.tab1 ||
                                    formError.tab2Content1 ||
                                    formError.tab2Content2 ||
                                    formError.tab3Content1 ||
                                    formError.tab3Content2 ||
                                    formError.tab3Content3 ||
                                    formError.tab3Content4
                                }
                                loading={loading}
                                loadingText={_(g.loading)}
                                onClick={onSubmit}
                                variant='primary'
                            >
                                {_(g.saveChanges)}
                            </Button>
                        }
                        actionSecondary={
                            <Button disabled={loading} onClick={handleReset} variant='secondary'>
                                {_(g.reset)}
                            </Button>
                        }
                        leftPanelCollapsed={collapsed}
                        show={isDirty}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </PageLayout>
    )
}

LinkedHubsDetailPage.displayName = 'LinkedHubsDetailPage'

export default LinkedHubsDetailPage
