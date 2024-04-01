import React, { FC } from 'react'

import { useForm } from '@shared-ui/common/hooks'

import { Props, Inputs } from './TabContent2.types'
import TlsPage from '../../../TlsPage'

const TabContent2: FC<Props> = (props) => {
    const { contentRefs, defaultFormData, loading } = props

    const { watch, setValue, control } = useForm<Inputs>({ defaultFormData, errorKey: 'tab2Content2' })

    return <TlsPage contentRefs={contentRefs} control={control} loading={loading} prefix='certificateAuthority.grpc.' setValue={setValue} watch={watch} />
}

TabContent2.displayName = 'TabContent2'

export default TabContent2
