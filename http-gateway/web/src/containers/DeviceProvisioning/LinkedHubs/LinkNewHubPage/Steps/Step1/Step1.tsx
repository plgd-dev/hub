import React, { FC, FormEvent, FormEventHandler, useCallback, useContext, useState } from 'react'
import { useIntl } from 'react-intl'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import Button from '@shared-ui/components/Atomic/Button'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { openTelemetry } from '@shared-ui/common/services/opentelemetry'

import { messages as t } from '../../../LinkedHubs.i18n'
import * as styles from './Step1.styles'
import { messages as g } from '@/containers/Global.i18n'
import { Props, Inputs } from './Step1.types'
import { getAppWellKnownConfiguration } from '@/containers/App/AppRest'

const Step1: FC<Props> = (props) => {
    const { defaultFormData, presetData } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError } = useContext(FormContext)

    const [loading, setLoading] = useState(false)

    const {
        formState: { errors },
        register,
        getValues,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step1' })

    const handleFormSubmit = useCallback((e: FormEvent) => {
        e.preventDefault()
        setLoading(true)

        const values = getValues()

        const fetchWellKnownConfig = async () => {
            try {
                const { data: wellKnown } = await openTelemetry.withTelemetry(
                    () => getAppWellKnownConfiguration(values.endpoint),
                    'get-endpoint-hub-configuration'
                )

                console.log(wellKnown)

                presetData({
                    certificateAuthorities: wellKnown.certificateAuthorities,
                })

                setLoading(false)
            } catch (e) {
                console.error(e)
                setLoading(false)
            }
        }

        fetchWellKnownConfig().then()
    }, [])

    return (
        <form>
            <Row>
                <Column lg={3} size={3} sm={2}></Column>
                <Column size={6}>
                    <Row>
                        <Column size={6}>
                            <FormGroup error={errors.hubName ? _(g.requiredField, { field: _(t.hubName) }) : undefined} id='hubName'>
                                <FormLabel text={_(t.hubName)} />
                                <FormInput {...register('hubName', { required: true, validate: (val) => val !== '' })} />
                            </FormGroup>
                        </Column>
                        <Column size={6}>
                            <FormGroup error={errors.endpoint ? _(g.requiredField, { field: _(t.endpoint) }) : undefined} id='endpoint'>
                                <FormLabel text={_(t.endpoint)} tooltipMaxWidth={270} tooltipText={_(t.endpointDescription)} />
                                <FormInput {...register('endpoint', { required: true, validate: (val) => val !== '' })} />
                            </FormGroup>
                        </Column>
                    </Row>
                    <div css={styles.loadingButtonWrapper}>
                        <Button
                            css={styles.continueBtn}
                            htmlType='submit'
                            loading={loading}
                            loadingText={_(t.continue)}
                            onClick={handleFormSubmit}
                            size='big'
                            variant='primary'
                        >
                            {_(t.continue)}
                        </Button>
                    </div>
                </Column>
                <Column size={3}></Column>
            </Row>
        </form>
    )
}

Step1.displayName = 'Step1'

export default Step1
