import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'
import get from 'lodash/get'

import Headline from '@shared-ui/components/Atomic/Headline'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import { useForm } from '@shared-ui/common/hooks'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { Props, Inputs } from './TabContent4.types'
import { useValidationsSchema } from '@/containers/DeviceProvisioning/LinkedHubs/validationSchema'
import { messages as g } from '@/containers/Global.i18n'

const TabContent4: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    const { formatMessage: _ } = useIntl()
    const schema = useValidationsSchema('group3')

    const {
        formState: { errors },
        register,
        watch,
        control,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'tab3Content4', schema })

    const timeoutN = watch('authorization.provider.http.timeout')
    const idleConnTimeout = watch('authorization.provider.http.idleConnTimeout')

    return (
        <form>
            <Headline type='h5'>{_(t.hTTP)}</Headline>
            <Loadable condition={!loading}>
                <Spacer type='pt-4'>
                    <SimpleStripTable
                        leftColSize={4}
                        rightColSize={8}
                        rows={[
                            {
                                attribute: _(t.maxIdleConnections),
                                value: (
                                    <FormGroup
                                        error={get(errors, 'authorization.provider.http.maxIdleConns.message')}
                                        id='authorization.provider.http.maxIdleConns'
                                    >
                                        <FormInput
                                            {...register('authorization.provider.http.maxIdleConns', {
                                                valueAsNumber: true,
                                            })}
                                            placeholder={_(t.maxIdleConnections)}
                                            type='number'
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.maxConnectionsPerHost),
                                value: (
                                    <FormGroup
                                        error={get(errors, 'authorization.provider.http.maxConnsPerHost.message')}
                                        id='authorization.provider.http.maxConnsPerHost'
                                    >
                                        <FormInput
                                            {...register('authorization.provider.http.maxConnsPerHost', {
                                                required: true,
                                                valueAsNumber: true,
                                            })}
                                            placeholder={_(t.maxConnectionsPerHost)}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.maxIdleConnectionsPerHost),
                                value: (
                                    <FormGroup
                                        error={get(errors, 'authorization.provider.http.maxIdleConnsPerHost.message')}
                                        id='authorization.provider.http.maxIdleConnsPerHost'
                                    >
                                        <FormInput
                                            {...register('authorization.provider.http.maxIdleConnsPerHost', {
                                                valueAsNumber: true,
                                            })}
                                            placeholder={_(t.maxIdleConnectionsPerHost)}
                                            type='number'
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.idleConnectionTimeout),
                                required: true,
                                value: (
                                    <Loadable condition={idleConnTimeout !== undefined}>
                                        <Controller
                                            control={control}
                                            name='authorization.provider.http.idleConnTimeout'
                                            render={({ field: { onChange, value } }) => (
                                                <TimeoutControl
                                                    required
                                                    defaultTtlValue={parseInt(value, 10)}
                                                    defaultValue={parseInt(value, 10)}
                                                    error={errors.authorization?.provider?.http?.timeout?.message}
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
                                            name='authorization.provider.http.timeout'
                                            render={({ field: { onChange, value } }) => (
                                                <TimeoutControl
                                                    required
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
                                                />
                                            )}
                                        />
                                    </Loadable>
                                ),
                            },
                        ]}
                    />
                </Spacer>
            </Loadable>
        </form>
    )
}

TabContent4.displayName = 'TabContent4'

export default TabContent4
