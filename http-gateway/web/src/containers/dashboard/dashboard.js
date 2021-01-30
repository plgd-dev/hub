import { useIntl } from 'react-intl'

import { Layout } from '@/components/layout'
import { messages as menuT } from '@/components/menu/menu-i18n'

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
