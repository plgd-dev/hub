import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent2.types'

const TabContent2: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, commonFormGroupProps, commonInputProps } = useContext(FormContext)

    const {
        formState: { errors },
        register,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'tab3Content2' })

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
