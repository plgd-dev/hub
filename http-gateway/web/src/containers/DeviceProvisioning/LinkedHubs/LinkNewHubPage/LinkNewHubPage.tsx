import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate, useParams } from 'react-router-dom'
import cloneDeep from 'lodash/cloneDeep'

import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import notificationId from '@/notificationId'
import { createLinkedHub } from '@/containers/DeviceProvisioning/rest'
import { pages } from '@/routes'

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
                description: 'Pre-configure Hub',
                link: '',
            },
            {
                name: _(t.hubDetails),
                description: 'Basic setup',
                link: '/hub-detail',
            },
            {
                name: _(t.certificateAuthority),
                description: 'Signing certificate',
                link: '/certificate-authority',
            },
            {
                name: _(t.authorization),
                description: 'JWT Token Acquisition',
                link: '/authorization',
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData, rehydrated] = usePersistentState<any>('dps-create-linked-hub-form', DEFAULT_FORM_DATA)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(`${pages.DPS.LINKED_HUBS.ADD}${steps[item].link}`, { replace: true })
        },
        [navigate, steps]
    )

    const onSubmit = async () => {
        try {
            delete formData.id
            const copy = cloneDeep(formData)
            copy.gateways = copy.gateways.map((i: { value: string }) => i.value)

            await createLinkedHub(copy)

            setFormData(DEFAULT_FORM_DATA)

            navigate(pages.DPS.LINKED_HUBS.LINK, { replace: true })
        } catch (error: any) {
            let e = error
            if (!(error instanceof Error)) {
                e = new Error(error)
            }

            Notification.error({ title: _(t.linkedHubsError), message: e.message }, { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_ERROR })
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
                navigate(pages.DPS.LINKED_HUBS.LINK, { replace: true })
            }}
            onStepChange={onStepChange}
            steps={steps}
            title={_(t.linkNewHub)}
        >
            <Loadable condition={rehydrated}>
                <FormContext.Provider value={context}>
                    <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                        <Step1 defaultFormData={formData} />
                        <Step2 defaultFormData={formData} />
                        <Step3 defaultFormData={formData} />
                        <Step4 defaultFormData={formData} onSubmit={onSubmit} />
                    </ContentSwitch>
                </FormContext.Provider>
            </Loadable>
        </FullPageWizard>
    )
}

LinkNewHubPage.displayName = 'LinkNewHubPage'

export default LinkNewHubPage
