import { useState } from 'react'
import { useIntl } from 'react-intl'

import Layout from '@shared-ui/components/new/Layout'
import { messages as menuT } from '@shared-ui/components/new/Menu/Menu.i18n'

import PendingCommandsList from '../PendingCommandsList'

const PendingCommandsListPage = () => {
  const { formatMessage: _ } = useIntl()
  const [loading, setLoading] = useState(false)

  return (
    <Layout
      title={_(menuT.pendingCommands)}
      breadcrumbs={[
        {
          label: _(menuT.pendingCommands),
        },
      ]}
      loading={loading}
    >
      <PendingCommandsList onLoading={setLoading} />
    </Layout>
  )
}

PendingCommandsListPage.displayName = 'PendingCommandsListPage'

export default PendingCommandsListPage
