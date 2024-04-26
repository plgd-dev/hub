import React, { FC, lazy } from 'react'

import Spacer from '@shared-ui/components/Atomic/Spacer'

import { Props } from './Tab2.types'

const TabContent1 = lazy(() => import('./Contents/TabContent1'))
const TabContent2 = lazy(() => import('./Contents/TabContent2'))

const Tab2: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    return (
        <>
            <TabContent1 defaultFormData={defaultFormData} loading={loading} />
            <Spacer type='pt-8'>
                <TabContent2 defaultFormData={defaultFormData} loading={loading} />
            </Spacer>
        </>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2
