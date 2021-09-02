import { useState } from 'react'
import { useIntl } from 'react-intl'

import { Layout } from '@/components/layout'
import { messages as menuT } from '@/components/menu/menu-i18n'

import { PendingCommandsList } from './_pending-commands-list'

export const PendingCommandsListPage = () => {
  const { formatMessage: _ } = useIntl()
  const [loading, setLoading] = useState(false)

  return (
    <Layout
      title={_(menuT.pendingCommands)}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
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
