import React, { FC } from 'react'

import { useForm } from '@shared-ui/common/hooks'

import { Props, Inputs } from './TabContent3.types'
import TlsPage from '@/containers/DeviceProvisioning/LinkedHubs/DetailPage/TlsPage'

const TabContent3: FC<Props> = (props) => {
    const { contentRefs, defaultFormData, loading } = props

    const { control, watch, setValue } = useForm<Inputs>({ defaultFormData, errorKey: 'tab3Content3' })

    return <TlsPage contentRefs={contentRefs} control={control} loading={loading} prefix='certificateAuthority.grpc.' setValue={setValue} watch={watch} />
}

TabContent3.displayName = 'TabContent3'

export default TabContent3
