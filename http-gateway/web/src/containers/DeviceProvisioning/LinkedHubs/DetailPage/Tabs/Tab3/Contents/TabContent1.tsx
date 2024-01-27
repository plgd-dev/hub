import React, { FC, useContext, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'
import cloneDeep from 'lodash/cloneDeep'

import Headline from '@shared-ui/components/Atomic/Headline'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { setProperty } from '@shared-ui/components/Atomic/_utils/utils'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './TabContent1.types'

const TabContent1: FC<Props> = (props) => {
    const { defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()

    const {
        formState: { errors, isDirty },
        register,
        watch,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const { updateData, setFormError, commonFormGroupProps, commonInputProps } = useContext(FormContext)

    const ownerClaim = watch('authorization.ownerClaim')

    useEffect(() => {
        if (defaultFormData && isDirty) {
            const copy = cloneDeep(defaultFormData)

            if (defaultFormData.authorization.ownerClaim !== ownerClaim) {
                updateData(setProperty(copy, 'authorization.ownerClaim', ownerClaim))
            }
        }
    }, [defaultFormData, isDirty, ownerClaim, updateData])

    useEffect(() => {
        isFunction(setFormError) && setFormError((prevState: any) => ({ ...prevState, tab3Content1: Object.keys(errors).length > 0 }))
    }, [errors, setFormError])

    return (
        <form>
            <Headline type='h5'>{_(t.general)}</Headline>
            <Spacer type='pt-4'>
                <Loadable condition={!loading}>
                    <SimpleStripTable
                        rows={[
                            {
                                attribute: _(t.ownerClaim),
                                value: (
                                    <FormGroup
                                        {...commonFormGroupProps}
                                        error={errors.authorization?.ownerClaim ? _(g.requiredField, { field: _(t.ownerClaim) }) : undefined}
                                        id='authorization?.ownerClaim'
                                    >
                                        <FormInput
                                            {...commonInputProps}
                                            {...register('authorization.ownerClaim', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                            placeholder={_(t.ownerClaim)}
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

TabContent1.displayName = 'TabContent1'

export default TabContent1
