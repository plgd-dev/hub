import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate, useParams } from 'react-router-dom'

import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { createLinkedHub } from '@/containers/DeviceProvisioning/rest'
import notificationId from '@/notificationId'

const Step1 = lazy(() => import('./Steps/Step1'))
const Step2 = lazy(() => import('./Steps/Step2'))
const Step3 = lazy(() => import('./Steps/Step3'))
const Step4 = lazy(() => import('./Steps/Step4'))

const LinkNewHubPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()
    const { step } = useParams()

    const steps = useMemo(
        () => [
            {
                name: _(t.linkHub),
                description: 'Short description...',
                link: '',
            },
            {
                name: _(t.hubDetails),
                description: 'Short description...',
                link: '/hub-detail',
            },
            {
                name: _(t.certificateAuthorityConfiguration),
                description: 'Short description...',
                link: '/certificate-authority-Configuration',
            },
            {
                name: _(t.authorization),
                description: 'Short description...',
                link: '/authorization',
            },
        ],
        []
    )

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData] = usePersistentState<any>('dps-create-linked-hub-form', DEFAULT_FORM_DATA)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(`/device-provisioning/linked-hubs/link-new-hub${steps[item].link}`, { replace: true })
        },
        [navigate, steps]
    )

    const onSubmit = async () => {
        try {
            await createLinkedHub(formData)

            navigate(`/device-provisioning/linked-hubs`, { replace: true })
        } catch (error: any) {
            if (!(error instanceof Error)) {
                error = new Error(error)
            }

            Notification.error({ title: _(t.linkedHubsError), message: error.message }, { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_ERROR })
        }
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
                navigate('/device-provisioning/linked-hubs')
            }}
            onStepChange={onStepChange}
            steps={steps}
            title={_(t.linkNewHub)}
        >
            <FormContext.Provider value={context}>
                <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                    <Step1 defaultFormData={formData} />
                    <Step2 defaultFormData={formData} />
                    <Step3 defaultFormData={formData} />
                    <Step4 defaultFormData={formData} />
                </ContentSwitch>
            </FormContext.Provider>
        </FullPageWizard>
    )
}

LinkNewHubPage.displayName = 'LinkNewHubPage'

export default LinkNewHubPage
