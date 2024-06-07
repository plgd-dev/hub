import { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'

import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { messages as confT } from '../../SnippetService.i18n'
import { pages } from '@/routes'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/LinkedHubs/utils'

const Step1 = lazy(() => import('./Steps/Step1'))
const Step2 = lazy(() => import('./Steps/Step2'))

const AddPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()
    const { step } = useParams()

    const steps = useMemo(
        () => [
            {
                name: _(confT.createConfig),
                description: _(confT.createConfigDescription),
                link: pages.CONDITIONS.APPLIED_DEVICE_CONFIG.ADD.STEPS[0],
            },
            {
                name: _(confT.applyToDevices),
                description: _(confT.applyToDevicesDescription),
                link: pages.CONDITIONS.APPLIED_DEVICE_CONFIG.ADD.STEPS[1],
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData, rehydrated] = usePersistentState<any>('dps-create-linked-hub-form', {})
    const [visitedStep, setVisitedStep] = useState<number>(activeItem)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(generatePath(pages.CONDITIONS.APPLIED_DEVICE_CONFIG.ADD.LINK, { step: steps[item].link }))

            if (item > visitedStep) {
                setVisitedStep(item)
            }
        },
        [navigate, steps, visitedStep]
    )

    const onSubmit = async () => {
        try {
            delete formData.id

            // await createLinkedHub(formatDataForSave(formData))

            Notification.success(
                {
                    title: _(confT.appliedDeviceConfigurationAddSuccess),
                    message: _(confT.appliedDeviceConfigurationAddSuccessMessage, { name: formData.name }),
                },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_DEVICE_ADD_SUCCESS }
            )

            setFormData({})

            navigate(pages.CONDITIONS.APPLIED_DEVICE_CONFIG.LINK)
        } catch (error: any) {
            Notification.error(
                { title: _(confT.appliedDeviceConfigurationAddError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_DEVICE_ADD_ERROR }
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
            title={_(confT.appliedDeviceConfiguration)}
            visitedStep={visitedStep}
        >
            <Loadable condition={rehydrated}>
                <FormContext.Provider value={context}>
                    <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                        <Step1 />
                        <Step2 />
                    </ContentSwitch>
                </FormContext.Provider>
            </Loadable>
        </FullPageWizard>
    )
}

AddPage.displayName = 'AddPage'

export default AddPage
