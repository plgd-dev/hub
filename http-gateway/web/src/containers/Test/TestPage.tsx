import React from 'react'
import Footer from '@shared-ui/components/Layout/Footer'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

const TestPage = () => {
    return (
        <PageLayout
            footer={<Footer paginationComponent={<div id='paginationPortalTarget'></div>} recentTasksPortal={<div id='recentTasksPortalTarget'></div>} />}
            title='Test'
        >
            <hr />
            <h4>Notifications</h4>
            <div>
                <button onClick={() => Notification.info({ title: 'Info', message: 'Info message' })}>Info</button>
                <button onClick={() => Notification.success({ title: 'Success', message: 'Success message' })}>Success</button>
                <button onClick={() => Notification.warning({ title: 'Warning', message: 'Warning message' })}>Warning</button>
                <button onClick={() => Notification.error({ title: 'Error', message: 'Error message' })}>Error</button>
            </div>
        </PageLayout>
    )
}

export default TestPage
