import React, { FC, useContext, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useForm } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Switch from '@shared-ui/components/Atomic/Switch'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Loadable from '@shared-ui/components/Atomic/Loadable/Loadable'

import { messages as t } from '../../../../LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent1.types'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput, { inputAligns } from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'

const TabContent1: FC<Props> = (props) => {
    const { defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()

    const { onSubmit } = useContext(FormContext)

    const {
        formState: { errors, isDirty, touchedFields, dirtyFields, defaultValues },
        register,
        handleSubmit,
        watch,
        control,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const time = watch('certificateAuthority.grpc.keepAlive.time')
    const timeoutN = watch('certificateAuthority.grpc.keepAlive.timeout')
    const permitWithoutStream = watch('certificateAuthority.grpc.keepAlive.permitWithoutStream')

    useEffect(() => {
        if (isDirty) {
            if (defaultFormData?.certificateAuthority.grpc.keepAlive.time !== defaultValues?.certificateAuthority?.grpc?.keepAlive?.time) {
                console.log('DIFF')
            }
        }
    }, [defaultFormData?.certificateAuthority.grpc, defaultValues?.certificateAuthority?.grpc?.keepAlive?.time, isDirty, time])

    console.log({ isDirty })
    console.log({ touchedFields })
    console.log({ dirtyFields })
    console.log(defaultValues)

    return (
        <form>
            <Headline type='h5'>{_(t.general)}</Headline>
            <Loadable condition={!loading}>
                <Spacer type='pt-4'>
                    <SimpleStripTable
                        rows={[
                            {
                                attribute: _(t.address),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.certificateAuthority?.grpc?.address ? _(g.requiredField, { field: _(g.name) }) : undefined}
                                        id='certificateAuthority.grpc.address'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(g.name)}
                                            {...register('certificateAuthority.grpc.address', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                        ]}
                    />
                </Spacer>
            </Loadable>
            <Spacer type='pt-8'>
                <Headline type='h5'>{_(t.keepAlive)}</Headline>
                <Spacer type='pt-4'>
                    <Loadable condition={!loading}>
                        <SimpleStripTable
                            leftColSize={4}
                            rightColSize={8}
                            rows={[
                                {
                                    attribute: _(t.time),
                                    value: (
                                        <Loadable condition={time !== undefined}>
                                            <Controller
                                                control={control}
                                                name='certificateAuthority.grpc.keepAlive.time'
                                                render={({ field: { onChange, value } }) => (
                                                    <TimeoutControl
                                                        inlineStyle
                                                        smallMode
                                                        watchUnitChange
                                                        align='right'
                                                        defaultTtlValue={parseInt(value, 10)}
                                                        defaultValue={parseInt(value, 10)}
                                                        i18n={{
                                                            default: '',
                                                            duration: '',
                                                            placeholder: '',
                                                            unit: '',
                                                        }}
                                                        onChange={(v) => onChange(v.toString())}
                                                    />
                                                )}
                                            />
                                        </Loadable>
                                    ),
                                },
                                {
                                    attribute: _(t.timeout),
                                    value: (
                                        <Loadable condition={timeoutN !== undefined}>
                                            <Controller
                                                control={control}
                                                name='certificateAuthority.grpc.keepAlive.timeout'
                                                render={({ field: { onChange, value } }) => (
                                                    <TimeoutControl
                                                        inlineStyle
                                                        smallMode
                                                        watchUnitChange
                                                        align='right'
                                                        defaultTtlValue={parseInt(value, 10)}
                                                        defaultValue={parseInt(value, 10)}
                                                        i18n={{
                                                            default: '',
                                                            duration: '',
                                                            placeholder: '',
                                                            unit: '',
                                                        }}
                                                        onChange={(v) => onChange(v.toString())}
                                                    />
                                                )}
                                            />
                                        </Loadable>
                                    ),
                                },
                                {
                                    attribute: _(t.permitWithoutStream),
                                    value: (
                                        <Loadable condition={permitWithoutStream !== undefined}>
                                            <Controller
                                                control={control}
                                                name='certificateAuthority.grpc.keepAlive.permitWithoutStream'
                                                render={({ field: { onChange, value } }) => (
                                                    <Switch labelBefore checked={value} label={permitWithoutStream ? _(g.on) : _(g.off)} onChange={onChange} />
                                                )}
                                            />
                                        </Loadable>
                                    ),
                                },
                            ]}
                        />
                    </Loadable>
                </Spacer>
            </Spacer>
        </form>
    )
}

TabContent1.displayName = 'TabContent1'

export default TabContent1
