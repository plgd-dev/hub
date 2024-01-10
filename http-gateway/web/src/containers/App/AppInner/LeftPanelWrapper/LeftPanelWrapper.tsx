import { FC, useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import LeftPanel, { getFirstActiveItemFromMenu, parseActiveItem } from '@shared-ui/components/Layout/LeftPanel'
import { Props } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'

import { mather } from '@/routes'

type LeftPanelWrapperType = {
    onLocationChange: (id: string) => void
} & Props

const LeftPanelWrapper: FC<LeftPanelWrapperType> = (props) => {
    const { onLocationChange, ...rest } = props
    const location = useLocation()
    const navigate = useNavigate()

    // Redirect to first active page from menu
    useEffect(() => {
        if (props.menu) {
            const firstActivePage: any = getFirstActiveItemFromMenu(props.menu)

            const activeId = parseActiveItem(location.pathname, props.menu, mather)

            if (firstActivePage && firstActivePage?.id !== activeId) {
                navigate(firstActivePage.link)
            }
        }
    }, [location, navigate, props.menu])

    useEffect(() => {
        onLocationChange(parseActiveItem(location.pathname, props.menu!, mather))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [location.pathname])

    return <LeftPanel {...rest} />
}

LeftPanelWrapper.displayName = 'LeftPanelWrapper'

export default LeftPanelWrapper
