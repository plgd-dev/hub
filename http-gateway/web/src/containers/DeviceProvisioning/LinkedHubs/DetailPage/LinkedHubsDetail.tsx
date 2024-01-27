import React, { FC, lazy, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import { FormProvider, SubmitHandler, useForm } from 'react-hook-form'
import ReactDOM from 'react-dom'

import Tabs from '@shared-ui/components/Atomic/Tabs'
import { useIsMounted } from '@shared-ui/common/hooks'
import AppContext from '@shared-ui/app/share/AppContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import testId from '@/testId'
import { messages as g } from '@/containers/Global.i18n'
import { getTabRoute } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { messages as dpsT } from '@/containers/DeviceProvisioning/DeviceProvisioning.i18n'
import { Inputs } from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/LinkedHubsDetailPage.types'
import { updateLinkedHubData } from '@/containers/DeviceProvisioning/rest'
import PageLayout from '@/containers/Common/PageLayout'
import DetailHeader from '@/containers/DeviceProvisioning/LinkedHubs/DetailHeader'
import isEqual from 'lodash/isEqual'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'

const Tab1 = lazy(() => import('./Tabs/Tab1/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2/Tab2'))
// const Tab3 = lazy(() => import('./Tabs/Tab3/Tab3'))

const LinkedHubsDetail: FC<any> = (props) => {
    const { data, loading, defaultActiveTab } = props
    const { formatMessage: _ } = useIntl()
    const { hubId } = useParams()

    const isMounted = useIsMounted()
    const { collapsed } = useContext(AppContext)

    const navigate = useNavigate()

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)
    const [pageLoading, setPageLoading] = useState(false)
    const [formData, setFormData] = useState(data)

    useEffect(() => {
        setFormData(data)
    }, [data])

    // const [dirty, setDirty] = useState(false)
    //
    // const caPool = formMethods.watch('certificateAuthority.grpc.tls.caPool')
    //
    // useEffect(() => {
    //     let isDirty = false
    //
    //     if (!isEqual(caPool, formMethods.formState.defaultValues?.certificateAuthority?.grpc?.tls?.caPool)) {
    //         // console.log('change!')
    //         // console.log(caPool)
    //         isDirty = true
    //     }
    //
    //     if (dirty !== isDirty) {
    //         setDirty(isDirty)
    //     }
    // }, [caPool, dirty, formMethods.formState.defaultValues?.certificateAuthority?.grpc?.tls?.caPool])

    const handleTabChange = useCallback((i: number) => {
        console.log('handleTabChange')
        setActiveTabItem(i)

        navigate(`/device-provisioning/linked-hubs/${hubId}${getTabRoute(i)}`, { replace: true })

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

    const onSubmitForm = (data: any) => {
        console.log('%c onSubmit!', 'background: #222; color: #bada55')
        console.log(data)

        setPageLoading(true)

        try {
            // const { data: newData } = await updateLinkedHubData(hubId!, values)
            // updateData(newData)
            // console.log({ newData })

            setPageLoading(false)
        } catch (error) {
            console.log('error')

            setPageLoading(false)
        }
    }

    const onSubmit = () => {
        console.log('onSubmit')
    }

    const handleReset = () => {
        console.log('handleReset')
    }

    const context = useMemo(
        () => ({
            ...getFormContextDefault('default'),
            updateData: (data: any) => console.log(data),
        }),
        []
    )

    // console.log({ dirty: formMethods.formState.isDirty })
    // console.log({ isDirty: dirty })

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
                        // {
                        //     name: _(t.authorization),
                        //     id: 2,
                        //     dataTestId: testId.dps.linkedHubs.detail.tabAuthorization,
                        //     content: <Tab3 loading={loading} />,
                        // },
                    ]}
                />
            </FormContext.Provider>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button
                                // disabled={Object.keys(formMethods.formState.errors).length > 0}
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
                        show={false}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </PageLayout>
    )
}

LinkedHubsDetail.displayName = 'LinkedHubsDetail'

export default LinkedHubsDetail
