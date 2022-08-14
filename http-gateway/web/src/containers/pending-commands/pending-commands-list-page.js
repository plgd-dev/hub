import { useState } from 'react'
import { useIntl } from 'react-intl'

import { Layout } from '@shared-ui/components/old/layout'
import { messages as menuT } from '@shared-ui/components/old/menu/menu-i18n'

import { PendingCommandsList } from './_pending-commands-list'

export const PendingCommandsListPage = () => {
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
