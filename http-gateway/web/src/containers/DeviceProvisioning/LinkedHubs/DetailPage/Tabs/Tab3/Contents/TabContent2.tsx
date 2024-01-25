import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import { useFormContext } from 'react-hook-form'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput, { inputAligns } from '@shared-ui/components/Atomic/FormInput'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './TabContent2.types'

const TabContent2: FC<Props> = (props) => {
    const { loading } = props
    const { formatMessage: _ } = useIntl()
    const {
        formState: { errors },
        register,
    } = useFormContext()

    return (
        <div>
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
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined}
                                        id='authorization.provider.name'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(g.name)}
                                            {...register('authorization.provider.name', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.clientId),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.clientId) }) : undefined}
                                        id='authorization.provider.clientId'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.clientId)}
                                            {...register('authorization.provider.clientId', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.clientSecret),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.clientSecret) }) : undefined}
                                        id='authorization.provider.clientSecret'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.clientSecret)}
                                            {...register('authorization.provider.clientSecret', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.authority),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.authority) }) : undefined}
                                        id='authorization.provider.authority'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.authority)}
                                            {...register('authorization.provider.authority', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                ),
                            },
                        ]}
                    />
                </Loadable>
            </Spacer>
        </div>
    )
}

TabContent2.displayName = 'TabContent2'

export default TabContent2
