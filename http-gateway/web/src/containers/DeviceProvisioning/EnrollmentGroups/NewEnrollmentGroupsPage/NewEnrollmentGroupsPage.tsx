import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate, useParams } from 'react-router-dom'
import cloneDeep from 'lodash/cloneDeep'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/EnrollmentGroups/NewEnrollmentGroupsPage/constants'
import { createEnrollmentGroup } from '@/containers/DeviceProvisioning/rest'
import { pemToString } from '@/containers/DeviceProvisioning/utils'

const Step1 = lazy(() => import('./Steps/Step1'))
const Step2 = lazy(() => import('./Steps/Step2'))
const Step3 = lazy(() => import('./Steps/Step3'))

const NewEnrollmentGroupsPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()
    const { step } = useParams()

    const steps = useMemo(
        () => [
            {
                name: _(t.enrollmentConfiguration),
                description: 'Short description...',
                link: '',
            },
            {
                name: _(t.deviceAuthentication),
                description: 'Short description...',
                link: '/device-authentication',
            },
            {
                name: _(t.deviceCredentials),
                description: 'Short description...',
                link: '/device-credentials',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData, rehydrated] = usePersistentState<any>('dps-create-enrollment-group-form', DEFAULT_FORM_DATA)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(`/device-provisioning/enrollment-groups/new-enrollment-group${steps[item].link}`, { replace: true })
        },
        [navigate, steps]
    )

    const onSubmit = async () => {
        const dataForSave = cloneDeep(formData)

        if (dataForSave.preSharedKey && dataForSave.preSharedKey !== '') {
            dataForSave.preSharedKey = pemToString(dataForSave.preSharedKey)
        }

        await createEnrollmentGroup(dataForSave)

        setFormData(DEFAULT_FORM_DATA)

        navigate(`/device-provisioning/enrollment-groups`, { replace: true })
    }

    const context = useMemo(
        () => ({
            ...getFormContextDefault(_(g.default)),
            updateData: (newFormData: any) => setFormData(newFormData),
            setFormError: () => {},
            setStep: onStepChange,
            onSubmit,
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <FullPageWizard
            activeStep={activeItem}
            i18n={{
                close: _(g.close),
            }}
            onClose={() => {
                setFormData(DEFAULT_FORM_DATA)
                navigate(`/device-provisioning/enrollment-groups`, { replace: true })
            }}
            onStepChange={onStepChange}
            steps={steps}
            title={_(t.addEnrollmentGroup)}
        >
            <Loadable condition={rehydrated}>
                <FormContext.Provider value={context}>
                    <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                        <Step1 defaultFormData={formData} />
                        <Step2 defaultFormData={formData} />
                        <Step3 defaultFormData={formData} onSubmit={onSubmit} />
                    </ContentSwitch>
                </FormContext.Provider>
            </Loadable>
        </FullPageWizard>
    )
}

NewEnrollmentGroupsPage.displayName = 'NewEnrollmentGroupsPage'

export default NewEnrollmentGroupsPage
