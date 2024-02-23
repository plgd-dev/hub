import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { Inputs, Props } from './Step2.types'
import * as commonStyles from '../../LinkNewHubPage.styles'
import SubStepButtons from '../SubStepButtons'

const Step2: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, setStep } = useContext(FormContext)

    const {
        formState: { errors },
        register,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step2' })
    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.hubDetails)}</h1>
            <p css={[commonStyles.description, commonStyles.descriptionLarge]}>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna
            </p>

            <FormGroup id='id'>
                <FormLabel text={_(g.id)} />
                <FormInput readOnly={true} value={defaultFormData.id || ''} />
            </FormGroup>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                <FormLabel text={_(g.name)} />
                <FormInput
                    {...register('name', {
                        required: true,
                        validate: (val) => val !== '',
                    })}
                />
            </FormGroup>

            <FormGroup error={errors.name ? _(t.coapGateway, { field: _(t.coapGateway) }) : undefined} id='coapGateway' marginBottom={false}>
                <FormLabel text={_(t.coapGateway)} />
                <FormInput
                    {...register('coapGateway', {
                        required: true,
                        validate: (val) => val !== '',
                    })}
                />
            </FormGroup>

            <SubStepButtons onClickBack={() => setStep?.(0)} onClickNext={() => setStep?.(2)} />
        </form>
    )
}

Step2.displayName = 'Step2'

export default Step2
