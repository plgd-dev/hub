import React, { FC, FormEvent, useCallback, useContext, useState } from 'react'
import { useIntl } from 'react-intl'
import merge from 'lodash/merge'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { openTelemetry } from '@shared-ui/common/services/opentelemetry'
import ButtonBox from '@shared-ui/components/Atomic/ButtonBox'
import * as commonStyles from '@shared-ui/components/Templates/FullPageWizard/FullPageWizardCommon.styles'

import { messages as t } from '../../../LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './Step1.types'
import { getAppWellKnownConfiguration } from '@/containers/App/AppRest'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/LinkedHubs/utils'

const Step1: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, setStep } = useContext(FormContext)

    const [loading, setLoading] = useState(false)

    const {
        formState: { errors },
        register,
        getValues,
        watch,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step1' })

    const name = watch('name')
    const endpoint = watch('endpoint')

    const handleFormSubmit = useCallback(
        (e: FormEvent) => {
            e.preventDefault()
            setLoading(true)

            const values = getValues()

            const fetchWellKnownConfig = async () => {
                try {
                    const { data: wellKnown } = await openTelemetry.withTelemetry(
                        () => getAppWellKnownConfiguration(values.endpoint),
                        'get-endpoint-hub-configuration'
                    )

                    updateData(
                        merge(DEFAULT_FORM_DATA, {
                            name: values.name,
                            endpoint: values.endpoint,
                            hubId: wellKnown.id,
                            certificateAuthority: {
                                grpc: {
                                    address: wellKnown.certificateAuthority,
                                },
                            },
                            authorization: {
                                ownerClaim: wellKnown.jwtOwnerClaim,
                                provider: {
                                    authority: wellKnown.authority,
                                },
                            },
                            gateways: [{ value: wellKnown.coapGateway }],
                        })
                    )

                    setStep?.(1)

                    setLoading(false)
                } catch (e) {
                    console.error(e)
                    setLoading(false)
                }
            }

            fetchWellKnownConfig().then()
        },
        [getValues, setStep, updateData]
    )

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.linkNewHub)}</h1>
            <p css={[commonStyles.description, commonStyles.descriptionLarge]}>
                The new linked hub offers a configuration interface for enrollment groups, enabling the onboarding of devices to the hub. Upon completing the setup, populate the "Endpoint" field with values retrieved from the hub.
            </p>
            <Row>
                <Column size={6}>
                    <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                        <FormLabel text={_(g.name)} />
                        <FormInput {...register('name', { required: true, validate: (val) => val !== '' })} />
                    </FormGroup>
                </Column>
                <Column size={6}>
                    <FormGroup error={errors.endpoint ? _(g.requiredField, { field: _(t.endpoint) }) : undefined} id='endpoint'>
                        <FormLabel text={_(t.endpoint)} tooltipMaxWidth={270} tooltipText={_(t.endpointDescription)} />
                        <FormInput {...register('endpoint', { required: true, validate: (val) => val !== '' })} />
                    </FormGroup>
                </Column>
            </Row>

            <ButtonBox
                disabled={name === '' || endpoint === '' || !name || !endpoint}
                htmlType='submit'
                loading={loading}
                loadingText={_(t.continue)}
                onClick={handleFormSubmit}
            >
                {_(t.continue)}
            </ButtonBox>
        </form>
    )
}

Step1.displayName = 'Step1'

export default Step1
