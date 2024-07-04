import React, { FC, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'
import { useForm, WellKnownConfigType } from '@shared-ui/common/hooks'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Button, { buttonVariants } from '@shared-ui/components/Atomic/Button'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { security } from '@shared-ui/common/services'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './Tab3.types'
import notificationId from '@/notificationId'
import { getOauthToken } from '@/containers/SnippetService/rest'

const Tab3: FC<Props> = (props) => {
    const { defaultFormData, resetIndex } = props

    const { formatMessage: _ } = useIntl()
    const schema = useValidationsSchema('tab3')

    const [loading, setLoading] = useState(false)

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const {
        formState: { errors },
        updateField,
        setValue,
        watch,
        reset,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab3',
        schema,
    })

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const handleLoadToken = async () => {
        setLoading(true)

        try {
            const accessToken = await getOauthToken()

            setValue('apiAccessToken', accessToken)
            updateField('apiAccessToken', accessToken)

            setLoading(false)
        } catch (error: any) {
            let e = error
            if (!(error instanceof Error)) {
                e = new Error(error)
            }

            Notification.error(
                { title: _(confT.conditionTokenError), message: e.message },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_CONDITIONS_DETAIL_GET_TOKEN_ERROR }
            )

            setLoading(false)
        }
    }

    const apiAccessToken = watch('apiAccessToken')

    return (
        <>
            <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
                <FormLabel text={_(confT.APIAccessToken)} />
                <FormTextarea
                    onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                    onChange={(e) => {
                        setValue('apiAccessToken', e.target.value)
                        updateField('apiAccessToken', e.target.value)
                    }}
                    style={{ height: 450 }}
                    value={apiAccessToken}
                />
            </FormGroup>
            {wellKnownConfig?.m2mOauthClient?.clientId && (
                <Spacer type='mt-3'>
                    <Button loading={loading} loadingText={_(g.loading)} onClick={handleLoadToken} variant={buttonVariants.SECONDARY}>
                        {_(confT.generateNewToken)}
                    </Button>
                </Spacer>
            )}
        </>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3
