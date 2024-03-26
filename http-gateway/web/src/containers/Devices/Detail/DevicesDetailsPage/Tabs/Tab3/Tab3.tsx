import React, { FC, useEffect } from 'react'

import Spacer from '@shared-ui/components/Atomic/Spacer'

import { Props } from './Tab3.types'
import SingleCert from './SingleCert'
import MultiCerts from './MultiCerts'

const Tab3: FC<Props> = (props) => {
    const { certificates, loading, handleTabChange, refresh } = props

    useEffect(() => {
        if (certificates?.length === 0) {
            handleTabChange(0)
        }
    }, [certificates, handleTabChange])

    if (!certificates || certificates.length === 0) {
        return <div />
    }

    if (certificates?.length === 1) {
        return <SingleCert certificate={certificates[0]} handleTabChange={handleTabChange} loading={loading} refresh={refresh} />
    } else {
        return (
            <Spacer style={{ height: '100%' }} type='px-10'>
                <MultiCerts certificates={certificates} loading={loading} refresh={refresh} />
            </Spacer>
        )
    }
}

Tab3.displayName = 'Tab3'

export default Tab3
