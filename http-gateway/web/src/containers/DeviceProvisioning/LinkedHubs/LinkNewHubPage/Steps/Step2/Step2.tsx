import React, { FC, useContext, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useFieldArray } from 'react-hook-form'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'
import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'
import Button from '@shared-ui/components/Atomic/Button'
import IconClose from '@shared-ui/components/Atomic/Icon/components/IconClose'
import { convertSize } from '@shared-ui/components/Atomic'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import Show from '@shared-ui/components/Atomic/Show'
import ValidationMessage from '@shared-ui/components/Atomic/ValidationMessage'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { Inputs, Props } from './Step2.types'
import * as styles from './Step2.styles'
import { useValidationsSchema } from '../../../validationSchema'

const Step2: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const schema = useValidationsSchema('group1')

    const {
        formState: { errors, isValid },
        register,
        control,
        updateField,
        watch,
        trigger,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step2', schema })

    const { fields, append, remove } = useFieldArray({
        control,
        name: 'gateways',
        shouldUnregister: true,
    })

    const gateways = watch('gateways')

    useEffect(() => {
        const validationResult = schema.safeParse(defaultFormData)

        if (!validationResult.success) {
            if (defaultFormData.hubId) {
                trigger('hubId')
            }
            if (defaultFormData.name) {
                trigger('name')
            }
        }
    }, [defaultFormData, schema, trigger])

    return (
        <form>
            <FullPageWizard.Headline>{_(t.hubDetails)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(t.addLinkedHubDetailsDescription)}</FullPageWizard.Description>

            <FormGroup error={errors.hubId ? errors.hubId.message : undefined} id='hubID'>
                <FormLabel required text={_(g.hubId)} />
                <FormInput {...register('hubId')} onBlur={(e) => updateField('hubId', e.target.value)} />
            </FormGroup>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                <FormLabel required text={_(g.name)} />
                <FormInput {...register('name')} onBlur={(e) => updateField('name', e.target.value)} />
            </FormGroup>

            <Show>
                <Show.When isTrue={fields.length > 0}>
                    {fields?.map((field, index) => (
                        <FormGroup
                            error={errors.gateways?.[index] ? _(t.deviceGateway, { field: _(t.deviceGateway) }) : undefined}
                            id={`gateways.${index}`}
                            key={field.id}
                        >
                            <FormLabel required text={_(t.deviceGateway)} />
                            <Controller
                                control={control}
                                name={`gateways.${index}` as any}
                                render={({ field: { onChange, value } }) => (
                                    <div css={styles.flex}>
                                        <FormInput
                                            onBlur={(e) => updateField(`gateways.${index}`, { value: e.target.value, id: field.id }, true)}
                                            onChange={(v) => {
                                                onChange({ value: v.target.value, id: field.id })
                                            }}
                                            value={value.value}
                                        />
                                        <a
                                            css={styles.removeIcon}
                                            href='#'
                                            onClick={(e) => {
                                                e.preventDefault()
                                                e.stopPropagation()
                                                remove(index)

                                                updateField(
                                                    'gateways',
                                                    gateways.filter((_, key) => key !== index)
                                                )
                                            }}
                                        >
                                            <IconClose {...convertSize(20)} />
                                        </a>
                                    </div>
                                )}
                                rules={{
                                    required: true,
                                    validate: (val) => val !== '',
                                }}
                            />
                        </FormGroup>
                    ))}
                </Show.When>
                <Show.Else>
                    <FormGroup id='gateways' marginBottom={false}>
                        <FormLabel text={_(t.deviceGateway)} />
                        <ValidationMessage>{_(t.deviceGatewayEmptyError)}</ValidationMessage>
                    </FormGroup>
                </Show.Else>
            </Show>

            <div css={styles.addButton}>
                <Button
                    disabled={defaultFormData.gateways && defaultFormData.gateways[defaultFormData.gateways.length - 1]?.value === ''}
                    icon={<IconPlus />}
                    onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()

                        append({ value: '' })
                    }}
                    size='small'
                    variant='filter'
                >
                    {_(t.addDeviceGateway)}
                </Button>
            </div>

            <StepButtons
                disableNext={!isValid || fields.length === 0 || gateways.some((f) => f.value === '')}
                i18n={{
                    back: _(g.back),
                    continue: _(g.continue),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(0)}
                onClickNext={() => setStep?.(2)}
            />
        </form>
    )
}

Step2.displayName = 'Step2'

export default Step2
