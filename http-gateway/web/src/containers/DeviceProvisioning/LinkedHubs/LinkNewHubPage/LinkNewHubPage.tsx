import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'

import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../LinkedHubs.i18n'
import { DEFAULT_FORM_DATA, formatDataForSave } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
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
                description: _(t.linkHubDescription),
                link: pages.DPS.LINKED_HUBS.ADD.TABS[0],
            },
            {
                name: _(t.hubDetails),
                description: _(t.hubDetailsDescription),
                link: pages.DPS.LINKED_HUBS.ADD.TABS[1],
            },
            {
                name: _(t.certificateAuthorityDescription),
                description: _(t.certificateAuthorityDescription),
                link: pages.DPS.LINKED_HUBS.ADD.TABS[2],
            },
            {
                name: _(t.authorization),
                description: _(t.authorizationDescription),
                link: pages.DPS.LINKED_HUBS.ADD.TABS[3],
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData, rehydrated] = usePersistentState<any>('dps-create-linked-hub-form', DEFAULT_FORM_DATA)
    const [visitedStep, setVisitedStep] = useState<number>(activeItem)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(generatePath(pages.DPS.LINKED_HUBS.ADD.LINK, { step: steps[item].link }))

            if (item > visitedStep) {
                setVisitedStep(item)
            }
        },
        [navigate, steps, visitedStep]
    )

    const onSubmit = async () => {
        try {
            delete formData.id

            await createLinkedHub(formatDataForSave(formData))

            Notification.success(
                { title: _(t.linkedHubsCreated), message: _(t.linkedHubsCreatedMessage) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_CREATED }
            )

            setFormData(DEFAULT_FORM_DATA)

            navigate(pages.DPS.LINKED_HUBS.LINK)
        } catch (error: any) {
            Notification.error(
                { title: _(t.linkedHubsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_ERROR }
            )
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
                navigate(pages.DPS.LINKED_HUBS.LINK)
            }}
            onStepChange={onStepChange}
            steps={steps}
            title={_(t.linkNewHub)}
            visitedStep={visitedStep}
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
