import React, { FC, useEffect } from 'react'

import { useForm } from '@shared-ui/common/hooks'

import { Step2FormComponent } from '@/containers/SnippetService/Conditions/FomComponents'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './Tab2.types'
import { useConditionFilterValidation } from '@/containers/SnippetService/hooks'
import isFunction from 'lodash/isFunction'

const Tab2: FC<Props> = (props) => {
    const { defaultFormData, resetIndex, setFilterError } = props

    const schema = useValidationsSchema('tab2')

    const { updateField, setValue, reset, watch } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab2',
        schema,
    })

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const invalidFilters = useConditionFilterValidation({ watch })

    useEffect(() => {
        isFunction(setFilterError) && setFilterError(invalidFilters)
    }, [invalidFilters, setFilterError])

    return <Step2FormComponent isActivePage={true} setValue={setValue} updateField={updateField} watch={watch} />
}

Tab2.displayName = 'Tab2'

export default Tab2
