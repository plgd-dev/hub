import { useState } from 'react'
import { useIntl } from 'react-intl'

import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'

import PendingCommandsList from '../PendingCommandsList'
import PageLayout from '@shared-ui/components/new/PageLayout'

const PendingCommandsListPage = () => {
    const { formatMessage: _ } = useIntl()
    const [loading, setLoading] = useState(false)

    return (
        <PageLayout
            breadcrumbs={[
                {
                    label: _(menuT.pendingCommands),
                },
            ]}
            loading={loading}
            title={_(menuT.pendingCommands)}
        >
            <PendingCommandsList onLoading={setLoading} />
        </PageLayout>
    )
}

PendingCommandsListPage.displayName = 'PendingCommandsListPage'

export default PendingCommandsListPage
