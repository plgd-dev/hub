import { FC, useEffect } from 'react'
import { useLocation } from 'react-router'

import LeftPanel, { parseActiveItem } from '@shared-ui/components/new/Layout/LeftPanel'
import { Props } from '@shared-ui/components/new/Layout/LeftPanel/LeftPanel.types'
import { mather } from '@/routes'

type LeftPanelWrapperType = {
    onLocationChange: (id: string) => void
} & Props

const LeftPanelWrapper: FC<LeftPanelWrapperType> = (props) => {
    const { onLocationChange, ...rest } = props
    const location = useLocation()

    useEffect(() => {
        onLocationChange(parseActiveItem(location.pathname, props.menu!, mather))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [location.pathname])
    return <LeftPanel {...rest} />
}

LeftPanelWrapper.displayName = 'LeftPanelWrapper'

export default LeftPanelWrapper
