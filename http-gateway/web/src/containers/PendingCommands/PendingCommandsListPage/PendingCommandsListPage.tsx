import { useState } from 'react'
import { useIntl } from 'react-intl'

import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'

import PendingCommandsList from '../PendingCommandsList'

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
