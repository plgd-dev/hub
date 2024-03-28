import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'

import Footer from '@shared-ui/components/Layout/Footer'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Tabs from '@shared-ui/components/Atomic/Tabs'

import { Props } from './ConfigurationPage.types'
import { messages as t } from './ConfigurationPage.i18n'
import Tab1 from './Tabs/Tab1'
import Tab2 from './Tabs/Tab2'
import { pages } from '@/routes'

const ConfigurationPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultActiveTab } = props

    const navigate = useNavigate()

    const [activeTabItem, setActiveTabItem] = useState(defaultActiveTab ?? 0)
    const [resetIndex, setResetIndex] = useState(0)

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)
        setResetIndex((prev) => prev + 1)

        navigate(generatePath(i === 0 ? pages.CONFIGURATION.LINK : pages.CONFIGURATION.THEME_GENERATOR))
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    return (
        <PageLayout
            footer={
                <Footer
                    innerPortalTarget={<div id='innerFooterPortalTarget'></div>}
                    paginationComponent={<div id='paginationPortalTarget'></div>}
                    recentTasksPortal={<div id='recentTasksPortalTarget'></div>}
                />
            }
            title={_(t.configuration)}
            xPadding={false}
        >
            <Tabs
                fullHeight
                innerPadding
                activeItem={activeTabItem}
                onItemChange={handleTabChange}
                tabs={[
                    {
                        name: _(t.general),
                        id: 0,
                        content: <Tab1 resetForm={resetIndex} />,
                    },
                    {
                        name: _(t.themeGenerator),
                        id: 1,
                        content: <Tab2 isTabActive={activeTabItem === 1} resetForm={resetIndex} />,
                    },
                ]}
            />
        </PageLayout>
    )
}

export default ConfigurationPage
