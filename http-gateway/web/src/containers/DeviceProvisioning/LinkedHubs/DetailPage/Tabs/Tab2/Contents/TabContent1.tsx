import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Switch from '@shared-ui/components/Atomic/Switch'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Loadable from '@shared-ui/components/Atomic/Loadable/Loadable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'

import { messages as t } from '../../../../LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent1.types'

const TabContent1: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, commonTimeoutControlProps, commonInputProps, commonFormGroupProps } = useContext(FormContext)

    const {
        formState: { errors },
        register,
        watch,
        control,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        errorKey: 'tab2Content1',
    })

    const time = watch('certificateAuthority.grpc.keepAlive.time')
    const timeoutN = watch('certificateAuthority.grpc.keepAlive.timeout')
    const permitWithoutStream = watch('certificateAuthority.grpc.keepAlive.permitWithoutStream')

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
                                        {...commonFormGroupProps}
                                        error={errors.certificateAuthority?.grpc?.address ? _(g.requiredField, { field: _(g.name) }) : undefined}
                                        id='certificateAuthority.grpc.address'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('certificateAuthority.grpc.address', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                            placeholder={_(g.name)}
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
                            leftColSize={5}
                            rightColSize={7}
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
                                                        {...commonTimeoutControlProps}
                                                        defaultTtlValue={parseInt(value, 10)}
                                                        defaultValue={parseInt(value, 10)}
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
                                                        {...commonTimeoutControlProps}
                                                        defaultTtlValue={parseInt(value, 10)}
                                                        defaultValue={parseInt(value, 10)}
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
