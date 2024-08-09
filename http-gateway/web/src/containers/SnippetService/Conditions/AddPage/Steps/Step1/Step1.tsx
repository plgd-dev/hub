import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { useForm } from '@shared-ui/common/hooks'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { Props, Inputs } from './Step1.types'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { useValidationsSchema } from '../../../validationSchema'
import testId from '@/testId'

const Step1: FC<Props> = (props) => {
    const { defaultFormData } = props
    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const schema = useValidationsSchema('tab1')

    const {
        formState: { errors },
        register,
        watch,
        updateField,
        control,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step1', schema })

    const name = watch('name')

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.createCondition)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.createConditionShortDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>{_(confT.createConditionSubHeadline)}</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>{_(confT.createConditionDescription)}</FullPageWizard.Description>

            <FullPageWizard.GroupHeadline>{_(g.general)}</FullPageWizard.GroupHeadline>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name' marginBottom={false}>
                <FormLabel required text={_(g.name)} />
                <FormInput
                    {...register('name')}
                    dataTestId={testId.snippetService.conditions.addPage.step1.form.name}
                    onBlur={(e) => updateField('name', e.target.value)}
                />
            </FormGroup>

            <Spacer type='pt-7'>
                <Controller
                    control={control}
                    name='enabled'
                    render={({ field: { onChange, value } }) => (
                        <TileToggle
                            darkBg
                            checked={value ?? false}
                            name={_(g.enabled)}
                            onChange={(e) => {
                                onChange(e.target.checked)
                                updateField('enabled', e.target.checked)
                            }}
                        />
                    )}
                />
            </Spacer>

            <StepButtons
                dataTestId={testId.snippetService.conditions.addPage.step1.buttons}
                disableNext={!name}
                i18n={{
                    back: _(g.back),
                    continue: _(g.continue),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickNext={() => setStep?.(1)}
            />
        </form>
    )
}

Step1.displayName = 'Step1'

export default Step1
