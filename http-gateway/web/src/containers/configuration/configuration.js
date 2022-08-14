import { useIntl } from 'react-intl'

import { Layout } from '@shared-ui/components/old/layout'
import { messages as menuT } from '@shared-ui/components/old/menu/menu-i18n'

export const Configuration = () => {
  const { formatMessage: _ } = useIntl()

  return (
    <Layout
      title={_(menuT.notifications)}
      breadcrumbs={[
        {
          to: '/',
          label: _(menuT.dashboard),
        },
        {
          label: _(menuT.configuration),
        },
      ]}
    >
      <div />
    </Layout>
  )
}
