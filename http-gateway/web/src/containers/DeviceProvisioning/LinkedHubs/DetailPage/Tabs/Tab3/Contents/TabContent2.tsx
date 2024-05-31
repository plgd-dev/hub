import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import get from 'lodash/get'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent2.types'
import { useValidationsSchema } from '@/containers/DeviceProvisioning/LinkedHubs/validationSchema'

const TabContent2: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    const { formatMessage: _ } = useIntl()
    const schema = useValidationsSchema('group3')

    const {
        formState: { errors },
        register,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'tab3Content2', schema })

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
                                required: true,
                                value: (
                                    <FormGroup error={get(errors, 'authorization.provider.name.message')} id='authorization.provider.name'>
                                        <FormInput {...register('authorization.provider.name')} placeholder={_(g.name)} />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.clientId),
                                required: true,
                                value: (
                                    <FormGroup error={get(errors, 'authorization.provider.clientId.message')} id='authorization.provider.clientId'>
                                        <FormInput {...register('authorization.provider.clientId')} placeholder={_(t.clientId)} />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.clientSecret),
                                required: true,
                                value: (
                                    <FormGroup error={get(errors, 'authorization.provider.clientSecret.message')} id='authorization.provider.clientSecret'>
                                        <FormInput {...register('authorization.provider.clientSecret')} placeholder={_(t.clientSecret)} />
                                    </FormGroup>
                                ),
                            },
                            {
                                attribute: _(t.authority),
                                required: true,
                                value: (
                                    <FormGroup error={get(errors, 'authorization.provider.authority.message')} id='authorization.provider.authority'>
                                        <FormInput {...register('authorization.provider.authority')} placeholder={_(t.authority)} />
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
