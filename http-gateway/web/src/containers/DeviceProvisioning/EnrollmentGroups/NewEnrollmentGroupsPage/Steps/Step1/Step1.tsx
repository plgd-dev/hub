import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import * as commonStyles from '@shared-ui/components/Templates/FullPageWizard/FullPageWizardCommon.styles'
import { FormContext } from '@shared-ui/common/context/FormContext'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { useLinkedHubsList } from '@/containers/DeviceProvisioning/hooks'
import { useForm } from '@shared-ui/common/hooks'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

import { messages as t } from '../../../EnrollmentGroups.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Inputs } from '../../../EnrollmentGroups.types'

const Step1: FC<any> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { data: hubsData } = useLinkedHubsList()
    const { updateData, setFormError, setStep } = useContext(FormContext)

    const {
        formState: { errors },
        register,
        control,
        updateField,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        errorKey: 'step1',
    })

    const linkedHubs = useMemo(
        () =>
            hubsData
                ? hubsData.map((linkedHub: { name: string; id: string }) => ({
                      value: linkedHub.id,
                      label: linkedHub.name,
                  }))
                : [],
        [hubsData]
    )

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.enrollmentConfiguration)}</h1>
            <FullPageWizard.Description large>{_(t.addEnrollmentGroupDescription)}</FullPageWizard.Description>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                <FormInput
                    {...register('name', { required: true, validate: (val) => val !== '' })}
                    onBlur={(e) => updateField('name', e.target.value)}
                    placeholder={_(g.name)}
                />
            </FormGroup>

            <FormGroup error={errors.hubIds ? _(g.requiredField, { field: _(t.linkedHubs) }) : undefined} id='linkedHubs'>
                <div>
                    <Controller
                        control={control}
                        name='hubIds'
                        render={({ field: { onChange, value } }) => (
                            <FormSelect
                                isMulti
                                error={!!errors.hubIds}
                                onChange={(options: OptionType[]) => {
                                    const v = options.map((option) => option.value)
                                    onChange(v)
                                    updateField('hubIds', v)
                                }}
                                options={linkedHubs}
                                value={value ? linkedHubs.filter((linkedHub: { value: string }) => value.includes(linkedHub.value)) : []}
                            />
                        )}
                    />
                </div>
            </FormGroup>

            <FormGroup error={errors.owner ? _(g.ownerID, { field: _(g.name) }) : undefined} id='owner'>
                <FormInput
                    {...register('owner', { required: true, validate: (val) => val !== '' })}
                    onBlur={(e) => updateField('owner', e.target.value)}
                    placeholder={_(g.ownerID)}
                />
            </FormGroup>

            <StepButtons
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
