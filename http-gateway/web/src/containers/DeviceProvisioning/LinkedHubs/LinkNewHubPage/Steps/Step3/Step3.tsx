import React, { FC, useContext, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'
import get from 'lodash/get'

import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { Props, Inputs } from './Step3.types'
import SubStepTls from '../SubStepTls'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { isTlsPageValid, useValidationsSchema } from '@/containers/DeviceProvisioning/LinkedHubs/validationSchema'

const Step3: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const schema = useValidationsSchema('group2')

    const {
        formState: { errors, isValid },
        register,
        control,
        updateField,
        watch,
        setValue,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step3', schema })

    useEffect(() => {
        const time = 'certificateAuthority.grpc.keepAlive.time'
        const timeout = 'certificateAuthority.grpc.keepAlive.timeout'

        if (!get(defaultFormData, time)) {
            setValue(time, get(DEFAULT_FORM_DATA, time))
            updateField(time, get(DEFAULT_FORM_DATA, time))
        }
        if (!get(defaultFormData, timeout)) {
            setValue(timeout, get(DEFAULT_FORM_DATA, timeout))
            updateField(timeout, get(DEFAULT_FORM_DATA, timeout))
        }
    }, [defaultFormData, setValue, updateField])

    const useSystemCaPool = watch('certificateAuthority.grpc.tls.useSystemCaPool')
    const caPool = watch('certificateAuthority.grpc.tls.caPool')
    const key = watch('certificateAuthority.grpc.tls.key')
    const cert = watch('certificateAuthority.grpc.tls.cert')

    const isFormValid = useMemo(() => isTlsPageValid(useSystemCaPool, isValid, caPool, key, cert), [useSystemCaPool, isValid, caPool, key, cert])

    return (
        <form>
            <FullPageWizard.Headline>{_(t.deviceAuthentication)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(t.addLinkedHubCertificateAuthorityDescription)}</FullPageWizard.Description>

            <FormGroup
                error={errors.certificateAuthority?.grpc?.address ? errors.certificateAuthority?.grpc?.address.message : undefined}
                id='certificateAuthority.grpc.address'
            >
                <FormLabel required text={_(t.address)} />
                <FormInput
                    {...register('certificateAuthority.grpc.address')}
                    onBlur={(e) => updateField('certificateAuthority.grpc.address', e.target.value)}
                />
            </FormGroup>

            <FullPageWizard.ToggleConfiguration
                i18n={{
                    hide: _(g.hideAdvancedConfiguration),
                    show: _(g.showAdvancedConfiguration),
                }}
            >
                <FullPageWizard.GroupHeadline tooltipText={_(t.addLinkedHubCertificateAuthorityKeepAliveDescription)}>
                    {_(t.connectionKeepAlive)}
                </FullPageWizard.GroupHeadline>

                <Controller
                    control={control}
                    name='certificateAuthority.grpc.keepAlive.time'
                    render={({ field: { onChange, value } }) => (
                        <TimeoutControl
                            required
                            watchUnitChange
                            align='left'
                            defaultTtlValue={parseInt(value, 10)}
                            defaultValue={parseInt(value, 10)}
                            error={errors.certificateAuthority?.grpc?.keepAlive?.time?.message}
                            i18n={{
                                default: _(g.default),
                                duration: _(g.time),
                                unit: _(g.metric),
                                placeholder: '',
                            }}
                            onBlur={(v) => updateField('certificateAuthority.grpc.keepAlive.time', v)}
                            onChange={(v) => onChange(v.toString())}
                            rightStyle={{
                                width: 150,
                            }}
                        />
                    )}
                />

                <Spacer type='pt-5'>
                    <Controller
                        control={control}
                        name='certificateAuthority.grpc.keepAlive.timeout'
                        render={({ field: { onChange, value } }) => (
                            <TimeoutControl
                                required
                                watchUnitChange
                                align='left'
                                defaultTtlValue={parseInt(value, 10)}
                                defaultValue={parseInt(value, 10)}
                                error={errors.certificateAuthority?.grpc?.keepAlive?.timeout?.message}
                                i18n={{
                                    default: _(g.default),
                                    duration: _(g.timeout),
                                    unit: _(g.metric),
                                    placeholder: '',
                                }}
                                onBlur={(v) => updateField('certificateAuthority.grpc.keepAlive.timeout', v)}
                                onChange={(v) => onChange(v.toString())}
                                rightStyle={{
                                    width: 150,
                                }}
                            />
                        )}
                    />
                </Spacer>

                <Spacer type='pt-5'>
                    <Controller
                        control={control}
                        name='certificateAuthority.grpc.keepAlive.permitWithoutStream'
                        render={({ field: { onChange, value } }) => (
                            <TileToggle
                                darkBg
                                checked={(value as boolean) ?? false}
                                name={_(t.permitWithoutStream)}
                                onChange={(e) => {
                                    updateField('certificateAuthority.grpc.keepAlive.permitWithoutStream', e.target.value === 'on')
                                    onChange(e)
                                }}
                            />
                        )}
                    />
                </Spacer>
            </FullPageWizard.ToggleConfiguration>

            <SubStepTls
                control={control}
                prefix='certificateAuthority.grpc.'
                setValue={(field: string, value: any) => {
                    // @ts-ignore
                    setValue(field, value)
                    updateField(field, value)
                }}
                updateField={updateField}
                watch={watch}
            />

            <StepButtons
                disableNext={!isFormValid}
                i18n={{
                    back: _(g.back),
                    continue: _(g.continue),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(1)}
                onClickNext={() => setStep?.(3)}
            />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
