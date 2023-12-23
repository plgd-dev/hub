import React, { useCallback, useState } from 'react'
import { useIntl } from 'react-intl'

import Footer from '@shared-ui/components/Layout/Footer'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Tabs from '@shared-ui/components/Atomic/Tabs'

import { messages as t } from './ConfigurationPage.i18n'

import Tab1 from './Tabs/Tab1'
import Tab2 from './Tabs/Tab2'

const ConfigurationPage = () => {
    const { formatMessage: _ } = useIntl()

    const [activeTabItem, setActiveTabItem] = useState(0)
    const [resetIndex, setResetIndex] = useState(0)

    const handleTabChange = useCallback((i: number) => {
        setActiveTabItem(i)
        setResetIndex((prev) => prev + 1)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    return (
        <PageLayout
            footer={<Footer paginationComponent={<div id='paginationPortalTarget'></div>} recentTasksPortal={<div id='recentTasksPortalTarget'></div>} />}
            title={_(t.configuration)}
        >
            <Tabs
                activeItem={activeTabItem}
                fullHeight={true}
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
