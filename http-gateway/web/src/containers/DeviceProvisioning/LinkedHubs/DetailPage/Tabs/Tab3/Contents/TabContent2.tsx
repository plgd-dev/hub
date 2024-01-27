import React, { FC, useContext, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { useForm } from 'react-hook-form'
import isFunction from 'lodash/isFunction'
import cloneDeep from 'lodash/cloneDeep'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { setProperty } from '@shared-ui/components/Atomic/_utils/utils'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent2.types'

const TabContent2: FC<Props> = (props) => {
    const { defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()

    const {
        formState: { errors, isDirty },
        register,
        watch,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const { updateData, setFormError, commonFormGroupProps, commonInputProps } = useContext(FormContext)

    const name = watch('authorization.provider.name')
    const clientId = watch('authorization.provider.clientId')
    const clientSecret = watch('authorization.provider.clientSecret')
    const authority = watch('authorization.provider.authority')

    useEffect(() => {
        if (defaultFormData && isDirty) {
            const copy = cloneDeep(defaultFormData)

            if (defaultFormData.authorization.provider.name !== name) {
                updateData(setProperty(copy, 'authorization.provider.name', name))
            }

            if (defaultFormData.authorization.provider.clientId !== clientId) {
                updateData(setProperty(copy, 'authorization.provider.clientId', clientId))
            }

            if (defaultFormData.authorization.provider.clientId !== clientSecret) {
                updateData(setProperty(copy, 'authorization.provider.clientSecret', clientSecret))
            }

            if (defaultFormData.authorization.provider.clientId !== authority) {
                updateData(setProperty(copy, 'authorization.provider.authority', authority))
            }
        }
    }, [authority, clientId, clientSecret, defaultFormData, isDirty, name, updateData])

    useEffect(() => {
        isFunction(setFormError) && setFormError((prevState: any) => ({ ...prevState, tab3Content2: Object.keys(errors).length > 0 }))
    }, [errors, setFormError])

    return (
        <form>
            <Headline type='h5'>{_(t.oAuthClient)}</Headline>
            <Spacer type='pt-4'>
                <Loadable condition={!loading}>
                    <SimpleStripTable
                        leftColSize={5}
                        rightColSize={7}
                        rows={[
                            {
                                attribute: _(g.name),
                                value: (
                                    <FormGroup
                                        {...commonFormGroupProps}
                                        error={errors?.authorization?.provider?.name ? _(g.requiredField, { field: _(g.name) }) : undefined}
                                        id='authorization.provider.name'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.provider.name', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                            placeholder={_(g.name)}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.clientId),
                                value: (
                                    <FormGroup
                                        {...commonFormGroupProps}
                                        error={errors?.authorization?.provider?.clientId ? _(g.requiredField, { field: _(t.clientId) }) : undefined}
                                        id='authorization.provider.clientId'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.provider.clientId', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                            placeholder={_(t.clientId)}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.clientSecret),
                                value: (
                                    <FormGroup
                                        {...commonFormGroupProps}
                                        error={errors?.authorization?.provider?.clientSecret ? _(g.requiredField, { field: _(t.clientSecret) }) : undefined}
                                        id='authorization.provider.clientSecret'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.provider.clientSecret', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                            placeholder={_(t.clientSecret)}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.authority),
                                value: (
                                    <FormGroup
                                        {...commonFormGroupProps}
                                        error={errors?.authorization?.provider?.authority ? _(g.requiredField, { field: _(t.authority) }) : undefined}
                                        id='authorization.provider.authority'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.provider.authority', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                            placeholder={_(t.authority)}
                                        />
                                    </FormGroup>
                                ),
                            },
                        ]}
                    />
                </Loadable>
            </Spacer>
        </form>
    )
}

TabContent2.displayName = 'TabContent2'

export default TabContent2
