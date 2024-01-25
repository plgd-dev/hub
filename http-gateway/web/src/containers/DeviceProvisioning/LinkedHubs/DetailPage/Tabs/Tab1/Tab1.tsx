import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { useForm } from 'react-hook-form'
import { useParams } from 'react-router-dom'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput, { inputAligns } from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { Props, Inputs } from './Tab1.types'
import { messages as g } from '../../../../../Global.i18n'
import { messages as t } from '../../../LinkedHubs.i18n'

const Tab1: FC<Props> = (props) => {
    const { defaultFormData } = props
    const { formatMessage: _ } = useIntl()
    const { hubId } = useParams()

    const {
        formState: { errors },
        register,
        handleSubmit,
    } = useForm<Inputs>({ mode: 'all', reValidateMode: 'onSubmit', values: defaultFormData })

    const { onSubmit } = useContext(FormContext)

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <SimpleStripTable
                rows={[
                    {
                        attribute: _(g.id),
                        value: <FormInput disabled inlineStyle align={inputAligns.RIGHT} value={hubId} />,
                    },
                    {
                        attribute: _(g.name),
                        value: (
                            <FormGroup
                                errorTooltip
                                fullSize
                                error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined}
                                id='name'
                                marginBottom={false}
                            >
                                <FormInput
                                    inlineStyle
                                    align={inputAligns.RIGHT}
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
                                errorTooltip
                                fullSize
                                error={errors.coapGateway ? _(g.requiredField, { field: _(t.coapGateway) }) : undefined}
                                id='coapGateway'
                                marginBottom={false}
                            >
                                <FormInput
                                    inlineStyle
                                    align={inputAligns.RIGHT}
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
