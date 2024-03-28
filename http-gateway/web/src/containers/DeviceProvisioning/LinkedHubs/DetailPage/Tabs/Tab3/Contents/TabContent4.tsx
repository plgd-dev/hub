import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent4.types'

const TabContent4: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, commonFormGroupProps, commonInputProps, commonTimeoutControlProps } = useContext(FormContext)

    const {
        formState: { errors },
        register,
        watch,
        control,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'tab3Content4' })

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
                                        {...commonFormGroupProps}
                                        error={
                                            errors?.authorization?.provider?.http?.maxIdleConns
                                                ? _(g.requiredField, { field: _(t.maxIdleConnections) })
                                                : undefined
                                        }
                                        id='authorization.provider.http.maxIdleConns'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.provider.http.maxIdleConns', {
                                                required: true,
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
                                        {...commonFormGroupProps}
                                        error={
                                            errors?.authorization?.provider?.http?.maxConnsPerHost
                                                ? _(g.requiredField, { field: _(t.maxConnectionsPerHost) })
                                                : undefined
                                        }
                                        id='authorization.provider.http.maxConnsPerHost'
                                    >
                                        <FormInput
                                            {...commonInputProps}
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
                                        {...commonFormGroupProps}
                                        error={
                                            errors?.authorization?.provider?.http?.maxIdleConnsPerHost
                                                ? _(g.requiredField, { field: _(t.maxIdleConnectionsPerHost) })
                                                : undefined
                                        }
                                        id='authorization.provider.http.maxIdleConnsPerHost'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.provider.http.maxIdleConnsPerHost', {
                                                required: true,
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
                                value: idleConnTimeout ? (
                                    <Controller
                                        control={control}
                                        name='authorization.provider.http.idleConnTimeout'
                                        render={({ field: { onChange, value } }) => (
                                            <TimeoutControl
                                                {...commonTimeoutControlProps}
                                                defaultTtlValue={parseInt(value, 10)}
                                                defaultValue={parseInt(value, 10)}
                                                onChange={(v) => onChange(v.toString())}
                                            />
                                        )}
                                    />
                                ) : (
                                    ''
                                ),
                            },
                            {
                                attribute: _(t.timeout),
                                value: timeoutN ? (
                                    <Controller
                                        control={control}
                                        name='authorization.provider.http.timeout'
                                        render={({ field: { onChange, value } }) => (
                                            <TimeoutControl
                                                {...commonTimeoutControlProps}
                                                defaultTtlValue={parseInt(value, 10)}
                                                defaultValue={parseInt(value, 10)}
                                                onChange={(v) => onChange(v.toString())}
                                            />
                                        )}
                                    />
                                ) : (
                                    ''
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
