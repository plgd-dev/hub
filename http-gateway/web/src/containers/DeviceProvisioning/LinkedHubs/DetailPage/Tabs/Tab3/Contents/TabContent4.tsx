import React, { FC, useContext, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useForm } from 'react-hook-form'
import cloneDeep from 'lodash/cloneDeep'

import Headline from '@shared-ui/components/Atomic/Headline'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import { setProperty } from '@shared-ui/components/Atomic/_utils/utils'
import { FormContext } from '@shared-ui/common/context/FormContext'
import isFunction from 'lodash/isFunction'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent4.types'

const TabContent4: FC<Props> = (props) => {
    const { defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()

    const {
        formState: { errors, isDirty },
        register,
        watch,
        control,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const { updateData, setFormError, commonFormGroupProps, commonInputProps, commonTimeoutControlProps } = useContext(FormContext)

    const maxIdleConns = watch('authorization.provider.http.maxIdleConns')
    const maxConnsPerHost = watch('authorization.provider.http.maxConnsPerHost')
    const maxIdleConnsPerHost = watch('authorization.provider.http.maxIdleConnsPerHost')
    const timeoutN = watch('authorization.provider.http.timeout')
    const idleConnTimeout = watch('authorization.provider.http.idleConnTimeout')

    useEffect(() => {
        if (defaultFormData && isDirty) {
            const copy = cloneDeep(defaultFormData)

            if (defaultFormData.authorization.provider.http.maxIdleConns !== maxIdleConns) {
                updateData(setProperty(copy, 'authorization.provider.http.maxIdleConns', maxIdleConns))
            }

            if (defaultFormData.authorization.provider.http.maxConnsPerHost !== maxConnsPerHost) {
                updateData(setProperty(copy, 'authorization.provider.http.maxConnsPerHost', maxConnsPerHost))
            }

            if (defaultFormData.authorization.provider.http.maxIdleConnsPerHost !== maxIdleConnsPerHost) {
                updateData(setProperty(copy, 'authorization.provider.http.maxIdleConnsPerHost', maxIdleConnsPerHost))
            }

            if (defaultFormData.authorization.provider.http.timeout !== timeoutN) {
                updateData(setProperty(copy, 'authorization.provider.http.timeout', timeoutN))
            }
        }
    }, [defaultFormData, isDirty, maxConnsPerHost, maxIdleConns, maxIdleConnsPerHost, timeoutN, updateData])

    useEffect(() => {
        isFunction(setFormError) && setFormError((prevState: any) => ({ ...prevState, tab3Content4: Object.keys(errors).length > 0 }))
    }, [errors, setFormError])

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
