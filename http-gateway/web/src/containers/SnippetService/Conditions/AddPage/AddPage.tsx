import { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'

import { getApiErrorMessage } from '@shared-ui/common/utils'
import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

import { messages as g } from '@/containers/Global.i18n'
import { DEFAULT_CONDITIONS_DATA } from '@/containers/SnippetService/constants'
import { pages } from '@/routes'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import notificationId from '@/notificationId'
import cloneDeep from 'lodash/cloneDeep'
import { createConditionApi } from '@/containers/SnippetService/rest'
import testId from '@/testId'

const Step1 = lazy(() => import('./Steps/Step1'))
const Step2 = lazy(() => import('./Steps/Step2'))
const Step3 = lazy(() => import('./Steps/Step3'))

const AddPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()
    const { step } = useParams()

    const steps = useMemo(
        () => [
            {
                name: _(confT.createCondition),
                description: _(confT.createConditionShortDescription),
                link: pages.SNIPPET_SERVICE.CONDITIONS.ADD.STEPS[0],
            },
            {
                name: _(confT.applyFilters),
                description: _(confT.applyFiltersShortDescription),
                link: pages.SNIPPET_SERVICE.CONDITIONS.ADD.STEPS[1],
            },
            {
                name: _(confT.selectConfiguration),
                description: _(confT.selectConfigurationShortDescription),
                link: pages.SNIPPET_SERVICE.CONDITIONS.ADD.STEPS[2],
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData, rehydrated] = usePersistentState<any>('snippet-service-create-condition', DEFAULT_CONDITIONS_DATA)
    const [visitedStep, setVisitedStep] = useState<number>(activeItem)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(generatePath(pages.SNIPPET_SERVICE.CONDITIONS.ADD.LINK, { tab: steps[item].link }))

            if (item > visitedStep) {
                setVisitedStep(item)
            }
        },
        [navigate, steps, visitedStep]
    )

    const onSubmit = async () => {
        try {
            delete formData.id

            const dataForSave = cloneDeep(formData)

            // FormSelect with multiple values
            if (dataForSave.deviceIdFilter) {
                dataForSave.deviceIdFilter = dataForSave.deviceIdFilter.map((device: string | OptionType) =>
                    typeof device === 'string' ? device : device.value
                )
            } else {
                dataForSave.deviceIdFilter = []
            }

            await createConditionApi(dataForSave)

            Notification.success(
                {
                    title: _(confT.addConditionsSuccess),
                    message: _(confT.addConditionsSuccessMessage, { name: formData.name }),
                },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_ADD_PAGE_SUCCESS }
            )

            setFormData(DEFAULT_CONDITIONS_DATA)
            navigate(pages.SNIPPET_SERVICE.CONDITIONS.LINK)
        } catch (error: any) {
            Notification.error(
                { title: _(confT.addConditionError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_ADD_PAGE_ERROR }
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
            dataTestId={testId.snippetService.conditions.addPage.wizard}
            i18n={{
                close: _(g.close),
            }}
            onClose={() => {
                setFormData(DEFAULT_CONDITIONS_DATA)
                navigate(pages.SNIPPET_SERVICE.CONDITIONS.LINK)
            }}
            onStepChange={onStepChange}
            steps={steps}
            title={_(confT.addCondition)}
            visitedStep={visitedStep}
        >
            <Loadable condition={rehydrated}>
                <FormContext.Provider value={context}>
                    <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                        <Step1 defaultFormData={formData} />
                        <Step2 defaultFormData={formData} isActivePage={activeItem === 1} />
                        <Step3 defaultFormData={formData} isActivePage={activeItem === 2} onFinish={onSubmit} />
                    </ContentSwitch>
                </FormContext.Provider>
            </Loadable>
        </FullPageWizard>
    )
}

AddPage.displayName = 'AddPage'

export default AddPage
