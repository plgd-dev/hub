import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useFieldArray } from 'react-hook-form'
import get from 'lodash/get'

import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { FormContext } from '@shared-ui/common/context/FormContext'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import Button from '@shared-ui/components/Atomic/Button'
import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'
import IconClose from '@shared-ui/components/Atomic/Icon/components/IconClose'
import { convertSize } from '@shared-ui/components/Atomic/Icon'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Inputs, Props } from './Step4.types'
import SubStepTls from '../SubStepTls'
import { isTlsPageValid, useValidationsSchema } from '@/containers/DeviceProvisioning/LinkedHubs/validationSchema'
import * as styles from '@/containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage/Steps/Step2/Step2.styles'

const Step4: FC<Props> = (props) => {
    const { defaultFormData, onSubmit } = props
    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const schema = useValidationsSchema('group3')

    const {
        formState: { errors, isValid },
        register,
        control,
        updateField,
        watch,
        setValue,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step4', schema })

    const useSystemCaPool = watch('authorization.provider.http.tls.useSystemCaPool')
    const caPool = watch('authorization.provider.http.tls.caPool')
    const key = watch('authorization.provider.http.tls.key')
    const cert = watch('authorization.provider.http.tls.cert')
    const scopes = watch('authorization.provider.scopes')

    const { fields, append, remove } = useFieldArray({
        control,
        name: 'authorization.provider.scopes',
        shouldUnregister: true,
    })

    const isFormValid = isTlsPageValid(useSystemCaPool, isValid, caPool, key, cert) && scopes.every((i) => i.value !== '')

    return (
        <form>
            <FullPageWizard.Headline>{_(t.authorization)}</FullPageWizard.Headline>

            <FullPageWizard.SubHeadline noBorder>{_(t.general)}</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>{_(t.addLinkedHubAuthorizationGeneralDescription)}</FullPageWizard.Description>

            <FormGroup error={get(errors, 'authorization.ownerClaim.message')} id='authorization.ownerClaim'>
                <FormLabel required text={_(t.ownerClaim)} />
                <FormInput {...register('authorization.ownerClaim')} onBlur={(e) => updateField('authorization.ownerClaim', e.target.value)} />
            </FormGroup>

            <FullPageWizard.SubHeadline>{_(t.oAuthClient)}</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>{_(t.addLinkedHubAuthorizationoAuthClientDescription)}</FullPageWizard.Description>

            <FormGroup error={get(errors, 'authorization.provider.name.message')} id='authorization.provider.name'>
                <FormLabel required text={_(t.deviceProviderName)} tooltipText={_(t.deviceProviderNameTooltip)} />
                <FormInput {...register('authorization.provider.name')} onBlur={(e) => updateField('authorization.provider.name', e.target.value)} />
            </FormGroup>

            <FormGroup error={get(errors, 'authorization.provider.clientId.message')} id='authorization.provider.clientId'>
                <FormLabel required text={_(t.clientId)} />
                <FormInput {...register('authorization.provider.clientId')} onBlur={(e) => updateField('authorization.provider.clientId', e.target.value)} />
            </FormGroup>

            <FormGroup error={get(errors, 'authorization.provider.clientSecret.message')} id='authorization.provider.clientSecret'>
                <FormLabel required text={_(t.clientSecret)} />
                <FormInput
                    {...register('authorization.provider.clientSecret')}
                    onBlur={(e) => updateField('authorization.provider.clientSecret', e.target.value)}
                />
            </FormGroup>

            <FormGroup error={get(errors, 'authorization.provider.authority.message')} id='authorization.provider.authority'>
                <FormLabel required text={_(t.authority)} />
                <FormInput {...register('authorization.provider.authority')} onBlur={(e) => updateField('authorization.provider.authority', e.target.value)} />
            </FormGroup>

            {fields?.map((field, index) => (
                <FormGroup
                    error={errors.authorization?.provider?.scopes?.[index] ? _(t.deviceGateway, { field: _(t.deviceGateway) }) : undefined}
                    id={`gateways.${index}`}
                    key={field.id}
                >
                    {index === 0 && <FormLabel required text={_(t.scopes)} />}
                    <Controller
                        control={control}
                        name={`authorization.provider.scopes.${index}` as any}
                        render={({ field: { onChange, value } }) => (
                            <div css={styles.flex}>
                                <FormInput
                                    onBlur={(e) => updateField(`authorization.provider.scopes.${index}`, { value: e.target.value, id: field.id }, true)}
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
                                            'authorization.provider.scopes',
                                            scopes.filter((_, key) => key !== index)
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

            <Button
                disabled={scopes && scopes[scopes.length - 1]?.value === ''}
                icon={<IconPlus />}
                onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()

                    append({ value: '' })
                }}
                size='small'
                variant='filter'
            >
                {_(t.addScope)}
            </Button>

            <SubStepTls
                control={control}
                prefix='authorization.provider.http.'
                setValue={(field: string, value: any) => {
                    // @ts-ignore
                    setValue(field, value)
                    updateField(field, value)
                }}
                updateField={updateField}
                watch={watch}
            />

            <Spacer type='pt-12'>
                <FullPageWizard.SubHeadline>{_(t.httpClient)}</FullPageWizard.SubHeadline>
                <FullPageWizard.Description>{_(t.addLinkedHubAuthorizationHttpDescription)}</FullPageWizard.Description>
            </Spacer>

            <FullPageWizard.ToggleConfiguration
                i18n={{
                    hide: _(g.hideAdvancedConfiguration),
                    show: _(g.showAdvancedConfiguration),
                }}
            >
                <FormGroup
                    error={errors?.authorization?.provider?.http?.maxIdleConns ? _(g.requiredField, { field: _(t.maxIdleConnections) }) : undefined}
                    id='authorization.provider.http.maxIdleConns'
                >
                    <FormLabel text={_(t.maxIdleConnections)} />
                    <FormInput
                        {...register('authorization.provider.http.maxIdleConns', {
                            valueAsNumber: true,
                        })}
                        type='number'
                    />
                </FormGroup>

                <FormGroup
                    error={errors?.authorization?.provider?.http?.maxConnsPerHost ? _(g.requiredField, { field: _(t.maxConnectionsPerHost) }) : undefined}
                    id='authorization.provider.http.maxConnsPerHost'
                >
                    <FormLabel text={_(t.maxConnectionsPerHost)} />
                    <FormInput
                        {...register('authorization.provider.http.maxConnsPerHost', {
                            valueAsNumber: true,
                        })}
                        type='number'
                    />
                </FormGroup>

                <FormGroup
                    error={
                        errors?.authorization?.provider?.http?.maxIdleConnsPerHost ? _(g.requiredField, { field: _(t.maxIdleConnectionsPerHost) }) : undefined
                    }
                    id='authorization.provider.http.maxIdleConnsPerHost'
                >
                    <FormLabel text={_(t.maxIdleConnectionsPerHost)} />
                    <FormInput
                        {...register('authorization.provider.http.maxIdleConnsPerHost', {
                            valueAsNumber: true,
                        })}
                        type='number'
                    />
                </FormGroup>

                <Controller
                    control={control}
                    name='authorization.provider.http.idleConnTimeout'
                    render={({ field: { onChange, value } }) => (
                        <TimeoutControl
                            required
                            watchUnitChange
                            align='left'
                            defaultTtlValue={parseInt(value, 10)}
                            defaultValue={parseInt(value, 10)}
                            error={errors.authorization?.provider?.http?.idleConnTimeout?.message}
                            i18n={{
                                default: _(g.default),
                                duration: _(t.idleConnectionTimeout),
                                unit: _(g.metric),
                                placeholder: _(t.idleConnectionTimeout),
                            }}
                            onChange={(v) => onChange(v.toString())}
                            rightStyle={{
                                width: 150,
                            }}
                        />
                    )}
                />

                <Spacer type='mt-5'>
                    <Controller
                        control={control}
                        name='authorization.provider.http.timeout'
                        render={({ field: { onChange, value } }) => (
                            <TimeoutControl
                                required
                                watchUnitChange
                                align='left'
                                defaultTtlValue={parseInt(value, 10)}
                                defaultValue={parseInt(value, 10)}
                                error={errors.authorization?.provider?.http?.timeout?.message}
                                i18n={{
                                    default: _(g.default),
                                    duration: _(g.timeout),
                                    unit: _(g.metric),
                                    placeholder: _(g.timeout),
                                }}
                                onChange={(v) => onChange(v.toString())}
                                rightStyle={{
                                    width: 150,
                                }}
                            />
                        )}
                    />
                </Spacer>
            </FullPageWizard.ToggleConfiguration>

            <StepButtons
                disableNext={!isFormValid}
                i18n={{
                    back: _(g.back),
                    continue: _(g.create),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(2)}
                onClickNext={() => onSubmit?.()}
            />
        </form>
    )
}

Step4.displayName = 'Step4'

export default Step4
