import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'
import get from 'lodash/get'

import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Switch from '@shared-ui/components/Atomic/Switch'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Loadable from '@shared-ui/components/Atomic/Loadable/Loadable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'

import { messages as t } from '../../../../LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent1.types'
import { useValidationsSchema } from '@/containers/DeviceProvisioning/LinkedHubs/validationSchema'

const TabContent1: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    const { formatMessage: _ } = useIntl()
    const schema = useValidationsSchema('group2')

    const {
        formState: { errors },
        register,
        watch,
        control,
        updateField,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab2Content1',
        schema,
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
                                required: true,
                                value: (
                                    <FormGroup error={get(errors, 'certificateAuthority.grpc.address.message')} id='certificateAuthority.grpc.address'>
                                        <FormInput
                                            {...register('certificateAuthority.grpc.address')}
                                            onBlur={(e) => updateField('certificateAuthority.grpc.address', e.target.value)}
                                            placeholder={_(t.address)}
                                        />
                                    </FormGroup>
                                ),
                            },
                        ]}
                    />
                </Spacer>
            </Loadable>
            <Spacer type='pt-8'>
                <Headline type='h5'>{_(t.connectionKeepAlive)}</Headline>
                <Spacer type='pt-4'>
                    <Loadable condition={!loading}>
                        <SimpleStripTable
                            leftColSize={5}
                            rightColSize={7}
                            rows={[
                                {
                                    attribute: _(t.time),
                                    required: true,
                                    value: (
                                        <Loadable condition={time !== undefined}>
                                            <Controller
                                                control={control}
                                                name='certificateAuthority.grpc.keepAlive.time'
                                                render={({ field: { onChange, value } }) => (
                                                    <TimeoutControl
                                                        required
                                                        defaultTtlValue={parseInt(value, 10)}
                                                        defaultValue={parseInt(value, 10)}
                                                        error={errors.certificateAuthority?.grpc?.keepAlive?.time?.message}
                                                        onChange={(v) => onChange(v.toString())}
                                                    />
                                                )}
                                            />
                                        </Loadable>
                                    ),
                                },
                                {
                                    attribute: _(t.timeout),
                                    required: true,
                                    value: (
                                        <Loadable condition={timeoutN !== undefined}>
                                            <Controller
                                                control={control}
                                                name='certificateAuthority.grpc.keepAlive.timeout'
                                                render={({ field: { onChange, value } }) => (
                                                    <TimeoutControl
                                                        required
                                                        defaultTtlValue={parseInt(value, 10)}
                                                        defaultValue={parseInt(value, 10)}
                                                        error={errors.certificateAuthority?.grpc?.keepAlive?.timeout?.message}
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
