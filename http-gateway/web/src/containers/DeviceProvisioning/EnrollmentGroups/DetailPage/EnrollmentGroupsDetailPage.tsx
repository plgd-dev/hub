import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'
import isEqual from 'lodash/isEqual'

import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import { useIsMounted } from '@shared-ui/common/hooks'
import AppContext from '@shared-ui/app/share/AppContext'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import PageLayout from '@/containers/Common/PageLayout'
import { messages as dpsT } from '../../DeviceProvisioning.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { Props } from './EnrollmentGroupsDetailPage.types'
import DetailHeader from '../DetailHeader'
import { useEnrollmentGroupDetail } from '@/containers/DeviceProvisioning/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import DetailForm from './DetailForm'

const EnrollmentGroupsDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { enrollmentId } = useParams()
    const { collapsed } = useContext(AppContext)

    const [pageLoading, setPageLoading] = useState(false)

    const { data, loading, error } = useEnrollmentGroupDetail(enrollmentId!)

    // const { data: hubData, loading: hubLoading, error: hubError } = useHubDetail(data?.hubIds[0]!, !!data?.hubIds)
    const isMounted = useIsMounted()

    const defaultFormState = useMemo(
        () => ({
            tab1: false,
            tab2: false,
        }),
        []
    )
    const [formDirty, setFormDirty] = useState(defaultFormState)
    const [formError, setFormError] = useState(defaultFormState)
    const [resetIndex, setResetIndex] = useState(0)

    const [formData, setFormData] = useState<any>(null)

    const isDirtyData = useMemo(() => !!data && !!formData && !isEqual(data, formData), [data, formData])
    const isDirty = useMemo(() => Object.values(formDirty).some((i) => i), [formDirty])

    // console.log({
    //     isDirtyData,
    //     isDirty,
    // })
    //
    // console.log({ data })
    // console.log(' ')

    useEffect(() => {
        if (data) {
            setFormData(data)
        }
    }, [data])

    // useEffect(() => {
    //     console.log('formData change!')
    //     console.log(formData)
    // }, [formData])

    useEffect(() => {
        const errorF = error

        if (errorF) {
            Notification.error(
                { title: _(t.enrollmentGroupsError), message: errorF },
                { notificationId: notificationId.HUB_DPS_ENROLLMENT_GROUP_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const breadcrumbs = useMemo(
        () => [
            { label: _(dpsT.deviceProvisioning), link: '/device-provisioning' },
            { label: _(dpsT.enrollmentGroups), link: '/device-provisioning/enrollment-groups' },
            { label: enrollmentId! },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const context = useMemo(
        () => ({
            ...getFormContextDefault(_(g.default)),
            updateData: (newFormData: any) => {
                console.log('updateFormData', newFormData)
                setFormData(newFormData)
            },
            setFormDirty,
            setFormError: () => {},
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const onSubmit = async () => {
        setPageLoading(true)

        try {
            console.log('formData')
            console.log(formData)
            // refresh()

            setPageLoading(false)
        } catch (error: any) {
            let e = error
            if (!(error instanceof Error)) {
                e = new Error(error)
            }
            Notification.error(
                { title: _(t.enrollmentGroupsError), message: e.message },
                { notificationId: notificationId.HUB_DPS_ENROLLMENT_GROUP_DETAIL_PAGE_ERROR }
            )
            setPageLoading(false)
        }
    }

    const handleReset = useCallback(() => {
        setFormData(data)
        setFormDirty(defaultFormState)
        setFormError(defaultFormState)
        setResetIndex((prev) => prev + 1)
    }, [data, defaultFormState])

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={<DetailHeader id={enrollmentId!} refresh={() => {}} />}
            loading={loading || pageLoading}
            title={enrollmentId}
        >
            <FormContext.Provider value={context}>
                <Loadable condition={!!formData}>
                    <DetailForm defaultFormData={formData} resetIndex={resetIndex} />
                </Loadable>
            </FormContext.Provider>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button
                                disabled={formError.tab1 || formError.tab2}
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
                        show={isDirty || isDirtyData}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </PageLayout>
    )
}

EnrollmentGroupsDetailPage.displayName = 'EnrollmentGroupsDetailPage'

export default EnrollmentGroupsDetailPage
