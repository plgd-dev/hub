import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import { FormContext } from '@shared-ui/common/context/FormContext'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import { useForm } from '@shared-ui/common/hooks'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './Step2.types'
import { Step2FormComponent } from '@/containers/SnippetService/Conditions/FomComponents'
import { useConditionFilterValidation } from '@/containers/SnippetService/hooks'

const Step2: FC<Props> = (props) => {
    const { defaultFormData, isActivePage } = props
    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const { updateField, watch, setValue } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab2',
    })

    const invalidFilters = useConditionFilterValidation({ watch })

    return (
        <form onSubmit={(e) => e.preventDefault()}>
            <FullPageWizard.Headline>{_(confT.applyFilters)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.createConditionDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>Headline H4</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>Popis čo tu uživateľ musí nastaviať a prípadne prečo</FullPageWizard.Description>

            <FullPageWizard.GroupHeadline>{_(g.filters)}</FullPageWizard.GroupHeadline>

            <Spacer type='pt-6'>
                <Step2FormComponent isActivePage={isActivePage} setValue={setValue} updateField={updateField} watch={watch} />
            </Spacer>

            <StepButtons
                disableNext={invalidFilters}
                i18n={{
                    back: _(g.back),
                    continue: _(g.continue),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(0)}
                onClickNext={() => setStep?.(2)}
                showRequiredMessage={false}
            />
        </form>
    )
}

Step2.displayName = 'Step2'

export default Step2
