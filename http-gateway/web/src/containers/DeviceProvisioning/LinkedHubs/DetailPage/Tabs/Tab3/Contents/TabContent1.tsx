import React, { FC } from 'react'
import { useFormContext } from 'react-hook-form'
import { useIntl } from 'react-intl'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormInput, { inputAligns } from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './TabContent1.types'

const TabContent1: FC<Props> = (props) => {
    const { loading } = props
    const { formatMessage: _ } = useIntl()
    const {
        formState: { errors },
        register,
    } = useFormContext()

    return (
        <div>
            <Headline type='h5'>{_(t.general)}</Headline>
            <Spacer type='pt-4'>
                <Loadable condition={!loading}>
                    <SimpleStripTable
                        rows={[
                            {
                                attribute: _(t.ownerClaim),
                                value: (
                                    <FormGroup
                                        errorTooltip
                                        fullSize
                                        error={errors.name ? _(g.requiredField, { field: _(t.ownerClaim) }) : undefined}
                                        id='name'
                                        marginBottom={false}
                                    >
                                        <FormInput
                                            inlineStyle
                                            align={inputAligns.RIGHT}
                                            placeholder={_(t.ownerClaim)}
                                            {...register('authorization.ownerClaim', { required: true, validate: (val) => val !== '' })}
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

TabContent1.displayName = 'TabContent1'

export default TabContent1
