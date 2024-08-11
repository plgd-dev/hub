import React from 'react'

import Footer from '@shared-ui/components/Layout/Footer'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'

const TestPage = () => {
    return (
        <PageLayout
            footer={<Footer paginationComponent={<div id='paginationPortalTarget'></div>} recentTasksPortal={<div id='recentTasksPortalTarget'></div>} />}
            title='Test'
        ></PageLayout>
    )
}

export default TestPage
