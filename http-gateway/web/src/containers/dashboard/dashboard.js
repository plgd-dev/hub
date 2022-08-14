import { useIntl } from 'react-intl'

import { Layout } from '@shared-ui/components/old/layout'
import { messages as menuT } from '@shared-ui/components/old/menu/menu-i18n'

export const Dashboard = () => {
  const { formatMessage: _ } = useIntl()

  return (
    <Layout
      title={_(menuT.dashboard)}
      breadcrumbs={[
        {
          label: _(menuT.dashboard),
        },
      ]}
    >
      <div />
    </Layout>
  )
}
