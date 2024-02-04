import React, { FC, useContext } from 'react'

import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'

import { Props, Inputs } from './TabContent2.types'
import TlsPage from '../../../TlsPage'

const TabContent2: FC<Props> = (props) => {
    const { contentRefs, defaultFormData, loading } = props

    const { updateData, setFormError } = useContext(FormContext)
    const { watch, setValue, control } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'tab2Content2' })

    return <TlsPage contentRefs={contentRefs} control={control} loading={loading} prefix='certificateAuthority.grpc.' setValue={setValue} watch={watch} />
}

TabContent2.displayName = 'TabContent2'

export default TabContent2
