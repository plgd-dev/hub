import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

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
import { useResourcesConfigList } from '@/containers/SnippetService/hooks'

const Step3: FC<Props> = (props) => {
    const { defaultFormData, onFinish } = props

    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)
    const { data, loading } = useResourcesConfigList()

    const {
        formState: { errors },
        updateField,
        register,
        control,
        watch,
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

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.selectConfiguration)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.selectConfigurationDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>Headline H4</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>Popis čo tu uživateľ musí nastaviať a prípadne prečo</FullPageWizard.Description>

            <FormGroup error={errors.configurationId ? _(g.requiredField, { field: _(confT.configuration) }) : undefined} id='configurationId'>
                <FormLabel text={_(confT.selectConfiguration)} />
                <div>
                    <Controller
                        control={control}
                        name='configurationId'
                        render={({ field: { onChange, value } }) => (
                            <FormSelect
                                isClearable
                                error={!!errors.configurationId}
                                onChange={(option: OptionType) => {
                                    onChange(option)
                                    updateField('configurationId', option)
                                }}
                                options={options}
                                value={configurationId}
                            />
                        )}
                    />
                </div>
            </FormGroup>

            <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
                <FormLabel text={_(confT.APIAccessToken)} />
                <FormTextarea
                    {...register('apiAccessToken', { required: true, validate: (val) => val !== '' })}
                    onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                    style={{ height: 450 }}
                />
            </FormGroup>

            <StepButtons
                disableNext={false}
                i18n={{
                    back: _(g.back),
                    continue: _(g.createAndSave),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(1)}
                onClickNext={onFinish}
                showRequiredMessage={false}
            />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
