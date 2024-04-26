import React, { FC, lazy } from 'react'

import { Props } from './Tab3.types'
import Spacer from '@shared-ui/components/Atomic/Spacer'

const TabContent1 = lazy(() => import('./Contents/TabContent1'))
const TabContent2 = lazy(() => import('./Contents/TabContent2'))
const TabContent3 = lazy(() => import('./Contents/TabContent3'))
const TabContent4 = lazy(() => import('./Contents/TabContent4'))

const Tab3: FC<Props> = (props) => {
    const { defaultFormData, loading } = props

    return (
        <>
            <TabContent1 defaultFormData={defaultFormData} loading={loading} />
            <Spacer type='pt-8'>
                <TabContent2 defaultFormData={defaultFormData} loading={loading} />
            </Spacer>
            <Spacer type='pt-8'>
                <TabContent3 defaultFormData={defaultFormData} loading={loading} />
            </Spacer>
            <Spacer type='pt-8'>
                <TabContent4 defaultFormData={defaultFormData} loading={loading} />
            </Spacer>
        </>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3
