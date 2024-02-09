import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import { useForm } from '@shared-ui/common/hooks'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { Props, Inputs } from './Tab1.types'
import { messages as g } from '../../../../../Global.i18n'
import { messages as t } from '../../../LinkedHubs.i18n'

const Tab1: FC<Props> = (props) => {
    const { defaultFormData } = props
    const { formatMessage: _ } = useIntl()
    const { hubId } = useParams()

    const { updateData, setFormError, commonFormGroupProps, commonInputProps } = useContext(FormContext)

    const {
        formState: { errors },
        register,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'tab1' })

    return (
        <form>
            <SimpleStripTable
                leftColSize={7}
                rightColSize={5}
                rows={[
                    {
                        attribute: _(g.id),
                        value: <FormInput {...commonInputProps} disabled value={hubId} />,
                    },
                    {
                        attribute: _(g.name),
                        value: (
                            <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                                <FormInput
                                    {...commonInputProps}
                                    placeholder={_(g.name)}
                                    {...register('name', { required: true, validate: (val) => val !== '' })}
                                />
                            </FormGroup>
                        ),
                    },
                    {
                        attribute: _(t.coapGateway),
                        value: (
                            <FormGroup
                                {...commonFormGroupProps}
                                error={errors.coapGateway ? _(g.requiredField, { field: _(t.coapGateway) }) : undefined}
                                id='coapGateway'
                            >
                                <FormInput
                                    {...commonInputProps}
                                    placeholder={_(t.coapGateway)}
                                    {...register('coapGateway', { required: true, validate: (val) => val !== '' })}
                                />
                            </FormGroup>
                        ),
                    },
                ]}
            />
        </form>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
