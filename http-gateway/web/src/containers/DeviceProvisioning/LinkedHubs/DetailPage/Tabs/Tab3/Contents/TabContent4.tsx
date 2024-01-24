import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useFormContext } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import FormInput, { inputAligns } from '@shared-ui/components/Atomic/FormInput'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './TabContent4.types'

const TabContent4: FC<Props> = (props) => {
    const { loading } = props
    const { formatMessage: _ } = useIntl()
    const {
        control,
        formState: { errors },
        register,
        watch,
    } = useFormContext()

    const timeoutN = watch('authorization.provider.http.timeout')
    const idleConnTimeout = watch('authorization.provider.http.idleConnTimeout')

    return (
        <div>
            <Headline type='h5'>{_(t.hTTP)}</Headline>
            <Loadable condition={!loading}>
                <Spacer type='pt-4'>
                    <SimpleStripTable
                        rows={[
                            {
                                attribute: _(t.maxIdleConnections),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.maxIdleConnections) }) : undefined}
                                        id='authorization.provider.http.maxIdleConns'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.maxIdleConnections)}
                                            type='number'
                                            {...register('authorization.provider.http.maxIdleConns', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.maxConnectionsPerHost),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.maxConnectionsPerHost) }) : undefined}
                                        id='authorization.provider.http.maxConnsPerHost'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.maxConnectionsPerHost)}
                                            type='number'
                                            {...register('authorization.provider.http.maxConnsPerHost', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.maxIdleConnectionsPerHost),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.maxIdleConnectionsPerHost) }) : undefined}
                                        id='authorization.provider.http.maxIdleConnsPerHost'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.maxIdleConnectionsPerHost)}
                                            type='number'
                                            {...register('authorization.provider.http.maxIdleConnsPerHost', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
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
                                ) : (
                                    ''
                                ),
                            },
                        ]}
                    />
                </Spacer>
            </Loadable>
        </div>
    )
}

TabContent4.displayName = 'TabContent4'

export default TabContent4
