import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'
import { z } from 'zod'
import get from 'lodash/get'

import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { FormContext } from '@shared-ui/common/context/FormContext'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import * as commonStyles from '@shared-ui/components/Templates/FullPageWizard/FullPageWizardCommon.styles'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Inputs, Props } from './Step4.types'
import SubStepTls from '../SubStepTls'

const Step4: FC<Props> = (props) => {
    const { defaultFormData, onSubmit } = props
    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, setStep } = useContext(FormContext)

    const schema = useMemo(
        () =>
            z.object({
                authorization: z.object({
                    ownerClaim: z.string().min(1, { message: _(g.requiredField, { field: _(t.ownerClaim) }) }),
                    provider: z.object({
                        name: z.string().min(1, { message: _(g.requiredField, { field: _(t.deviceProviderName) }) }),
                        clientId: z.string().min(1, { message: _(g.requiredField, { field: _(t.clientId) }) }),
                        clientSecret: z.string().min(1, { message: _(g.requiredField, { field: _(t.clientSecret) }) }),
                        authority: z.string().min(1, { message: _(g.requiredField, { field: _(t.authority) }) }),
                        http: z.object({
                            idleConnTimeout: z.number().min(1, { message: _(g.requiredField, { field: _(t.idleConnectionTimeout) }) }),
                            timeout: z.number().min(1, { message: _(g.requiredField, { field: _(t.timeout) }) }),
                        }),
                    }),
                }),
            }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const {
        formState: { errors, isValid },
        register,
        control,
        updateField,
        watch,
        setValue,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step4', schema })

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.authorization)}</h1>

            <FullPageWizard.SubHeadline noBorder>{_(t.general)}</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>{_(t.addLinkedHubAuthorizationGeneralDescription)}</FullPageWizard.Description>

            <FormGroup error={get(errors, 'authorization.ownerClaim.message')} id='authorization.ownerClaim'>
                <FormLabel required text={_(t.ownerClaim)} />
                <FormInput {...register('authorization.ownerClaim')} onBlur={(e) => updateField('authorization.ownerClaim', e.target.value)} />
            </FormGroup>

            <h2 css={commonStyles.subHeadline}>{_(t.oAuthClient)}</h2>
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
            <Controller
                control={control}
                name='authorization.provider.scopes'
                render={({ field: { onChange, value } }) => (
                    <FormGroup
                        error={errors.authorization?.provider?.authority ? _(g.requiredField, { field: _(t.scopes) }) : undefined}
                        id='authorization.provider.scopes'
                    >
                        <FormLabel text={_(t.scopes)} />
                        <FormInput
                            onBlur={(e) => updateField('authorization.provider.scopes', e.target.value.split(' '))}
                            onChange={(e) => onChange(e.target.value.split(' '))}
                            value={Array.isArray(value) ? value.join(' ') : value}
                        />
                    </FormGroup>
                )}
            />

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
                <FullPageWizard.SubHeadline>{_(t.hTTP)}</FullPageWizard.SubHeadline>
                <FullPageWizard.Description>{_(t.addLinkedHubAuthorizationHttpDescription)}</FullPageWizard.Description>
            </Spacer>

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
                error={errors?.authorization?.provider?.http?.maxIdleConnsPerHost ? _(g.requiredField, { field: _(t.maxIdleConnectionsPerHost) }) : undefined}
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
                            placeholder: _(g.placeholder),
                        }}
                        onChange={(v) => onChange(v)}
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
                                placeholder: _(g.placeholder),
                            }}
                            onChange={(v) => onChange(v)}
                            rightStyle={{
                                width: 150,
                            }}
                        />
                    )}
                />
            </Spacer>

            <StepButtons
                disableNext={!isValid}
                i18n={{
                    back: _(g.back),
                    continue: _(g.create),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(3)}
                onClickNext={() => onSubmit?.()}
            />
        </form>
    )
}

Step4.displayName = 'Step4'

export default Step4
