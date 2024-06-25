import React, { FC, useEffect } from 'react'
import { useIntl } from 'react-intl'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'
import { useForm } from '@shared-ui/common/hooks'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './Tab3.types'

const Tab3: FC<Props> = (props) => {
    const { defaultFormData, resetIndex } = props

    const { formatMessage: _ } = useIntl()
    const schema = useValidationsSchema('tab3')

    const {
        formState: { errors },
        updateField,
        register,
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

    return (
        <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
            <FormLabel text={_(confT.APIAccessToken)} />
            <FormTextarea
                {...register('apiAccessToken', { required: true, validate: (val) => val !== '' })}
                onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                style={{ height: 450 }}
            />
        </FormGroup>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3
