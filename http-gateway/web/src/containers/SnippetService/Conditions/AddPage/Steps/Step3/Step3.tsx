import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'

import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { useForm } from '@shared-ui/common/hooks'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import { FormContext } from '@shared-ui/common/context/FormContext'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { Props, Inputs } from './Step3.types'
import { messages as g } from '@/containers/Global.i18n'
import { useConfigurationList } from '@/containers/SnippetService/hooks'

const Step3: FC<Props> = (props) => {
    const { defaultFormData, onFinish } = props

    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)
    const { data, loading } = useConfigurationList()

    const {
        formState: { errors },
        updateField,
        register,
        watch,
        setValue,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab3',
    })

    const options = useMemo(
        () =>
            loading
                ? []
                : data?.map((item: { name: string; id: string; version: string }) => ({
                      label: `${item.name} - v${item.version} - ${item.id}`,
                      value: item.id,
                  })) || [],
        [data, loading]
    )

    const configurationId = watch('configurationId')
    const apiAccessToken = watch('apiAccessToken')

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.selectConfiguration)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.selectConfigurationDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>Headline H4</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>Popis čo tu uživateľ musí nastaviať a prípadne prečo</FullPageWizard.Description>

            <FormGroup error={errors.configurationId ? _(g.requiredField, { field: _(confT.configuration) }) : undefined} id='configurationId'>
                <FormLabel required text={_(confT.selectConfiguration)} />
                <FormSelect
                    isClearable
                    error={!!errors.configurationId}
                    onChange={(option: OptionType) => {
                        const v = option ? option.value : ''
                        setValue('configurationId', v.toString())
                        updateField('configurationId', v)
                    }}
                    options={options}
                    value={configurationId ? options?.find((o: OptionType) => o.value === configurationId) : null}
                />
            </FormGroup>

            <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
                <FormLabel required text={_(confT.APIAccessToken)} />
                <FormTextarea
                    {...register('apiAccessToken', { required: true, validate: (val) => val !== '' })}
                    onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                    style={{ height: 450 }}
                />
            </FormGroup>

            <StepButtons
                disableNext={!configurationId || !apiAccessToken || Object.keys(errors).length > 0}
                i18n={{
                    back: _(g.back),
                    continue: _(g.createAndSave),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(1)}
                onClickNext={onFinish}
            />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
